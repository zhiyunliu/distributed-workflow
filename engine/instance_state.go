package engine

import (
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/zhiyunliu/distributed-workflow/interfaces"
	"github.com/zhiyunliu/distributed-workflow/types"
)

// InstanceStateServiceImpl 实现 interfaces.InstanceStateService
type InstanceStateServiceImpl struct {
	repo  interfaces.WorkflowRepository
	sched interfaces.SchedulerService
}

// NewInstanceStateService 创建 InstanceStateService 实例
// sched 可以为 nil，后续通过 SetScheduler 注入（解决循环依赖）
func NewInstanceStateService(repo interfaces.WorkflowRepository) *InstanceStateServiceImpl {
	return &InstanceStateServiceImpl{repo: repo}
}

// SetScheduler 注入调度服务（解决 SchedulerService↔InstanceStateService 循环依赖）
func (s *InstanceStateServiceImpl) SetScheduler(sched interfaces.SchedulerService) {
	s.sched = sched
}

func (s *InstanceStateServiceImpl) CreateInstance(instance *types.WorkflowInstance) error {
	now := time.Now()
	instance.StartTime = now
	return s.repo.CreateWorkflowInstance(instance)
}

func (s *InstanceStateServiceImpl) UpdateInstanceStatus(instanceID string, status types.WorkflowStatus, errMsg string) error {
	instance, err := s.repo.GetWorkflowInstance(instanceID)
	if err != nil {
		return fmt.Errorf("get instance '%s': %w", instanceID, err)
	}
	instance.Status = status
	instance.ErrorMessage = errMsg
	if status == types.WorkflowStatusCompleted || status == types.WorkflowStatusFailed {
		now := time.Now()
		instance.EndTime = &now
	}
	return s.repo.UpdateWorkflowInstance(instance)
}

func (s *InstanceStateServiceImpl) CreateNodeState(state *types.WorkflowNodeState) error {
	return s.repo.CreateWorkflowNodeState(state)
}

func (s *InstanceStateServiceImpl) UpdateNodeState(state *types.WorkflowNodeState) error {
	return s.repo.UpdateWorkflowNodeState(state)
}

func (s *InstanceStateServiceImpl) GetNodeState(instanceID string, nodeID string) (*types.WorkflowNodeState, error) {
	return s.repo.GetWorkflowNodeState(instanceID, nodeID)
}

func (s *InstanceStateServiceImpl) GetInstanceNodeStates(instanceID string) ([]*types.WorkflowNodeState, error) {
	return s.repo.ListWorkflowNodeStates(instanceID)
}

