package engine

import (
	"fmt"
	"time"

	api "github.com/zhiyunliu/distributed-workflow/runtime-execution/pkg/api"
	"github.com/zhiyunliu/distributed-workflow/runtime-execution/pkg/types"
)

// validTransitions 合法的状态转换表
var validTransitions = map[types.WorkflowStatus]map[types.WorkflowStatus]bool{
	types.WorkflowStatusPending: {
		types.WorkflowStatusRunning: true,
	},
	types.WorkflowStatusRunning: {
		types.WorkflowStatusCompleted: true,
		types.WorkflowStatusFailed:    true,
		types.WorkflowStatusPaused:    true,
		types.WorkflowStatusCancelled: true,
	},
	types.WorkflowStatusPaused: {
		types.WorkflowStatusRunning:   true, // resume
		types.WorkflowStatusCancelled: true,
	},
	types.WorkflowStatusFailed: {
		types.WorkflowStatusRunning: true, // retry
	},
	// completed / cancelled 不允许任何转换
	types.WorkflowStatusCompleted: {},
	types.WorkflowStatusCancelled: {},
}

// lifecycleManagerImpl LifecycleManager 实现
type lifecycleManagerImpl struct {
	repo      api.WorkflowRepository
	stateRepo api.InstanceStateService
}

// NewLifecycleManager 创建生命周期管理器
func NewLifecycleManager(repo api.WorkflowRepository, stateRepo api.InstanceStateService) api.LifecycleManager {
	return &lifecycleManagerImpl{repo: repo, stateRepo: stateRepo}
}

// Pause 暂停流程实例
func (l *lifecycleManagerImpl) Pause(instanceID string, operator string) error {
	inst, err := l.repo.GetWorkflowInstance(instanceID)
	if err != nil {
		return fmt.Errorf("get instance: %w", err)
	}

	if err := l.stateRepo.ValidateStateTransition(inst.Status, types.WorkflowStatusPaused); err != nil {
		return err
	}

	now := time.Now()
	inst.Status = types.WorkflowStatusPaused
	inst.PauseTime = &now
	inst.Operator = operator
	if err := l.repo.UpdateWorkflowInstance(inst); err != nil {
		return fmt.Errorf("update instance status to paused: %w", err)
	}
	return nil
}

// Resume 恢复暂停的流程实例
func (l *lifecycleManagerImpl) Resume(instanceID string, operator string) error {
	inst, err := l.repo.GetWorkflowInstance(instanceID)
	if err != nil {
		return fmt.Errorf("get instance: %w", err)
	}

	if err := l.stateRepo.ValidateStateTransition(inst.Status, types.WorkflowStatusRunning); err != nil {
		return err
	}

	inst.Status = types.WorkflowStatusRunning
	inst.PauseTime = nil
	inst.Operator = operator
	if err := l.repo.UpdateWorkflowInstance(inst); err != nil {
		return fmt.Errorf("update instance status to running: %w", err)
	}
	return nil
}

// Cancel 取消流程实例（级联取消所有 pending/running 节点）
func (l *lifecycleManagerImpl) Cancel(instanceID string, operator string) error {
	inst, err := l.repo.GetWorkflowInstance(instanceID)
	if err != nil {
		return fmt.Errorf("get instance: %w", err)
	}

	if err := l.stateRepo.ValidateStateTransition(inst.Status, types.WorkflowStatusCancelled); err != nil {
		return err
	}

	now := time.Now()
	inst.Status = types.WorkflowStatusCancelled
	inst.CancelTime = &now
	inst.Operator = operator
	if err := l.repo.UpdateWorkflowInstance(inst); err != nil {
		return fmt.Errorf("update instance status to cancelled: %w", err)
	}

	// 级联取消仍在执行/等待的节点
	nodeStates, err := l.repo.ListWorkflowNodeStates(instanceID)
	if err != nil {
		return fmt.Errorf("get node states for cancel: %w", err)
	}
	var activeNodes []string
	for _, ns := range nodeStates {
		if ns.Status == types.WorkflowNodeStatusPending || ns.Status == types.WorkflowNodeStatusRunning {
			activeNodes = append(activeNodes, ns.NodeID)
		}
	}
	if len(activeNodes) > 0 {
		if err := l.stateRepo.BatchUpdateNodeStatus(instanceID, activeNodes, types.WorkflowNodeStatusCancelled, "instance cancelled"); err != nil {
			return fmt.Errorf("batch cancel nodes: %w", err)
		}
	}
	return nil
}

// Retry 重试已失败的流程实例（重置实例状态 + 失败节点）
func (l *lifecycleManagerImpl) Retry(instanceID string, operator string) error {
	inst, err := l.repo.GetWorkflowInstance(instanceID)
	if err != nil {
		return fmt.Errorf("get instance: %w", err)
	}

	if err := l.stateRepo.ValidateStateTransition(inst.Status, types.WorkflowStatusRunning); err != nil {
		return err
	}

	inst.Status = types.WorkflowStatusRunning
	inst.Operator = operator
	if err := l.repo.UpdateWorkflowInstance(inst); err != nil {
		return fmt.Errorf("update instance status to running: %w", err)
	}

	// 将所有失败节点重置为 pending
	failedNodes, err := l.repo.GetFailedNodes(instanceID)
	if err != nil {
		return fmt.Errorf("get failed nodes: %w", err)
	}
	var failedNodeIDs []string
	for _, ns := range failedNodes {
		failedNodeIDs = append(failedNodeIDs, ns.NodeID)
	}
	if len(failedNodeIDs) > 0 {
		if err := l.stateRepo.BatchUpdateNodeStatus(instanceID, failedNodeIDs, types.WorkflowNodeStatusPending, "instance retry"); err != nil {
			return fmt.Errorf("reset failed nodes: %w", err)
		}
	}
	return nil
}

// RetryNode 重试单个失败节点
func (l *lifecycleManagerImpl) RetryNode(instanceID string, nodeID string, operator string) error {
	nodeStates, err := l.repo.ListWorkflowNodeStates(instanceID)
	if err != nil {
		return fmt.Errorf("get node states: %w", err)
	}

	var target *types.WorkflowNodeState
	for _, ns := range nodeStates {
		if ns.NodeID == nodeID {
			target = ns
			break
		}
	}
	if target == nil {
		return fmt.Errorf("node '%s' not found in instance '%s'", nodeID, instanceID)
	}
	if target.Status != types.WorkflowNodeStatusFailed {
		return fmt.Errorf("node '%s' is not in failed status (current: %s)", nodeID, target.Status)
	}

	// 更新单节点状态为 pending，重置重试计数由调度器递增
	target.Status = types.WorkflowNodeStatusPending
	target.ErrorMessage = ""
	if err := l.repo.UpdateWorkflowNodeState(target); err != nil {
		return fmt.Errorf("reset node state: %w", err)
	}
	return nil
}


