package engine

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/zhiyunliu/distributed-workflow/interfaces"
	"github.com/zhiyunliu/distributed-workflow/types"
)

const maxSubflowDepth = 5

// subflowManagerImpl SubflowManager 实现
type subflowManagerImpl struct {
	workflowSvc interfaces.WorkflowService
	repo        interfaces.WorkflowRepository
}

// NewSubflowManager 创建子流程管理器
func NewSubflowManager(workflowSvc interfaces.WorkflowService, repo interfaces.WorkflowRepository) interfaces.SubflowManager {
	return &subflowManagerImpl{workflowSvc: workflowSvc, repo: repo}
}

// TriggerSubflow 触发子流程
func (s *subflowManagerImpl) TriggerSubflow(parentInstanceID, parentNodeID string, subflowID string, inputData map[string]interface{}, mode types.SubflowCallMode) (string, error) {
	// 深度检查
	depth, err := s.GetSubflowDepth(parentInstanceID)
	if err != nil {
		return "", fmt.Errorf("get subflow depth: %w", err)
	}
	if depth >= maxSubflowDepth {
		return "", fmt.Errorf("max subflow nesting depth (%d) exceeded", maxSubflowDepth)
	}

	// 获取子流程定义
	def, err := s.repo.GetWorkflowDef(subflowID)
	if err != nil {
		return "", fmt.Errorf("get subflow def '%s': %w", subflowID, err)
	}

	// 创建子流程实例
	instanceID := uuid.New().String()
	instance := &types.WorkflowInstance{
		ID:               instanceID,
		WorkflowID:       def.ID,
		Status:           types.WorkflowStatusPending,
		InputData:        inputData,
		IsSubflowInstance: true,
		ParentInstanceID: parentInstanceID,
		ParentNodeID:     parentNodeID,
	}

	if _, err := s.workflowSvc.StartWorkflow(def.ID, inputData, "subflow"); err != nil {
		return "", fmt.Errorf("start subflow instance: %w", err)
	}
	_ = instance // instance var created above for reference

	return instanceID, nil
}

// OnSubflowComplete 子流程完成回调（更新父节点状态）
func (s *subflowManagerImpl) OnSubflowComplete(subflowInstanceID string, success bool, outputData map[string]interface{}) error {
	inst, err := s.repo.GetWorkflowInstance(subflowInstanceID)
	if err != nil {
		return fmt.Errorf("get subflow instance: %w", err)
	}
	if !inst.IsSubflowInstance || inst.ParentInstanceID == "" {
		return nil
	}

	// 更新父节点状态
	nodeStates, err := s.repo.ListWorkflowNodeStates(inst.ParentInstanceID)
	if err != nil {
		return fmt.Errorf("get parent node states: %w", err)
	}

	var parentNode *types.WorkflowNodeState
	for _, ns := range nodeStates {
		if ns.NodeID == inst.ParentNodeID {
			parentNode = ns
			break
		}
	}
	if parentNode == nil {
		return fmt.Errorf("parent node '%s' not found", inst.ParentNodeID)
	}

	if success {
		parentNode.Status = types.WorkflowNodeStatusCompleted
		parentNode.OutputData = outputData
	} else {
		parentNode.Status = types.WorkflowNodeStatusFailed
		parentNode.ErrorMessage = "subflow failed"
	}

	if err := s.repo.UpdateWorkflowNodeState(parentNode); err != nil {
		return fmt.Errorf("update parent node state: %w", err)
	}
	return nil
}

// GetSubflowDepth 计算实例的子流程嵌套深度
func (s *subflowManagerImpl) GetSubflowDepth(instanceID string) (int, error) {
	depth := 0
	currentID := instanceID
	visited := make(map[string]bool)

	for {
		if visited[currentID] {
			return 0, fmt.Errorf("cycle detected in subflow chain at '%s'", currentID)
		}
		visited[currentID] = true

		inst, err := s.repo.GetWorkflowInstance(currentID)
		if err != nil {
			return 0, fmt.Errorf("get instance '%s': %w", currentID, err)
		}

		if !inst.IsSubflowInstance || inst.ParentInstanceID == "" {
			break
		}
		depth++
		currentID = inst.ParentInstanceID
	}
	return depth, nil
}
