package engine

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	redisstore "github.com/zhiyunliu/distributed-workflow/storage/redis"

	"github.com/zhiyunliu/distributed-workflow/dag"
	"github.com/zhiyunliu/distributed-workflow/interfaces"
	"github.com/zhiyunliu/distributed-workflow/types"
)

const (
	scheduleLockTTL = 10 * time.Second
)

// SchedulerServiceImpl 实现 interfaces.SchedulerService
type SchedulerServiceImpl struct {
	repo    interfaces.WorkflowRepository
	state   interfaces.InstanceStateService
	worker  interfaces.WorkerManagerService
	redis   interfaces.RedisRepository
	dagPars *dag.DAGParser
}

// NewSchedulerService 创建调度服务
func NewSchedulerService(
	repo interfaces.WorkflowRepository,
	state interfaces.InstanceStateService,
	worker interfaces.WorkerManagerService,
	redis interfaces.RedisRepository,
) interfaces.SchedulerService {
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
