package engine

import (
	"context"
	"fmt"
	"time"

	"github.com/zhiyunliu/distributed-workflow/interfaces"
	"github.com/zhiyunliu/distributed-workflow/types"
)

// 编译时接口断言
var _ interfaces.AdvancedApprovalService = (*advancedApprovalService)(nil)

// advancedApprovalService 高级审批业务服务实现
type advancedApprovalService struct {
	repo interfaces.AdvancedApprovalRepository
}

// NewAdvancedApprovalService 创建高级审批服务
func NewAdvancedApprovalService(repo interfaces.AdvancedApprovalRepository) interfaces.AdvancedApprovalService {
	return &advancedApprovalService{repo: repo}
}

// AddSign 加签审批（前加签/后加签）
func (s *advancedApprovalService) AddSign(ctx context.Context, req types.AddSignRequest) error {
	if req.WorkflowInstanceID == "" {
		return fmt.Errorf("workflowInstanceId不能为空")
	}
	if req.NodeID == "" {
		return fmt.Errorf("nodeId不能为空")
	}
	if req.OperatorID == "" {
		return fmt.Errorf("operatorId不能为空")
	}
	if len(req.SignUsers) == 0 {
		return fmt.Errorf("signUsers不能为空")
	}
	if req.SignType != "before" && req.SignType != "after" {
		return fmt.Errorf("signType必须为before或after")
	}

	record := &types.AddSignRecord{
		WorkflowInstanceID: req.WorkflowInstanceID,
		NodeID:             req.NodeID,
		OperatorID:         req.OperatorID,
		SignType:           req.SignType,
		SignUsers:          req.SignUsers,
		CreatedAt:          time.Now(),
	}
	return s.repo.CreateAddSignRecord(ctx, record)
}

// Transfer 转签审批（审批责任转移）
func (s *advancedApprovalService) Transfer(ctx context.Context, req types.TransferRequest) error {
	if req.WorkflowInstanceID == "" {
		return fmt.Errorf("workflowInstanceId不能为空")
	}
	if req.NodeID == "" {
		return fmt.Errorf("nodeId不能为空")
	}
	if req.OperatorID == "" {
		return fmt.Errorf("operatorId不能为空")
	}
	if req.TargetUserID == "" {
		return fmt.Errorf("targetUserId不能为空")
	}
	if req.OperatorID == req.TargetUserID {
		return fmt.Errorf("不能转签给自己")
	}

	record := &types.AddSignRecord{
		WorkflowInstanceID: req.WorkflowInstanceID,
		NodeID:             req.NodeID,
		OperatorID:         req.OperatorID,
		SignType:           "transfer",
		SignUsers:          []string{req.TargetUserID},
		CreatedAt:          time.Now(),
	}
	return s.repo.CreateAddSignRecord(ctx, record)
}

// Return 退回审批（退回到指定历史节点）
func (s *advancedApprovalService) Return(ctx context.Context, req types.ReturnRequest) error {
	if req.WorkflowInstanceID == "" {
		return fmt.Errorf("workflowInstanceId不能为空")
	}
	if req.NodeID == "" {
		return fmt.Errorf("nodeId不能为空")
	}
	if req.OperatorID == "" {
		return fmt.Errorf("operatorId不能为空")
	}

	record := &types.AddSignRecord{
		WorkflowInstanceID: req.WorkflowInstanceID,
		NodeID:             req.NodeID,
		OperatorID:         req.OperatorID,
		SignType:           "return",
		SignUsers:          []string{},
		CreatedAt:          time.Now(),
	}
	return s.repo.CreateAddSignRecord(ctx, record)
}

// Withdraw 撤回审批（发起人撤回）
func (s *advancedApprovalService) Withdraw(ctx context.Context, req types.WithdrawRequest) error {
	if req.WorkflowInstanceID == "" {
		return fmt.Errorf("workflowInstanceId不能为空")
	}
	if req.OperatorID == "" {
		return fmt.Errorf("operatorId不能为空")
	}

	record := &types.AddSignRecord{
		WorkflowInstanceID: req.WorkflowInstanceID,
		NodeID:             "*",
		OperatorID:         req.OperatorID,
		SignType:           "withdraw",
		SignUsers:          []string{},
		CreatedAt:          time.Now(),
	}
	return s.repo.CreateAddSignRecord(ctx, record)
}

// SetDelegate 设置审批委托
func (s *advancedApprovalService) SetDelegate(ctx context.Context, cfg types.DelegateConfig) error {
	if cfg.DelegatorID == "" {
		return fmt.Errorf("delegatorId不能为空")
	}
	if cfg.AgentID == "" {
		return fmt.Errorf("agentId不能为空")
	}
	if cfg.DelegatorID == cfg.AgentID {
		return fmt.Errorf("不能委托给自己")
	}
	if !cfg.StartTime.Before(cfg.EndTime) {
		return fmt.Errorf("startTime必须早于endTime")
	}
	return s.repo.CreateDelegate(ctx, &cfg)
}

// GetDelegates 查询用户的委托配置列表
func (s *advancedApprovalService) GetDelegates(ctx context.Context, userID string) ([]*types.DelegateConfig, error) {
	return s.repo.GetDelegatesByDelegator(ctx, userID)
}

// SendCC 发送审批抄送
func (s *advancedApprovalService) SendCC(ctx context.Context, req types.CCRequest) error {
	if req.WorkflowInstanceID == "" {
		return fmt.Errorf("workflowInstanceId不能为空")
	}
	if req.NodeID == "" {
		return fmt.Errorf("nodeId不能为空")
	}
	if len(req.CCUserIDs) == 0 {
		return fmt.Errorf("ccUserIds不能为空")
	}

	now := time.Now()
	for _, userID := range req.CCUserIDs {
		record := &types.CCRecord{
			WorkflowInstanceID: req.WorkflowInstanceID,
			NodeID:             req.NodeID,
			CCUserID:           userID,
			CCTime:             now,
			IsRead:             false,
		}
		if err := s.repo.CreateCC(ctx, record); err != nil {
			return fmt.Errorf("创建抄送记录失败(userId=%s): %w", userID, err)
		}
	}
	return nil
}

// GetCCList 分页查询抄送给当前用户的审批列表
func (s *advancedApprovalService) GetCCList(ctx context.Context, userID string, page, pageSize int) ([]*types.CCRecord, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		return nil, 0, fmt.Errorf("pageSize必须在1到100之间")
	}
	return s.repo.GetCCList(ctx, userID, page, pageSize)
}

// MarkCCRead 标记抄送已读
func (s *advancedApprovalService) MarkCCRead(ctx context.Context, ccID int64) error {
	if ccID <= 0 {
		return fmt.Errorf("ccID必须大于0")
	}
	return s.repo.MarkCCRead(ctx, ccID)
}
