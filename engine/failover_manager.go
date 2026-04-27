package engine

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"github.com/zhiyunliu/distributed-workflow/interfaces"
	"github.com/zhiyunliu/distributed-workflow/rpc"
	"github.com/zhiyunliu/distributed-workflow/types"
)

const (
	failoverHealthCheckInterval = 10 * time.Second
	unhealthyThreshold          = 3 // 连续 N 次失败 = unhealthy
	offlineThreshold            = 5 // 连续 N 次失败 = offline + failover
)

// failoverManagerImpl FailoverManager 实现
type failoverManagerImpl struct {
	workerMgr    *WorkerManagerServiceImpl
	repo         interfaces.WorkflowRepository
	stateService interfaces.InstanceStateService
	workerClient *rpc.WorkerClient
	redisClient  *redis.Client
	logger       zerolog.Logger

	mu      sync.Mutex
	stopCh  chan struct{}
	wg      sync.WaitGroup
	running bool
}

// NewFailoverManager 创建故障切换管理器
func NewFailoverManager(
	workerMgr *WorkerManagerServiceImpl,
	repo interfaces.WorkflowRepository,
	stateService interfaces.InstanceStateService,
	workerClient *rpc.WorkerClient,
	redisClient *redis.Client,
	logger zerolog.Logger,
) interfaces.FailoverManager {
	return &failoverManagerImpl{
		workerMgr:    workerMgr,
		repo:         repo,
		stateService: stateService,
		workerClient: workerClient,
		redisClient:  redisClient,
		logger:       logger,
		stopCh:       make(chan struct{}),
	}
}

// StartHealthCheckLoop 启动健康检查循环
func (f *failoverManagerImpl) StartHealthCheckLoop() {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.running {
		return
	}
	f.running = true
	f.stopCh = make(chan struct{})
	f.wg.Add(1)
	go f.loop()
}

// StopHealthCheckLoop 停止健康检查循环
func (f *failoverManagerImpl) StopHealthCheckLoop() {
	f.mu.Lock()
	defer f.mu.Unlock()
	if !f.running {
		return
	}
	close(f.stopCh)
	f.wg.Wait()
	f.running = false
}

func (f *failoverManagerImpl) loop() {
	defer f.wg.Done()
	ticker := time.NewTicker(failoverHealthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-f.stopCh:
			return
		case <-ticker.C:
			f.checkAllWorkers()
		}
	}
}

func (f *failoverManagerImpl) checkAllWorkers() {
	workers := f.workerMgr.GetAllWorkers()
	for _, worker := range workers {
		if worker.Status == types.WorkerStatusOffline {
			continue
		}
		if _, err := f.HealthCheckWorker(worker.ID); err != nil {
			f.logger.Warn().Str("workerID", worker.ID).Err(err).Msg("failover: health check failed")
		}
	}
}

// HealthCheckWorker 对单个 Worker 执行健康检查，返回 (healthy bool, err)
func (f *failoverManagerImpl) HealthCheckWorker(workerID string) (bool, error) {
	worker, err := f.workerMgr.GetWorkerInfo(workerID)
	if err != nil {
		return false, fmt.Errorf("get worker info: %w", err)
	}

	healthy := f.workerClient.HealthCheck(worker.Address, workerID)

	if healthy {
		worker.HealthStatus = types.WorkerHealthStatusHealthy
		worker.FailedHeartbeatCount = 0
	} else {
		worker.FailedHeartbeatCount++
		if worker.FailedHeartbeatCount >= offlineThreshold {
			worker.HealthStatus = types.WorkerHealthStatusUnhealthy
			worker.Status = types.WorkerStatusOffline
			f.logger.Warn().Str("workerID", workerID).Msg("failover: worker offline, triggering failover")
			go func(wid string) {
				if err := f.FailoverWorker(wid); err != nil {
					f.logger.Error().Str("workerID", wid).Err(err).Msg("failover: failover worker failed")
				}
			}(workerID)
		} else if worker.FailedHeartbeatCount >= unhealthyThreshold {
			worker.HealthStatus = types.WorkerHealthStatusUnhealthy
		}
	}

	if err := f.workerMgr.UpdateWorkerInfo(worker); err != nil {
		return healthy, fmt.Errorf("update worker health: %w", err)
	}
	return healthy, nil
}

// FailoverWorker 将 Worker 上的任务故障转移（重新分配给其他可用 Worker）
func (f *failoverManagerImpl) FailoverWorker(workerID string) error {
	lockKey := "workflow:lock:failover:" + workerID
	ctx := context.Background()

	// 分布式锁，防止并发 failover
	set, err := f.redisClient.SetNX(ctx, lockKey, "1", 2*time.Minute).Result()
	if err != nil {
		return fmt.Errorf("acquire failover lock: %w", err)
	}
	if !set {
		f.logger.Info().Str("workerID", workerID).Msg("failover: lock already held, skip")
		return nil
	}
	defer f.redisClient.Del(ctx, lockKey)

	// 获取分配给该 Worker 的节点
	nodes, err := f.repo.GetAssignedNodesByWorker(workerID)
	if err != nil {
		return fmt.Errorf("get assigned nodes: %w", err)
	}

	if len(nodes) == 0 {
		return nil
	}

	var nodeIDs []string
	for _, n := range nodes {
		nodeIDs = append(nodeIDs, n.NodeID)
	}

	// 将节点重置为 pending，由调度器重新分配
	instanceGroups := make(map[string][]string)
	for _, n := range nodes {
		instanceGroups[n.InstanceID] = append(instanceGroups[n.InstanceID], n.NodeID)
	}

	for instanceID, nids := range instanceGroups {
		if err := f.stateService.BatchUpdateNodeStatus(instanceID, nids, types.WorkflowNodeStatusPending, "worker failover: "+workerID); err != nil {
			f.logger.Error().Str("instanceID", instanceID).Err(err).Msg("failover: reset nodes failed")
		}
	}

	f.logger.Info().
		Str("workerID", workerID).
		Int("nodeCount", len(nodeIDs)).
		Msg("failover: nodes reset to pending for re-dispatch")

	return nil
}
