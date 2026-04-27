package engine

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/zhiyunliu/distributed-workflow/interfaces"
	"github.com/zhiyunliu/distributed-workflow/types"
)

// approvalServiceImpl 人工审批服务实现
type approvalServiceImpl struct {
	repo         interfaces.WorkflowRepository
	stateService interfaces.InstanceStateService
	auditMgr     interfaces.AuditLogManager
	callbackMgr  *callbackManagerImpl
	sched        interfaces.SchedulerService
}

// NewApprovalService 创建审批服务
func NewApprovalService(
	repo interfaces.WorkflowRepository,
	stateService interfaces.InstanceStateService,
	auditMgr interfaces.AuditLogManager,
) interfaces.ApprovalService {
	return &approvalServiceImpl{
		repo:         repo,
		stateService: stateService,
		auditMgr:     auditMgr,
		callbackMgr:  NewCallbackManager(),
	}
}

// SetScheduler 注入调度服务（解决循环依赖）
func (s *approvalServiceImpl) SetScheduler(sched interfaces.SchedulerService) {
	s.sched = sched
}

// Approve 审批通过
func (s *approvalServiceImpl) Approve(instanceID, nodeID, approver, comment string, formData map[string]interface{}, operateIP string) error {
	return s.handleApproval(instanceID, nodeID, approver, "approve", comment, formData, operateIP)
}

// Reject 审批驳回
func (s *approvalServiceImpl) Reject(instanceID, nodeID, approver, comment string, formData map[string]interface{}, operateIP string) error {
	return s.handleApproval(instanceID, nodeID, approver, "reject", comment, formData, operateIP)
}

// GetPendingTasks 获取待审批任务
func (s *approvalServiceImpl) GetPendingTasks(filter types.ApprovalTaskFilter) ([]*types.WorkflowNodeState, error) {
	return s.stateService.GetPendingApprovalTasks(filter)
}

// GetApprovalRecords 获取审批记录
func (s *approvalServiceImpl) GetApprovalRecords(instanceID, nodeID string) ([]*types.ApprovalRecord, error) {
	return s.stateService.GetApprovalRecords(instanceID, nodeID)
}