// ReportTaskResult 接收 Worker 上报的任务结果，更新节点状态并触发后续调度
func (s *InstanceStateServiceImpl) ReportTaskResult(result *types.TaskResult) error {
	state, err := s.repo.GetWorkflowNodeState(result.InstanceID, result.NodeID)
	if err != nil {
		return fmt.Errorf("get node state '%s/%s': %w", result.InstanceID, result.NodeID, err)
	}

	// 人工任务等待状态：不调度后续节点，等待审批服务驱动
	if result.ErrorMessage == "node is waiting for human approval" {
		state.Status = types.WorkflowNodeStatusWaiting
		state.ApprovalStatus = types.ApprovalStatusPending
		if err := s.repo.UpdateWorkflowNodeState(state); err != nil {
			return fmt.Errorf("update node state to waiting: %w", err)
		}
		log.Info().
			Str("instance_id", result.InstanceID).
			Str("node_id", result.NodeID).
			Msg("task entered human approval waiting state")
		return nil
	}

	now := result.EndTime
	state.EndTime = &now
	state.OutputData = result.OutputData
	if result.Success {
		state.Status = types.WorkflowNodeStatusCompleted
	} else {
		state.Status = types.WorkflowNodeStatusFailed
		state.ErrorMessage = result.ErrorMessage
	}

	if err := s.repo.UpdateWorkflowNodeState(state); err != nil {
		return fmt.Errorf("update node state: %w", err)
	}

	log.Info().
		Str("instance_id", result.InstanceID).
		Str("node_id", result.NodeID).
		Bool("success", result.Success).
		Msg("task result reported")

	// 触发下一步调度
	if s.sched != nil {
		if err := s.sched.HandleNodeCompleted(
			result.InstanceID,
			result.NodeID,
			result.Success,
			result.OutputData,
			result.ErrorMessage,
		); err != nil {
			log.Error().Err(err).
				Str("instance_id", result.InstanceID).
				Str("node_id", result.NodeID).
				Msg("handle node completed failed")
		}
	}
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// D2 新增方法
// ─────────────────────────────────────────────────────────────────────────────

// ValidateStateTransition 验证状态转换是否合法
func (s *InstanceStateServiceImpl) ValidateStateTransition(fromStatus, toStatus types.WorkflowStatus) error {
	targets, ok := validTransitions[fromStatus]
	if !ok {
		return fmt.Errorf("unknown source status '%s'", fromStatus)
	}
	if !targets[toStatus] {
		return fmt.Errorf("invalid state transition: %s → %s", fromStatus, toStatus)
	}
	return nil
}

// BatchUpdateNodeStatus 批量更新节点状态
func (s *InstanceStateServiceImpl) BatchUpdateNodeStatus(instanceID string, nodeIDs []string, status types.WorkflowNodeStatus, reason string) error {
	return s.repo.BatchUpdateNodeStates(instanceID, nodeIDs, status, reason)
}

// UpdateRetryState 更新节点重试状态（重试计数 + 下次重试时间）
func (s *InstanceStateServiceImpl) UpdateRetryState(instanceID string, nodeID string, retryCount int, nextRetryTimeUnix *int64) error {
	state, err := s.repo.GetWorkflowNodeState(instanceID, nodeID)
	if err != nil {
		return fmt.Errorf("get node state: %w", err)
	}
	state.RetryCount = retryCount
	if nextRetryTimeUnix != nil {
		t := time.Unix(*nextRetryTimeUnix, 0)
		state.NextRetryTime = &t
	}
	return s.repo.UpdateWorkflowNodeState(state)
}

// GetFailedNodes 获取实例所有失败节点
func (s *InstanceStateServiceImpl) GetFailedNodes(instanceID string) ([]*types.WorkflowNodeState, error) {
	return s.repo.GetFailedNodes(instanceID)
}

// GetAssignedNodesByWorker 获取分配给指定 Worker 的节点
func (s *InstanceStateServiceImpl) GetAssignedNodesByWorker(workerID string) ([]*types.WorkflowNodeState, error) {
	return s.repo.GetAssignedNodesByWorker(workerID)
}

// ─── D3 新增 ───

// UpdateNodeApprovalState 更新节点审批状态
func (s *InstanceStateServiceImpl) UpdateNodeApprovalState(instanceID, nodeID string, status types.ApprovalStatus, approverIndex int) error {
	state, err := s.repo.GetWorkflowNodeState(instanceID, nodeID)
	if err != nil {
		return fmt.Errorf("UpdateNodeApprovalState get state: %w", err)
	}
	state.ApprovalStatus = status
	state.CurrentApproverIndex = approverIndex
	return s.repo.UpdateWorkflowNodeState(state)
}

// GetPendingApprovalTasks 查询待审批任务列表
func (s *InstanceStateServiceImpl) GetPendingApprovalTasks(filter types.ApprovalTaskFilter) ([]*types.WorkflowNodeState, error) {
	return s.repo.GetPendingApprovalNodeStates(filter)
}

// AddApprovalRecord 添加审批记录
func (s *InstanceStateServiceImpl) AddApprovalRecord(record *types.ApprovalRecord) error {
	return s.repo.CreateApprovalRecord(record)
}

// GetApprovalRecords 获取节点审批记录
func (s *InstanceStateServiceImpl) GetApprovalRecords(instanceID, nodeID string) ([]*types.ApprovalRecord, error) {
	return s.repo.GetApprovalRecords(instanceID, nodeID)
}
