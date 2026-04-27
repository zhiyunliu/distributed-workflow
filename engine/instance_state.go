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
