package engine

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	api "github.com/zhiyunliu/distributed-workflow/runtime-execution/pkg/api"
	"github.com/zhiyunliu/distributed-workflow/runtime-execution/internal/common/types"
)

// auditLogManagerImpl 审计日志管理器，使用带缓冲 channel 异步批量写入
type auditLogManagerImpl struct {
	repo          api.WorkflowRepository
	ch            chan *types.WorkflowAuditLog
	stopCh        chan struct{}
	doneCh        chan struct{}
	batchSize     int
	flushInterval time.Duration
	// currentHash 当前哈希链末端的哈希值，由 flush goroutine 独占访问，无需加锁
	currentHash string
}

// hashLogEntry 计算单条审计日志的防篡改哈希值
// 哈希输入：上一条日志哈希 + 当前日志关键字段（用 | 分隔）
func hashLogEntry(prevHash string, entry *types.WorkflowAuditLog) string {
	h := sha256.New()
	h.Write([]byte(prevHash))
	content := fmt.Sprintf("%s|%s|%s|%s|%s|%s|%s",
		entry.ID, string(entry.OperationType), entry.Operator,
		entry.WorkflowID, entry.InstanceID, entry.NodeID,
		entry.OperateTime.UTC().Format(time.RFC3339))
	h.Write([]byte(content))
	h.Write([]byte(entry.Detail))
	return hex.EncodeToString(h.Sum(nil))
}

// NewAuditLogManager 创建审计日志管理器
func NewAuditLogManager(repo api.WorkflowRepository) api.AuditLogManager {
	return &auditLogManagerImpl{
		repo:          repo,
		ch:            make(chan *types.WorkflowAuditLog, 2048),
		stopCh:        make(chan struct{}),
		doneCh:        make(chan struct{}),
		batchSize:     50,
		flushInterval: 100 * time.Millisecond,
	}
}

// Start 启动异步写入协程，并从数据库加载最新哈希以初始化链
func (m *auditLogManagerImpl) Start() {
	latest, err := m.repo.GetLatestAuditLog()
	if err != nil {
		log.Warn().Err(err).Msg("audit log manager: failed to load latest hash, starting from genesis")
	} else if latest != nil {
		m.currentHash = latest.Hash
	}
	go m.runFlushLoop()
}

// Stop 停止并 flush 剩余日志
func (m *auditLogManagerImpl) Stop() {
	close(m.stopCh)
	<-m.doneCh
}

// RecordLog 异步记录单条审计日志
func (m *auditLogManagerImpl) RecordLog(auditLog *types.WorkflowAuditLog) error {
	if auditLog.ID == "" {
		auditLog.ID = uuid.New().String()
	}
	if auditLog.OperateTime.IsZero() {
		auditLog.OperateTime = time.Now().UTC()
	}
	select {
	case m.ch <- auditLog:
	default:
		// channel 满时降级为同步写入，避免阻塞
		return m.repo.CreateAuditLog(auditLog)
	}
	return nil
}

// BatchRecordLog 异步批量记录审计日志
func (m *auditLogManagerImpl) BatchRecordLog(logs []*types.WorkflowAuditLog) error {
	for _, l := range logs {
		if err := m.RecordLog(l); err != nil {
			return err
		}
	}
	return nil
}

// QueryLogs 分页查询审计日志
func (m *auditLogManagerImpl) QueryLogs(filter types.AuditLogFilter, page, pageSize int) ([]*types.WorkflowAuditLog, int64, error) {
	return m.repo.QueryAuditLogs(filter, page, pageSize)
}

// GetLogByID 获取单条审计日志
func (m *auditLogManagerImpl) GetLogByID(logID string) (*types.WorkflowAuditLog, error) {
	return m.repo.GetAuditLog(logID)
}

// ArchiveLogs 归档历史审计日志
func (m *auditLogManagerImpl) ArchiveLogs(beforeTime time.Time) error {
	return m.repo.ArchiveAuditLogs(beforeTime)
}

// runFlushLoop 批量 flush 循环
func (m *auditLogManagerImpl) runFlushLoop() {
	defer close(m.doneCh)

	ticker := time.NewTicker(m.flushInterval)
	defer ticker.Stop()

	batch := make([]*types.WorkflowAuditLog, 0, m.batchSize)

	flush := func() {
		if len(batch) == 0 {
			return
		}
		// 计算哈希链：逐条计算，prevHash = 上一条日志的 hash
		prevHash := m.currentHash
		for _, entry := range batch {
			entry.Hash = hashLogEntry(prevHash, entry)
			prevHash = entry.Hash
		}
		m.currentHash = prevHash

		if err := m.repo.BatchCreateAuditLogs(batch); err != nil {
			log.Error().Err(err).Msg("audit log batch flush failed")
		}
		batch = batch[:0]
	}

	for {
		select {
		case l := <-m.ch:
			batch = append(batch, l)
			if len(batch) >= m.batchSize {
				flush()
			}
		case <-ticker.C:
			flush()
		case <-m.stopCh:
			// drain channel
			for {
				select {
				case l := <-m.ch:
					batch = append(batch, l)
				default:
					flush()
					return
				}
			}
		}
	}
}

