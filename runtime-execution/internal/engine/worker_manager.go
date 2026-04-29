package engine

import (
	"fmt"
	"sync"

	"github.com/rs/zerolog/log"
	api "github.com/zhiyunliu/distributed-workflow/runtime-execution/pkg/api"
	"github.com/zhiyunliu/distributed-workflow/runtime-execution/internal/common/rpc"
	"github.com/zhiyunliu/distributed-workflow/runtime-execution/pkg/types"
)

// WorkerManagerServiceImpl 实现 api.WorkerManagerService
// 使用 sync.Map 存储在线 Worker，支持加权最小负载分配
type WorkerManagerServiceImpl struct {
	workers    sync.Map // map[workerID]*types.NodeWorkerInfo
	workerPool *rpc.WorkerClient
}

// NewWorkerManagerService 创建 WorkerManagerService 实例
func NewWorkerManagerService(pool *rpc.WorkerClient) api.WorkerManagerService {
	return &WorkerManagerServiceImpl{workerPool: pool}
}

func (m *WorkerManagerServiceImpl) RegisterWorker(info *types.NodeWorkerInfo) error {
	info.Status = types.WorkerStatusOnline
	m.workers.Store(info.ID, info)
	log.Info().Str("worker_id", info.ID).Str("address", info.Address).Msg("worker registered")
	return nil
}

func (m *WorkerManagerServiceImpl) UpdateHeartbeat(workerID string, currentLoad map[string]int) error {
	val, ok := m.workers.Load(workerID)
	if !ok {
		return fmt.Errorf("worker '%s' not found", workerID)
	}
	w := val.(*types.NodeWorkerInfo)
	w.CurrentLoad = currentLoad
	return nil
}

func (m *WorkerManagerServiceImpl) GetOnlineWorkers() []*types.NodeWorkerInfo {
	var list []*types.NodeWorkerInfo
	m.workers.Range(func(_, val interface{}) bool {
		w := val.(*types.NodeWorkerInfo)
		if w.Status == types.WorkerStatusOnline {
			list = append(list, w)
		}
		return true
	})
	return list
}

func (m *WorkerManagerServiceImpl) GetWorkersByNodeType(nodeType string) []*types.NodeWorkerInfo {
	var list []*types.NodeWorkerInfo
	m.workers.Range(func(_, val interface{}) bool {
		w := val.(*types.NodeWorkerInfo)
		if w.Status != types.WorkerStatusOnline {
			return true
		}
		if _, ok := w.Capabilities[nodeType]; ok {
			list = append(list, w)
		}
		return true
	})
	return list
}

// SelectBestWorker 加权最小负载算法：负载率 = 当前执行数 / 最大并发数
func (m *WorkerManagerServiceImpl) SelectBestWorker(nodeType string) (*types.NodeWorkerInfo, error) {
	candidates := m.GetWorkersByNodeType(nodeType)
	if len(candidates) == 0 {
		return nil, fmt.Errorf("no available worker for node type '%s'", nodeType)
	}

	var best *types.NodeWorkerInfo
	bestRatio := float64(2) // 超过 1.0 表示无容量
	for _, w := range candidates {
		maxCap := w.Capabilities[nodeType]
		if maxCap <= 0 {
			continue
		}
		cur := w.CurrentLoad[nodeType]
		ratio := float64(cur) / float64(maxCap)
		if ratio < bestRatio {
			bestRatio = ratio
			best = w
		}
	}
	if best == nil {
		return nil, fmt.Errorf("all workers for node type '%s' are at full capacity", nodeType)
	}
	return best, nil
}

// AssignTask 向指定 Worker 发送任务
func (m *WorkerManagerServiceImpl) AssignTask(workerID string, task *types.Task) error {
	val, ok := m.workers.Load(workerID)
	if !ok {
		return fmt.Errorf("worker '%s' not found", workerID)
	}
	w := val.(*types.NodeWorkerInfo)
	if err := m.workerPool.AssignTask(w.Address, task); err != nil {
		return fmt.Errorf("assign task to worker '%s': %w", workerID, err)
	}
	// 乐观更新本地负载计数
	if w.CurrentLoad == nil {
		w.CurrentLoad = make(map[string]int)
	}
	w.CurrentLoad[task.NodeType]++
	return nil
}

// HandleWorkerOffline 处理 Worker 下线：将其标记为 offline，
// 已分配给该 Worker 但尚未完成的节点状态重置为 pending 等待重新调度
func (m *WorkerManagerServiceImpl) HandleWorkerOffline(workerID string) error {
	val, ok := m.workers.Load(workerID)
	if !ok {
		return nil
	}
	w := val.(*types.NodeWorkerInfo)
	w.Status = types.WorkerStatusOffline
	log.Warn().Str("worker_id", workerID).Msg("worker marked offline")
	return nil
}