// handleApproval 处理审批动作
func (s *approvalServiceImpl) handleApproval(instanceID, nodeID, approver, action, comment string, formData map[string]interface{}, operateIP string) error {
	// 1. 获取节点状态
	nodeState, err := s.repo.GetWorkflowNodeState(instanceID, nodeID)
	if err != nil {
		return fmt.Errorf("get node state: %w", err)
	}
	if nodeState.Status != types.WorkflowNodeStatusWaiting {
		return fmt.Errorf("node %s is not in waiting state (current: %s)", nodeID, nodeState.Status)
	}
	if nodeState.ApprovalStatus != types.ApprovalStatusPending {
		return fmt.Errorf("node %s approval is not pending (current: %s)", nodeID, nodeState.ApprovalStatus)
	}

	// 2. 获取流程实例以获取工作流定义
	instance, err := s.repo.GetWorkflowInstance(instanceID)
	if err != nil {
		return fmt.Errorf("get workflow instance: %w", err)
	}

	// 3. 获取节点配置
	wfDef, err := s.repo.GetWorkflowDef(instance.WorkflowID)
	if err != nil {
		return fmt.Errorf("get workflow def: %w", err)
	}
	node := findNodeByID(wfDef, nodeID)
	if node == nil {
		return fmt.Errorf("node %s not found in workflow definition", nodeID)
	}

	htCfg := node.HumanTaskConfig
	if htCfg == nil {
		return fmt.Errorf("node %s has no human task config", nodeID)
	}

	// 4. 验证审批人权限
	if !isValidApprover(htCfg, approver, nodeState.CurrentApproverIndex) {
		return fmt.Errorf("approver %s is not authorized to approve node %s", approver, nodeID)
	}

	// 5. 记录审批动作
	record := &types.ApprovalRecord{
		ID:          uuid.New().String(),
		InstanceID:  instanceID,
		NodeID:      nodeID,
		Approver:    approver,
		Action:      action,
		Comment:     comment,
		FormData:    formData,
		OperateTime: time.Now().UTC(),
		OperateIP:   operateIP,
	}
	if err := s.stateService.AddApprovalRecord(record); err != nil {
		return fmt.Errorf("add approval record: %w", err)
	}

	// 6. 根据审批模式和动作决定下一步
	newApproverIndex := nodeState.CurrentApproverIndex
	var finalAction string

	switch htCfg.ApprovalMode {
	case types.ApprovalModeOrSign:
		// 或签：任意一人通过即可
		if action == "approve" {
			finalAction = "approve"
		} else {
			// 检查是否所有人都驳回了
			records, _ := s.stateService.GetApprovalRecords(instanceID, nodeID)
			allRejected := countRejectApprovals(records, htCfg)
			if allRejected >= len(htCfg.Approvers) {
				finalAction = "reject"
			}
		}
	case types.ApprovalModeSequential:
		// 顺序审批：按顺序依次
		if action == "reject" {
			finalAction = "reject"
		} else {
			// 通过，检查是否还有下一个审批人
			nextIndex := nodeState.CurrentApproverIndex + 1
			if nextIndex >= len(htCfg.Approvers) {
				finalAction = "approve"
			} else {
				// 更新为下一个审批人
				newApproverIndex = nextIndex
				if err := s.stateService.UpdateNodeApprovalState(instanceID, nodeID, types.ApprovalStatusPending, newApproverIndex); err != nil {
					return fmt.Errorf("update approver index: %w", err)
				}
				// 通知 Webhook
				s.notifyWebhook(htCfg, instanceID, nodeID, "sequential_progress", map[string]interface{}{
					"nextApprover": htCfg.Approvers[nextIndex],
				})
				return nil
			}
		}
	default: // ApprovalModeSingle
		finalAction = action
	}

	// 7. 最终判定：完成或驳回
	if finalAction == "" {
		// 尚未达到终态（or_sign 中有人驳回但未全部驳回）
		return nil
	}

	if finalAction == "approve" {
		return s.completeApproval(instanceID, nodeID, instance.WorkflowID, htCfg, record)
	}
	return s.rejectApproval(instanceID, nodeID, instance.WorkflowID, htCfg, record)
}

// completeApproval 审批通过，节点完成
func (s *approvalServiceImpl) completeApproval(instanceID, nodeID, workflowID string, htCfg *types.HumanTaskConfig, record *types.ApprovalRecord) error {
	nodeState, err := s.repo.GetWorkflowNodeState(instanceID, nodeID)
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	nodeState.Status = types.WorkflowNodeStatusCompleted
	nodeState.ApprovalStatus = types.ApprovalStatusApproved
	nodeState.EndTime = &now
	if record.FormData != nil {
		if nodeState.OutputData == nil {
			nodeState.OutputData = map[string]interface{}{}
		}
		nodeState.OutputData["formData"] = record.FormData
	}
	if err := s.repo.UpdateWorkflowNodeState(nodeState); err != nil {
		return fmt.Errorf("update node state on approval: %w", err)
	}

	// 审计日志
	_ = s.auditMgr.RecordLog(NewAuditLog(
		types.AuditOpApprovalPass, record.Approver, record.OperateIP,
		WithInstance(instanceID), WithWorkflow(workflowID), WithNode(nodeID),
		WithDetailf("审批通过，意见：%s", record.Comment),
	))

	// Webhook 通知
	s.notifyWebhook(htCfg, instanceID, nodeID, "approved", map[string]interface{}{
		"approver": record.Approver,
		"comment":  record.Comment,
	})

	// 触发下一轮调度
	if s.sched != nil {
		if err := s.sched.HandleNodeCompleted(instanceID, nodeID, true, nodeState.OutputData, ""); err != nil {
			log.Error().Err(err).Str("instanceID", instanceID).Str("nodeID", nodeID).Msg("approval complete trigger scheduler failed")
		}
	}
	return nil
}

