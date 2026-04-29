package engine

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	api "github.com/zhiyunliu/distributed-workflow/runtime-execution/pkg/api"
	"github.com/zhiyunliu/distributed-workflow/runtime-execution/internal/common/types"
)

// errorHandlerImpl ErrorHandler + 死信队列管理
type errorHandlerImpl struct {
	repo api.WorkflowRepository
}

// NewErrorHandler 创建错误处理器
func NewErrorHandler(repo api.WorkflowRepository) api.ErrorHandler {
	return &errorHandlerImpl{repo: repo}
}

// HandleError 处理节点执行错误：判断是否重试，超限则入死信队列
func (e *errorHandlerImpl) HandleError(instanceID, nodeID, workerID, workerIP string, errMsg, errCode string, retryCount int, policy *types.RetryPolicy, taskData map[string]interface{}) error {
	if policy != nil && retryCount < policy.MaxRetryCount {
		// 未达重试上限：交由调度器重试（此处只记录错误，不入死信）
		return nil
	}

	// 已超重试上限 or 无重试策略：入死信队列
	return e.MoveToDeadLetter(instanceID, nodeID, workerID, workerIP, errMsg, errCode, retryCount, taskData)
}

// MoveToDeadLetter 将任务移入死信队列
func (e *errorHandlerImpl) MoveToDeadLetter(instanceID, nodeID, workerID, workerIP, errMsg, errCode string, retryCount int, taskData map[string]interface{}) error {
	task := &types.DeadLetterTask{
		ID:         uuid.New().String(),
		InstanceID: instanceID,
		NodeID:     nodeID,
		WorkerID:   workerID,
		WorkerIP:   workerIP,
		Error:      errMsg,
		ErrorCode:  errCode,
		RetryCount: retryCount,
		TaskData:   taskData,
		CreatedAt:  time.Now(),
	}
	if err := e.repo.CreateDeadLetterTask(task); err != nil {
		return fmt.Errorf("move to dead letter queue: %w", err)
	}
	return nil
}

// ResendDeadLetterTask 重发死信任务（将节点状态重置为 pending）
func (e *errorHandlerImpl) ResendDeadLetterTask(taskID string) error {
	task, err := e.repo.GetDeadLetterTask(taskID)
	if err != nil {
		return fmt.Errorf("get dead letter task: %w", err)
	}

	// 获取节点状态
	nodeStates, err := e.repo.ListWorkflowNodeStates(task.InstanceID)
	if err != nil {
		return fmt.Errorf("get node states: %w", err)
	}

	var target *types.WorkflowNodeState
	for _, ns := range nodeStates {
		if ns.NodeID == task.NodeID {
			target = ns
			break
		}
	}
	if target == nil {
		return fmt.Errorf("node '%s' not found in instance '%s'", task.NodeID, task.InstanceID)
	}

	// 重置节点状态
	target.Status = types.WorkflowNodeStatusPending
	target.ErrorMessage = ""
	target.RetryCount = 0
	if err := e.repo.UpdateWorkflowNodeState(target); err != nil {
		return fmt.Errorf("reset node state: %w", err)
	}

	// 更新死信任务重发计数
	now := time.Now()
	task.ResendCount++
	task.LastResendAt = &now
	if err := e.repo.UpdateDeadLetterTask(task); err != nil {
		return fmt.Errorf("update dead letter task: %w", err)
	}
	return nil
}

// ListDeadLetterTasks 列举实例死信任务
func (e *errorHandlerImpl) ListDeadLetterTasks(instanceID string) ([]*types.DeadLetterTask, error) {
	return e.repo.ListDeadLetterTasks(instanceID)
}
