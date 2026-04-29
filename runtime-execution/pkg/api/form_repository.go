package api

import (
	"context"

	"github.com/zhiyunliu/distributed-workflow/runtime-execution/internal/common/types"
)

// FormRepository 表单数据仓库接口
// 所有写操作需通过数据库事务保证原子性
type FormRepository interface {
	// ─── 表单定义 ─────────────────────────────────────────────────────

	// CreateFormDef 创建表单定义
	CreateFormDef(ctx context.Context, def *types.FormDefinition) error
	// GetFormDef 获取表单定义
	GetFormDef(ctx context.Context, formID string) (*types.FormDefinition, error)
	// UpdateFormDef 更新表单定义
	UpdateFormDef(ctx context.Context, def *types.FormDefinition) error
	// ListFormDefs 分页查询表单定义
	ListFormDefs(ctx context.Context, params types.FormListParams) ([]*types.FormDefinition, int64, error)

	// ─── 版本管理 ─────────────────────────────────────────────────────

	// PublishFormDef 发布表单版本（递增version，写入version_history，更新status）
	PublishFormDef(ctx context.Context, formID, changeLog, operatorID string) error
	// RollbackFormDef 回滚表单版本
	RollbackFormDef(ctx context.Context, formID string, version int, operatorID string) error
	// GetFormVersionHistory 查询表单版本历史
	GetFormVersionHistory(ctx context.Context, formID string) ([]*types.FormVersionHistory, error)

	// ─── 表单实例 ─────────────────────────────────────────────────────

	// CreateFormInstance 创建表单实例
	CreateFormInstance(ctx context.Context, inst *types.FormInstance) error
	// UpdateFormInstance 更新表单实例
	UpdateFormInstance(ctx context.Context, inst *types.FormInstance) error
	// GetFormInstance 获取表单实例
	GetFormInstance(ctx context.Context, instanceID string) (*types.FormInstance, error)
	// GetFormInstanceByWorkflow 按流程实例查询表单实例
	GetFormInstanceByWorkflow(ctx context.Context, workflowInstanceID, nodeID string) (*types.FormInstance, error)
}