// VerifyLogChain 验证指定时间范围内日志链的完整性
// 对范围内按时间升序排列的相邻日志逐对校验哈希，任何不匹配则返回 false
// 最大支持 10000 条日志，超出则返回错误
func (m *auditLogManagerImpl) VerifyLogChain(ctx context.Context, startTime, endTime time.Time) (bool, error) {
	const maxVerifySize = 10000
	filter := types.AuditLogFilter{
		StartTime: &startTime,
		EndTime:   &endTime,
	}
	logs, total, err := m.repo.QueryAuditLogs(filter, 1, maxVerifySize)
	if err != nil {
		return false, fmt.Errorf("VerifyLogChain: query failed: %w", err)
	}
	if total > maxVerifySize {
		return false, fmt.Errorf("VerifyLogChain: range contains %d logs, exceeds max verify size %d", total, maxVerifySize)
	}
	if len(logs) <= 1 {
		return true, nil
	}

	// 按 OperateTime ASC，同时刻按 ID 字典序保证稳定排序
	sort.Slice(logs, func(i, j int) bool {
		if logs[i].OperateTime.Equal(logs[j].OperateTime) {
			return logs[i].ID < logs[j].ID
		}
		return logs[i].OperateTime.Before(logs[j].OperateTime)
	})

	// 逐对验证：log[i].Hash == hashLogEntry(log[i-1].Hash, log[i])
	for i := 1; i < len(logs); i++ {
		select {
		case <-ctx.Done():
			return false, ctx.Err()
		default:
		}
		expected := hashLogEntry(logs[i-1].Hash, logs[i])
		if logs[i].Hash != expected {
			return false, nil
		}
	}
	return true, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// auditLogServiceImpl
// ─────────────────────────────────────────────────────────────────────────────

type auditLogServiceImpl struct {
	mgr api.AuditLogManager
}

// NewAuditLogService 创建审计日志服务
func NewAuditLogService(mgr api.AuditLogManager) api.AuditLogService {
	return &auditLogServiceImpl{mgr: mgr}
}

func (s *auditLogServiceImpl) Record(auditLog *types.WorkflowAuditLog) error {
	return s.mgr.RecordLog(auditLog)
}

func (s *auditLogServiceImpl) QueryLogs(filter types.AuditLogFilter, page, pageSize int) ([]*types.WorkflowAuditLog, int64, error) {
	return s.mgr.QueryLogs(filter, page, pageSize)
}

func (s *auditLogServiceImpl) ArchiveLogs(beforeTime time.Time) error {
	return s.mgr.ArchiveLogs(beforeTime)
}

// ─────────────────────────────────────────────────────────────────────────────
// 内部辅助：构建审计日志
// ─────────────────────────────────────────────────────────────────────────────

// NewAuditLog 构建审计日志条目
func NewAuditLog(opType types.AuditOperationType, operator, operateIP string, opts ...func(*types.WorkflowAuditLog)) *types.WorkflowAuditLog {
	l := &types.WorkflowAuditLog{
		ID:            uuid.New().String(),
		OperationType: opType,
		Operator:      operator,
		OperateIP:     operateIP,
		OperateTime:   time.Now().UTC(),
	}
	for _, opt := range opts {
		opt(l)
	}
	return l
}

// WithInstance 设置实例 ID
func WithInstance(instanceID string) func(*types.WorkflowAuditLog) {
	return func(l *types.WorkflowAuditLog) { l.InstanceID = instanceID }
}

// WithWorkflow 设置流程 ID
func WithWorkflow(workflowID string) func(*types.WorkflowAuditLog) {
	return func(l *types.WorkflowAuditLog) { l.WorkflowID = workflowID }
}

// WithNode 设置节点 ID
func WithNode(nodeID string) func(*types.WorkflowAuditLog) {
	return func(l *types.WorkflowAuditLog) { l.NodeID = nodeID }
}

// WithEndpoint 设置端点 ID
func WithEndpoint(endpointID string) func(*types.WorkflowAuditLog) {
	return func(l *types.WorkflowAuditLog) { l.EndpointID = endpointID }
}

// WithDetail 设置详情描述
func WithDetail(detail string) func(*types.WorkflowAuditLog) {
	return func(l *types.WorkflowAuditLog) { l.Detail = detail }
}

// WithDetailf 设置格式化详情描述
func WithDetailf(format string, args ...interface{}) func(*types.WorkflowAuditLog) {
	return func(l *types.WorkflowAuditLog) { l.Detail = fmt.Sprintf(format, args...) }
}

// WithAfterData 设置变更后数据
func WithAfterData(data map[string]interface{}) func(*types.WorkflowAuditLog) {
	return func(l *types.WorkflowAuditLog) { l.AfterData = data }
}
