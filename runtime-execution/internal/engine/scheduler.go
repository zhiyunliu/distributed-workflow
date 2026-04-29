package engine

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	redisstore "github.com/zhiyunliu/distributed-workflow/runtime-execution/internal/storage/redis"

	"github.com/zhiyunliu/distributed-workflow/runtime-execution/internal/common/dag"
	api "github.com/zhiyunliu/distributed-workflow/runtime-execution/pkg/api"
	"github.com/zhiyunliu/distributed-workflow/runtime-execution/internal/common/types"
)

const (
	scheduleLockTTL = 10 * time.Second
)

// SchedulerServiceImpl 实现 api.SchedulerService
type SchedulerServiceImpl struct {
	repo    api.WorkflowRepository
	state   api.InstanceStateService
	worker  api.WorkerManagerService
	redis   api.RedisRepository
	dagPars *dag.DAGParser
}

// NewSchedulerService 创建调度服务
func NewSchedulerService(
	repo api.WorkflowRepository,
	state api.InstanceStateService,
	worker api.WorkerManagerService,
	redis api.RedisRepository,
) api.SchedulerService {
	return &SchedulerServiceImpl{
		repo:    repo,
		state:   state,
		worker:  worker,
		redis:   redis,
		dagPars: dag.NewDAGParser(),
	}
}

