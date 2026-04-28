package interfaces

import (
	"context"

	"github.com/zhiyunliu/distributed-workflow/types"
)

// FormService 表单引擎服务接口
// 负责表单定义的版本管理、发布回滚，以及流程节点表单实例的存取
type FormService interface {
	// CreateForm 创建表单定义
	CreateForm(ctx context.Context, def *types.FormDefinition) error
	// GetForm 获取表单定义详情
	GetForm(ctx context.Context, formID string) (*types.FormDefinition, error)
	// UpdateForm 更新表单定义（仅草稿状态可修改）
	UpdateForm(ctx context.Context, def *types.FormDefinition) error
	// ListForms 分页查询表单列表
	ListForms(ctx context.Context, params types.FormListParams) ([]*types.FormDefinition, int64, error)
	// PublishForm 发布表单版本
	PublishForm(ctx context.Context, formID, changeLog, operatorID string) error
	// RollbackForm 回滚表单到指定版本
	RollbackForm(ctx context.Context, formID string, version int, operatorID string) error
	// GetFormVersions 查询表单版本历史
	GetFormVersions(ctx context.Context, formID string) ([]*types.FormVersionHistory, error)
	// SaveFormInstance 保存表单实例数据（不存在则创建，存在则更新）
	SaveFormInstance(ctx context.Context, inst *types.FormInstance) error
	// GetFormInstance 获取表单实例
	GetFormInstance(ctx context.Context, instanceID string) (*types.FormInstance, error)
	// GetFormInstanceByWorkflow 按流程实例和节点查询表单实例
	GetFormInstanceByWorkflow(ctx context.Context, workflowInstanceID, nodeID string) (*types.FormInstance, error)
}