// rejectApproval 审批驳回
func (s *approvalServiceImpl) rejectApproval(instanceID, nodeID, workflowID string, htCfg *types.HumanTaskConfig, record *types.ApprovalRecord) error {
	nodeState, err := s.repo.GetWorkflowNodeState(instanceID, nodeID)
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	nodeState.Status = types.WorkflowNodeStatusFailed
	nodeState.ApprovalStatus = types.ApprovalStatusRejected
	nodeState.EndTime = &now
	nodeState.ErrorMessage = fmt.Sprintf("审批驳回，意见：%s", record.Comment)
	if err := s.repo.UpdateWorkflowNodeState(nodeState); err != nil {
		return fmt.Errorf("update node state on reject: %w", err)
	}

	// 审计日志
	_ = s.auditMgr.RecordLog(NewAuditLog(
		types.AuditOpApprovalReject, record.Approver, record.OperateIP,
		WithInstance(instanceID), WithWorkflow(workflowID), WithNode(nodeID),
		WithDetailf("审批驳回，意见：%s", record.Comment),
	))

	// Webhook 通知
	s.notifyWebhook(htCfg, instanceID, nodeID, "rejected", map[string]interface{}{
		"approver": record.Approver,
		"comment":  record.Comment,
	})

	// 如果驳回动作为 goto，触发目标节点
	if htCfg != nil && htCfg.RejectAction == "goto" && htCfg.RejectGotoNodeId != "" {
		log.Info().Str("instanceID", instanceID).Str("gotoNode", htCfg.RejectGotoNodeId).Msg("approval rejected, goto node")
		// 通过 scheduler 的 OnTaskComplete 触发特定节点（需平台支持）
	} else if s.sched != nil {
		if err := s.sched.HandleNodeCompleted(instanceID, nodeID, false, nil, nodeState.ErrorMessage); err != nil {
			log.Error().Err(err).Str("instanceID", instanceID).Str("nodeID", nodeID).Msg("approval reject trigger scheduler failed")
		}
	}
	return nil
}

func (s *approvalServiceImpl) notifyWebhook(htCfg *types.HumanTaskConfig, instanceID, nodeID, event string, extra map[string]interface{}) {
	if htCfg == nil || htCfg.Webhook == "" {
		return
	}
	data := map[string]interface{}{
		"instanceId": instanceID,
		"nodeId":     nodeID,
	}
	for k, v := range extra {
		data[k] = v
	}
	_ = s.callbackMgr.InvokeWebhook(htCfg.Webhook, event, data)
}

// findNodeByID 在工作流定义中查找节点
func findNodeByID(wfDef *types.WorkflowDef, nodeID string) *types.WorkflowNode {
	if node, ok := wfDef.Nodes[nodeID]; ok {
		return node
	}
	return nil
}

// isValidApprover 校验审批人是否有权审批
func isValidApprover(cfg *types.HumanTaskConfig, approver string, currentIndex int) bool {
	if len(cfg.Approvers) == 0 {
		return true // 无限制
	}
	switch cfg.ApprovalMode {
	case types.ApprovalModeSequential:
		if currentIndex < len(cfg.Approvers) {
			return cfg.Approvers[currentIndex] == approver
		}
		return false
	default: // single, or_sign
		for _, a := range cfg.Approvers {
			if a == approver {
				return true
			}
		}
		return false
	}
}

// countRejectApprovals 统计已驳回的唯一审批人数
func countRejectApprovals(records []*types.ApprovalRecord, cfg *types.HumanTaskConfig) int {
	rejected := map[string]struct{}{}
	for _, r := range records {
		if r.Action == "reject" {
			rejected[r.Approver] = struct{}{}
		}
	}
	return len(rejected)
}