// StartExecution 启动流程实例执行：更新状态 → 创建上下文 → 调度第一批节点
func (s *SchedulerServiceImpl) StartExecution(instanceID string) error {
	instance, err := s.repo.GetWorkflowInstance(instanceID)
	if err != nil {
		return fmt.Errorf("get instance '%s': %w", instanceID, err)
	}

	// 更新实例状态为 running
	if err := s.state.UpdateInstanceStatus(instanceID, types.WorkflowStatusRunning, ""); err != nil {
		return fmt.Errorf("update instance status: %w", err)
	}

	// 创建流程上下文
	ctx := &types.WorkflowContext{
		InstanceID: instanceID,
		Data:       copyMap(instance.InputData),
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	if err := s.repo.CreateWorkflowContext(ctx); err != nil {
		return fmt.Errorf("create workflow context: %w", err)
	}

	return s.ScheduleNextNodes(instanceID)
}

// ScheduleNextNodes 获取 DAG 中当前可执行的节点并分配任务
func (s *SchedulerServiceImpl) ScheduleNextNodes(instanceID string) error {
	instance, err := s.repo.GetWorkflowInstance(instanceID)
	if err != nil {
		return fmt.Errorf("get instance '%s': %w", instanceID, err)
	}

	// 暂停或已终止的实例不继续调度
	if instance.Status == types.WorkflowStatusPaused ||
		instance.Status == types.WorkflowStatusCancelled ||
		instance.Status == types.WorkflowStatusCompleted {
		return nil
	}

	def, err := s.repo.GetWorkflowDef(instance.WorkflowID)
	if err != nil {
		return fmt.Errorf("get workflow def '%s': %w", instance.WorkflowID, err)
	}

	graph, err := s.dagPars.Parse(def)
	if err != nil {
		return fmt.Errorf("parse dag: %w", err)
	}

	// 加载所有节点状态，构建 map
	nodeStates, err := s.repo.ListWorkflowNodeStates(instanceID)
	if err != nil {
		return fmt.Errorf("list node states: %w", err)
	}
	stateMap := make(map[string]*types.WorkflowNodeState, len(nodeStates))
	for _, ns := range nodeStates {
		stateMap[ns.NodeID] = ns
	}

	// 获取流程上下文
	wfCtx, err := s.repo.GetWorkflowContext(instanceID)
	if err != nil {
		return fmt.Errorf("get workflow context: %w", err)
	}

	readyNodeIDs := graph.GetReadyNodes(stateMap)
	if len(readyNodeIDs) == 0 {
		// 检查流程是否全部完成
		return s.checkFlowCompletion(instanceID, def.EndNodeID, stateMap)
	}

	for _, nodeID := range readyNodeIDs {
		nodeID := nodeID
		// 分布式锁：防止并发重复调度同一节点
		lockKey := redisstore.LockKeyFormat(instanceID, nodeID)
		lockVal := uuid.New().String()
		locked, err := s.redis.Lock(lockKey, lockVal, scheduleLockTTL)
		if err != nil || !locked {
			log.Debug().Str("instance_id", instanceID).Str("node_id", nodeID).Msg("node already being scheduled, skip")
			continue
		}

		if err := s.scheduleNode(instance, def, graph, nodeID, wfCtx, stateMap); err != nil {
			log.Error().Err(err).
				Str("instance_id", instanceID).
				Str("node_id", nodeID).
				Msg("schedule node failed")
			// 释放锁，让下次重试
			_, _ = s.redis.Unlock(lockKey, lockVal)
		}
		// 锁在任务分配后随 TTL 自然过期（防止重复分配）
	}
	return nil
}

// scheduleNode 为单个节点创建 NodeState 并分配给 Worker
func (s *SchedulerServiceImpl) scheduleNode(
	instance *types.WorkflowInstance,
	def *types.WorkflowDef,
	graph *dag.DAGGraph,
	nodeID string,
	wfCtx *types.WorkflowContext,
	stateMap map[string]*types.WorkflowNodeState,
) error {
	node, ok := graph.GetNode(nodeID)
	if !ok {
		return fmt.Errorf("node '%s' not found in DAG", nodeID)
	}

	// 避免重复创建 NodeState
	if _, exists := stateMap[nodeID]; exists {
		return nil
	}

	now := time.Now()
	nodeState := &types.WorkflowNodeState{
		InstanceID: instance.ID,
		NodeID:     nodeID,
		Status:     types.WorkflowNodeStatusAssigned,
		StartTime:  &now,
		InputData:  wfCtx.Data,
	}

	// 选择最优 Worker
	worker, err := s.worker.SelectBestWorker(node.Type)
	if err != nil {
		return fmt.Errorf("select worker for node type '%s': %w", node.Type, err)
	}
	nodeState.AssignedTo = worker.ID
	nodeState.WorkerIP = worker.IP

	if err := s.state.CreateNodeState(nodeState); err != nil {
		return fmt.Errorf("create node state: %w", err)
	}

	task := &types.Task{
		TaskID:     fmt.Sprintf("%s-%s", instance.ID, nodeID),
		InstanceID: instance.ID,
		NodeID:     nodeID,
		NodeType:   node.Type,
		Config:     node.Config,
		InputData:  wfCtx.Data,
		Timeout:    node.Timeout,
	}

	if err := s.worker.AssignTask(worker.ID, task); err != nil {
		// 回滚 NodeState
		nodeState.Status = types.WorkflowNodeStatusFailed
		nodeState.ErrorMessage = err.Error()
		_ = s.state.UpdateNodeState(nodeState)
		return fmt.Errorf("assign task: %w", err)
	}

	log.Info().
		Str("instance_id", instance.ID).
		Str("node_id", nodeID).
		Str("worker_id", worker.ID).
		Msg("node scheduled")
	return nil
}

// HandleNodeCompleted 节点完成后处理：更新上下文 → 检查流程完成 → 调度下一批节点
func (s *SchedulerServiceImpl) HandleNodeCompleted(
	instanceID string,
	nodeID string,
	success bool,
	output map[string]interface{},
	errMsg string,
) error {
	// 如果节点失败，将流程标记为失败
	if !success {
		if err := s.state.UpdateInstanceStatus(instanceID, types.WorkflowStatusFailed, errMsg); err != nil {
			log.Error().Err(err).Str("instance_id", instanceID).Msg("update instance status to failed")
		}
		return nil
	}

	// 合并输出数据到流程上下文
	if len(output) > 0 {
		if err := s.mergeContext(instanceID, output); err != nil {
			log.Error().Err(err).Str("instance_id", instanceID).Msg("merge context failed")
		}
	}

	// Worker 负载减一
	nodeStates, _ := s.repo.ListWorkflowNodeStates(instanceID)
	for _, ns := range nodeStates {
		if ns.NodeID == nodeID && ns.AssignedTo != "" {
			instance, _ := s.repo.GetWorkflowInstance(instanceID)
			if instance != nil {
				def, _ := s.repo.GetWorkflowDef(instance.WorkflowID)
				if def != nil {
					if node, ok := def.Nodes[nodeID]; ok {
						_ = s.worker.UpdateWorkerLoad(ns.AssignedTo, node.Type, -1)
					}
				}
			}
			break
		}
	}

	return s.ScheduleNextNodes(instanceID)
}

// RecoverUnfinishedInstances 引擎重启时恢复未完成实例
func (s *SchedulerServiceImpl) RecoverUnfinishedInstances() error {
	instances, err := s.repo.ListUnfinishedInstances()
	if err != nil {
		return fmt.Errorf("list unfinished instances: %w", err)
	}
	log.Info().Int("count", len(instances)).Msg("recovering unfinished instances")
	for _, inst := range instances {
		if err := s.ScheduleNextNodes(inst.ID); err != nil {
			log.Error().Err(err).Str("instance_id", inst.ID).Msg("recover instance failed")
		}
	}
	return nil
}

// checkFlowCompletion 检查流程是否已全部完成
func (s *SchedulerServiceImpl) checkFlowCompletion(
	instanceID string,
	endNodeID string,
	stateMap map[string]*types.WorkflowNodeState,
) error {
	endState, ok := stateMap[endNodeID]
	if !ok {
		return nil // 结束节点还没执行
	}
	if endState.Status == types.WorkflowNodeStatusCompleted {
		if err := s.state.UpdateInstanceStatus(instanceID, types.WorkflowStatusCompleted, ""); err != nil {
			return fmt.Errorf("update instance to completed: %w", err)
		}
		log.Info().Str("instance_id", instanceID).Msg("workflow completed")
	}
	return nil
}

// mergeContext 将节点输出合并到流程上下文
func (s *SchedulerServiceImpl) mergeContext(instanceID string, output map[string]interface{}) error {
	ctx, err := s.repo.GetWorkflowContext(instanceID)
	if err != nil {
		return err
	}
	for k, v := range output {
		ctx.Data[k] = v
	}
	ctx.UpdatedAt = time.Now()
	return s.repo.UpdateWorkflowContext(ctx)
}

// copyMap 浅拷贝 map
func copyMap(src map[string]interface{}) map[string]interface{} {
	if src == nil {
		return make(map[string]interface{})
	}
	dst := make(map[string]interface{}, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

// marshalJSON 辅助序列化（供后续使用）
func marshalJSON(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

// ─────────────────────────────────────────────────────────────────────────────
// D2 新增：生命周期调度控制
// ─────────────────────────────────────────────────────────────────────────────

// PauseExecution 暂停实例调度（中断继续派发，正在执行的节点不干预）
func (s *SchedulerServiceImpl) PauseExecution(instanceID string) error {
	// 暂停实例状态已由 LifecycleManager 更新；此处只需确保不再派发新节点
	// ScheduleNextNodes 内部会检查实例状态，paused 状态下直接返回
	log.Info().Str("instance_id", instanceID).Msg("execution paused")
	return nil
}

// ResumeExecution 恢复实例调度
func (s *SchedulerServiceImpl) ResumeExecution(instanceID string) error {
	return s.ScheduleNextNodes(instanceID)
}

// CancelExecution 取消执行（向所有运行中节点发送 Pause/Cancel 指令）
func (s *SchedulerServiceImpl) CancelExecution(instanceID string) error {
	nodeStates, err := s.repo.ListWorkflowNodeStates(instanceID)
	if err != nil {
		return fmt.Errorf("cancel execution list node states: %w", err)
	}
	for _, ns := range nodeStates {
		if ns.Status == types.WorkflowNodeStatusRunning && ns.AssignedTo != "" {
			if err := s.worker.HandleWorkerOffline(ns.AssignedTo); err != nil {
				log.Warn().Err(err).Str("node_id", ns.NodeID).Msg("cancel: notify worker failed")
			}
		}
	}
	log.Info().Str("instance_id", instanceID).Msg("execution cancelled")
	return nil
}

// RetryExecution 重试整个实例（重新调度）
func (s *SchedulerServiceImpl) RetryExecution(instanceID string) error {
	return s.ScheduleNextNodes(instanceID)
}

// RetrySingleNode 重试单个节点
func (s *SchedulerServiceImpl) RetrySingleNode(instanceID string, nodeID string) error {
	instance, err := s.repo.GetWorkflowInstance(instanceID)
	if err != nil {
		return fmt.Errorf("get instance: %w", err)
	}
	def, err := s.repo.GetWorkflowDef(instance.WorkflowID)
	if err != nil {
		return fmt.Errorf("get def: %w", err)
	}
	graph, err := s.dagPars.Parse(def)
	if err != nil {
		return fmt.Errorf("parse dag: %w", err)
	}
	nodeStates, err := s.repo.ListWorkflowNodeStates(instanceID)
	if err != nil {
		return fmt.Errorf("list node states: %w", err)
	}
	stateMap := make(map[string]*types.WorkflowNodeState, len(nodeStates))
	for _, ns := range nodeStates {
		stateMap[ns.NodeID] = ns
	}
	wfCtx, err := s.repo.GetWorkflowContext(instanceID)
	if err != nil {
		return fmt.Errorf("get context: %w", err)
	}
	// 删除 stateMap 中目标节点，以便 scheduleNode 重新创建
	delete(stateMap, nodeID)
	return s.scheduleNode(instance, def, graph, nodeID, wfCtx, stateMap)
}

// HandleWorkerFailure 处理 Worker 故障：将分配给该 Worker 的节点重新调度
func (s *SchedulerServiceImpl) HandleWorkerFailure(workerID string) error {
	nodes, err := s.repo.GetAssignedNodesByWorker(workerID)
	if err != nil {
		return fmt.Errorf("get assigned nodes by worker: %w", err)
	}
	// 重置节点为 pending 状态，等待重新调度
	instanceGroups := make(map[string][]string)
	for _, n := range nodes {
		instanceGroups[n.InstanceID] = append(instanceGroups[n.InstanceID], n.NodeID)
	}
	for instanceID, nodeIDs := range instanceGroups {
		if err := s.state.BatchUpdateNodeStatus(instanceID, nodeIDs, types.WorkflowNodeStatusPending, "worker failure: "+workerID); err != nil {
			log.Error().Err(err).Str("instance_id", instanceID).Msg("reset nodes for worker failure")
		}
		// 触发重新调度
		if err := s.ScheduleNextNodes(instanceID); err != nil {
			log.Error().Err(err).Str("instance_id", instanceID).Msg("reschedule after worker failure")
		}
	}
	return nil
}

// ProcessRetryQueue 处理到期的重试节点（定时调用）
func (s *SchedulerServiceImpl) ProcessRetryQueue() error {
	// 从数据库查询 next_retry_time <= now 且 status = 'failed' 的节点
	// 此处调用 ScheduleNextNodes 对各实例重新调度
	instances, err := s.repo.ListUnfinishedInstances()
	if err != nil {
		return fmt.Errorf("process retry queue: list instances: %w", err)
	}
	now := time.Now()
	for _, inst := range instances {
		nodeStates, err := s.repo.ListWorkflowNodeStates(inst.ID)
		if err != nil {
			continue
		}
		for _, ns := range nodeStates {
			if ns.Status == types.WorkflowNodeStatusFailed && ns.NextRetryTime != nil && !ns.NextRetryTime.After(now) {
				// 重置为 pending 触发重新调度
				ns.Status = types.WorkflowNodeStatusPending
				if err := s.repo.UpdateWorkflowNodeState(ns); err != nil {
					log.Error().Err(err).Str("node_id", ns.NodeID).Msg("retry queue reset node failed")
				}
			}
		}
		_ = s.ScheduleNextNodes(inst.ID)
	}
	return nil
}
