package engine

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/zhiyunliu/distributed-workflow/interfaces"
	"github.com/zhiyunliu/distributed-workflow/types"
)

// auditLogManagerImpl 审计日志管理器，使用带缓冲 channel 异步批量写入
type auditLogManagerImpl struct {
	repo          interfaces.WorkflowRepository
	ch            chan *types.WorkflowAuditLog
	stopCh        chan struct{}
	doneCh        chan struct{}
	batchSize     int
	flushInterval time.Duration
}

// NewAuditLogManager 创建审计日志管理器
func NewAuditLogManager(repo interfaces.WorkflowRepository) interfaces.AuditLogManager {
	return &auditLogManagerImpl{
		repo:          repo,
		ch:            make(chan *types.WorkflowAuditLog, 2048),
		stopCh:        make(chan struct{}),
		doneCh:        make(chan struct{}),
		batchSize:     50,
		flushInterval: 100 * time.Millisecond,
	}
}

// Start 启动异步写入协程
func (m *auditLogManagerImpl) Start() {
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

// ─────────────────────────────────────────────────────────────────────────────
// auditLogServiceImpl
// ─────────────────────────────────────────────────────────────────────────────

type auditLogServiceImpl struct {
	mgr interfaces.AuditLogManager
}

// NewAuditLogService 创建审计日志服务
func NewAuditLogService(mgr interfaces.AuditLogManager) interfaces.AuditLogService {
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