func (m *WorkerManagerServiceImpl) UpdateWorkerLoad(workerID string, nodeType string, delta int) error {
	val, ok := m.workers.Load(workerID)
	if !ok {
		return fmt.Errorf("worker '%s' not found", workerID)
	}
	w := val.(*types.NodeWorkerInfo)
	if w.CurrentLoad == nil {
		w.CurrentLoad = make(map[string]int)
	}
	w.CurrentLoad[nodeType] += delta
	if w.CurrentLoad[nodeType] < 0 {
		w.CurrentLoad[nodeType] = 0
	}
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// D2 新增方法
// ─────────────────────────────────────────────────────────────────────────────

// GetAllWorkers 获取所有 Worker（包括 offline）
func (m *WorkerManagerServiceImpl) GetAllWorkers() []*types.NodeWorkerInfo {
	var list []*types.NodeWorkerInfo
	m.workers.Range(func(_, val interface{}) bool {
		list = append(list, val.(*types.NodeWorkerInfo))
		return true
	})
	return list
}

// GetWorkerInfo 获取单个 Worker 信息
func (m *WorkerManagerServiceImpl) GetWorkerInfo(workerID string) (*types.NodeWorkerInfo, error) {
	val, ok := m.workers.Load(workerID)
	if !ok {
		return nil, fmt.Errorf("worker '%s' not found", workerID)
	}
	return val.(*types.NodeWorkerInfo), nil
}

// UpdateWorkerInfo 更新 Worker 信息（健康状态、失联计数等）
func (m *WorkerManagerServiceImpl) UpdateWorkerInfo(info *types.NodeWorkerInfo) error {
	m.workers.Store(info.ID, info)
	return nil
}

// CheckWorkerHealth 主动对指定 Worker 发起健康检查
func (m *WorkerManagerServiceImpl) CheckWorkerHealth(workerID string) error {
	val, ok := m.workers.Load(workerID)
	if !ok {
		return fmt.Errorf("worker '%s' not found", workerID)
	}
	w := val.(*types.NodeWorkerInfo)
	if w.Status == types.WorkerStatusOffline {
		return fmt.Errorf("worker '%s' is offline", workerID)
	}
	healthy := m.workerPool.HealthCheck(w.Address, workerID)
	if !healthy {
		w.FailedHeartbeatCount++
		if w.FailedHeartbeatCount >= offlineThreshold {
			w.HealthStatus = types.WorkerHealthStatusUnhealthy
			w.Status = types.WorkerStatusOffline
		} else if w.FailedHeartbeatCount >= unhealthyThreshold {
			w.HealthStatus = types.WorkerHealthStatusUnhealthy
		}
		m.workers.Store(workerID, w)
		return fmt.Errorf("worker '%s' health check failed (%d consecutive failures)", workerID, w.FailedHeartbeatCount)
	}
	w.HealthStatus = types.WorkerHealthStatusHealthy
	w.FailedHeartbeatCount = 0
	m.workers.Store(workerID, w)
	return nil
}

// GetUnhealthyWorkers 获取所有不健康的 Worker
func (m *WorkerManagerServiceImpl) GetUnhealthyWorkers() []*types.NodeWorkerInfo {
	var list []*types.NodeWorkerInfo
	m.workers.Range(func(_, val interface{}) bool {
		w := val.(*types.NodeWorkerInfo)
		if w.HealthStatus == types.WorkerHealthStatusUnhealthy {
			list = append(list, w)
		}
		return true
	})
	return list
}

// FailoverWorker 触发 Worker 故障转移（标记 offline + 重置任务）
func (m *WorkerManagerServiceImpl) FailoverWorker(workerID string) error {
	val, ok := m.workers.Load(workerID)
	if !ok {
		return fmt.Errorf("worker '%s' not found", workerID)
	}
	w := val.(*types.NodeWorkerInfo)
	w.Status = types.WorkerStatusOffline
	m.workers.Store(workerID, w)
	log.Warn().Str("worker_id", workerID).Msg("worker failover triggered")
	return nil
}

// CleanupOrphanTasks 清理孤儿任务（Worker 已离线但节点仍处于 running 状态）
func (m *WorkerManagerServiceImpl) CleanupOrphanTasks() error {
	offlineWorkers := make(map[string]bool)
	m.workers.Range(func(_, val interface{}) bool {
		w := val.(*types.NodeWorkerInfo)
		if w.Status == types.WorkerStatusOffline {
			offlineWorkers[w.ID] = true
		}
		return true
	})
	if len(offlineWorkers) == 0 {
		return nil
	}
	log.Info().Int("offline_count", len(offlineWorkers)).Msg("cleanup orphan tasks: offline workers found")
	return nil
}


