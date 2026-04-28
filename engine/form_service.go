package engine

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/zhiyunliu/distributed-workflow/interfaces"
	"github.com/zhiyunliu/distributed-workflow/types"
)

// 编译期接口断言
var _ interfaces.FormService = (*formService)(nil)

// formService 表单引擎服务实现
type formService struct {
	repo interfaces.FormRepository
}

// NewFormService 创建表单服务
func NewFormService(repo interfaces.FormRepository) interfaces.FormService {
	return &formService{repo: repo}
}

// CreateForm 创建表单定义
func (s *formService) CreateForm(ctx context.Context, def *types.FormDefinition) error {
	if def.FormName == "" {
		return fmt.Errorf("表单名称不能为空")
	}
	now := time.Now()
	def.FormID = uuid.New().String()
	def.Version = 1
	def.Status = types.FormStatusDraft
	def.CreatedAt = now
	def.UpdatedAt = now
	return s.repo.CreateFormDef(ctx, def)
}

// GetForm 获取表单定义详情
func (s *formService) GetForm(ctx context.Context, formID string) (*types.FormDefinition, error) {
	def, err := s.repo.GetFormDef(ctx, formID)
	if err != nil {
		return nil, err
	}
	if def == nil {
		return nil, fmt.Errorf("表单不存在")
	}
	return def, nil
}

// UpdateForm 更新表单定义（仅草稿状态可修改）
func (s *formService) UpdateForm(ctx context.Context, def *types.FormDefinition) error {
	existing, err := s.repo.GetFormDef(ctx, def.FormID)
	if err != nil {
		return err
	}
	if existing == nil {
		return fmt.Errorf("表单不存在")
	}
	if existing.Status != types.FormStatusDraft {
		return fmt.Errorf("只有草稿状态的表单可以修改")
	}
	def.UpdatedAt = time.Now()
	return s.repo.UpdateFormDef(ctx, def)
}

// ListForms 分页查询表单列表
func (s *formService) ListForms(ctx context.Context, params types.FormListParams) ([]*types.FormDefinition, int64, error) {
	return s.repo.ListFormDefs(ctx, params)
}

// PublishForm 发布表单版本
func (s *formService) PublishForm(ctx context.Context, formID, changeLog, operatorID string) error {
	existing, err := s.repo.GetFormDef(ctx, formID)
	if err != nil {
		return err
	}
	if existing == nil {
		return fmt.Errorf("表单不存在")
	}
	if existing.Status == types.FormStatusDisabled {
		return fmt.Errorf("已停用的表单不可发布")
	}
	return s.repo.PublishFormDef(ctx, formID, changeLog, operatorID)
}

// RollbackForm 回滚表单到指定版本
func (s *formService) RollbackForm(ctx context.Context, formID string, version int, operatorID string) error {
	histories, err := s.repo.GetFormVersionHistory(ctx, formID)
	if err != nil {
		return err
	}
	found := false
	for _, h := range histories {
		if h.Version == version {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("指定版本 %d 不存在", version)
	}
	return s.repo.RollbackFormDef(ctx, formID, version, operatorID)
}

// GetFormVersions 查询表单版本历史
func (s *formService) GetFormVersions(ctx context.Context, formID string) ([]*types.FormVersionHistory, error) {
	return s.repo.GetFormVersionHistory(ctx, formID)
}

// SaveFormInstance 保存表单实例数据（不存在则创建，存在则更新）
func (s *formService) SaveFormInstance(ctx context.Context, inst *types.FormInstance) error {
	if inst.WorkflowInstanceID == "" {
		return fmt.Errorf("流程实例ID不能为空")
	}
	now := time.Now()
	if inst.InstanceID == "" {
		inst.InstanceID = uuid.New().String()
		inst.CreatedAt = now
		inst.UpdatedAt = now
		return s.repo.CreateFormInstance(ctx, inst)
	}
	inst.UpdatedAt = now
	return s.repo.UpdateFormInstance(ctx, inst)
}

// GetFormInstance 获取表单实例
func (s *formService) GetFormInstance(ctx context.Context, instanceID string) (*types.FormInstance, error) {
	inst, err := s.repo.GetFormInstance(ctx, instanceID)
	if err != nil {
		return nil, err
	}
	if inst == nil {
		return nil, fmt.Errorf("表单实例不存在")
	}
	return inst, nil
}

// GetFormInstanceByWorkflow 按流程实例和节点查询表单实例
func (s *formService) GetFormInstanceByWorkflow(ctx context.Context, workflowInstanceID, nodeID string) (*types.FormInstance, error) {
	return s.repo.GetFormInstanceByWorkflow(ctx, workflowInstanceID, nodeID)
}
