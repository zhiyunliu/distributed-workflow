package interfaces

import (
	"context"

	"github.com/zhiyunliu/distributed-workflow/types"
)

// AdvancedApprovalService 高级审批服务接口（D6新增）
// 提供会签、加签、转签、委托、退回、撤回、抄送等企业级审批能力
type AdvancedApprovalService interface {
	// AddSign 加签审批（前加签/后加签）
	AddSign(ctx context.Context, req types.AddSignRequest) error
	// Transfer 转签审批（审批责任转移）
	Transfer(ctx context.Context, req types.TransferRequest) error
	// Return 退回审批（退回到指定历史节点）
	Return(ctx context.Context, req types.ReturnRequest) error
	// Withdraw 撤回审批（发起人撤回）
	Withdraw(ctx context.Context, req types.WithdrawRequest) error
	// SetDelegate 设置审批委托
	SetDelegate(ctx context.Context, cfg types.DelegateConfig) error
	// GetDelegates 查询用户的委托配置列表
	GetDelegates(ctx context.Context, userID string) ([]*types.DelegateConfig, error)
	// SendCC 发送审批抄送
	SendCC(ctx context.Context, req types.CCRequest) error
	// GetCCList 分页查询抄送给当前用户的审批列表
	GetCCList(ctx context.Context, userID string, page, pageSize int) ([]*types.CCRecord, int64, error)
	// MarkCCRead 标记抄送已读
	MarkCCRead(ctx context.Context, ccID int64) error
}

// AdvancedApprovalRepository 高级审批数据仓库接口
type AdvancedApprovalRepository interface {
	// CreateDelegate 创建委托配置
	CreateDelegate(ctx context.Context, cfg *types.DelegateConfig) error
	// GetActiveDelegateByDelegator 查询用户当前有效的委托
	GetActiveDelegateByDelegator(ctx context.Context, delegatorID string) (*types.DelegateConfig, error)
	// GetDelegatesByDelegator 查询用户所有委托记录
	GetDelegatesByDelegator(ctx context.Context, delegatorID string) ([]*types.DelegateConfig, error)
	// CreateCC 创建抄送记录
	CreateCC(ctx context.Context, record *types.CCRecord) error
	// GetCCList 分页查询抄送列表
	GetCCList(ctx context.Context, userID string, page, pageSize int) ([]*types.CCRecord, int64, error)
	// MarkCCRead 标记已读
	MarkCCRead(ctx context.Context, ccID int64) error
	// CreateAddSignRecord 创建加签记录
	CreateAddSignRecord(ctx context.Context, record *types.AddSignRecord) error
	// GetAddSignRecords 查询加签记录
	GetAddSignRecords(ctx context.Context, instanceID, nodeID string) ([]*types.AddSignRecord, error)
}
