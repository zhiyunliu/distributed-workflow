package engine

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	api "github.com/zhiyunliu/distributed-workflow/runtime-execution/pkg/api"
	"github.com/zhiyunliu/distributed-workflow/runtime-execution/pkg/types"
)

type templateServiceImpl struct {
	repo      api.WorkflowRepository
	workflow  api.WorkflowService
	auditLogs api.AuditLogManager
}

// NewTemplateService 创建流程模板服务
func NewTemplateService(
	repo api.WorkflowRepository,
	workflow api.WorkflowService,
	auditLogs api.AuditLogManager,
) api.TemplateService {
	return &templateServiceImpl{
		repo:      repo,
		workflow:  workflow,
		auditLogs: auditLogs,
	}
}

func (s *templateServiceImpl) ListTemplates(filter types.WorkflowTemplateFilter, page, pageSize int) ([]*types.WorkflowTemplate, int64, error) {
	return s.repo.ListWorkflowTemplates(filter, page, pageSize)
}

func (s *templateServiceImpl) GetTemplate(templateID string) (*types.WorkflowTemplate, error) {
	return s.repo.GetWorkflowTemplate(templateID)
}

func (s *templateServiceImpl) CreateTemplate(template *types.WorkflowTemplate) (string, error) {
	if err := validateTemplateForSave(template); err != nil {
		return "", err
	}
	if template.TemplateID == "" {
		template.TemplateID = uuid.New().String()
	}
	now := time.Now().UTC()
	template.CreatedAt = now
	template.UpdatedAt = now
	if template.Version == "" {
		template.Version = "1.0.0"
	}
	if template.Status == 0 {
		template.Status = types.WorkflowTemplateStatusDraft
	}
	if template.VisibleScope == 0 {
		template.VisibleScope = types.WorkflowTemplateVisibleScopeCompany
	}

	if err := s.repo.CreateWorkflowTemplate(template); err != nil {
		return "", fmt.Errorf("create workflow template: %w", err)
	}
	_ = s.auditLogs.RecordLog(NewAuditLog(
		types.AuditOpTemplateCreate, template.Author, "",
		WithDetailf("创建流程模板 %s", template.TemplateName),
	))
	return template.TemplateID, nil
}

func (s *templateServiceImpl) UpdateTemplate(template *types.WorkflowTemplate) error {
	if template.TemplateID == "" {
		return fmt.Errorf("template id is required")
	}
	if err := validateTemplateForSave(template); err != nil {
		return err
	}
	current, err := s.repo.GetWorkflowTemplate(template.TemplateID)
	if err != nil {
		return fmt.Errorf("get workflow template before update: %w", err)
	}

	template.Author = current.Author
	template.CreatedAt = current.CreatedAt
	template.InstallCount = current.InstallCount
	template.StartCount = current.StartCount
	template.UpdatedAt = time.Now().UTC()
	if template.Version == "" {
		template.Version = current.Version
	}

	if err = s.repo.UpdateWorkflowTemplate(template); err != nil {
		return fmt.Errorf("update workflow template: %w", err)
	}
	_ = s.auditLogs.RecordLog(NewAuditLog(
		types.AuditOpTemplateUpdate, current.Author, "",
		WithDetailf("更新流程模板 %s", template.TemplateName),
	))
	return nil
}

func (s *templateServiceImpl) PublishTemplate(templateID string) error {
	template, err := s.repo.GetWorkflowTemplate(templateID)
	if err != nil {
		return fmt.Errorf("get workflow template before publish: %w", err)
	}
	if template.WorkflowDef == nil {
		return fmt.Errorf("workflow template definition is required")
	}
	version := template.Version
	if version == "" {
		version = "1.0.0"
	}
	if err = s.repo.PublishWorkflowTemplate(templateID, version, time.Now().UTC()); err != nil {
		return fmt.Errorf("publish workflow template: %w", err)
	}
	_ = s.auditLogs.RecordLog(NewAuditLog(
		types.AuditOpTemplatePublish, template.Author, "",
		WithDetailf("发布流程模板 %s", template.TemplateName),
	))
	return nil
}

func (s *templateServiceImpl) InstallTemplate(templateID string, operator string) (*types.WorkflowTemplateInstallResult, error) {
	template, err := s.repo.GetWorkflowTemplate(templateID)
	if err != nil {
		return nil, fmt.Errorf("get workflow template before install: %w", err)
	}
	if template.WorkflowDef == nil {
		return nil, fmt.Errorf("workflow template definition is required")
	}

	defCopy, err := cloneWorkflowDef(template.WorkflowDef)
	if err != nil {
		return nil, err
	}
	defCopy.ID = ""
	defCopy.Name = template.TemplateName
	defCopy.Description = template.Description
	defCopy.CreatedBy = operator
	defCopy.UpdatedBy = operator
	defCopy.CreatedAt = time.Now().UTC()
	defCopy.UpdatedAt = defCopy.CreatedAt
	defCopy.Disabled = false

	workflowID, err := s.workflow.CreateWorkflow(defCopy)
	if err != nil {
		return nil, fmt.Errorf("install workflow template: %w", err)
	}
	if err = s.repo.IncrementWorkflowTemplateInstallCount(templateID); err != nil {
		return nil, fmt.Errorf("increment workflow template install count: %w", err)
	}
	_ = s.auditLogs.RecordLog(NewAuditLog(
		types.AuditOpTemplateInstall, operator, "",
		WithWorkflow(workflowID),
		WithDetailf("安装流程模板 %s", template.TemplateName),
	))
	return &types.WorkflowTemplateInstallResult{
		TemplateID: templateID,
		WorkflowID: workflowID,
	}, nil
}

func (s *templateServiceImpl) ImportTemplate(template *types.WorkflowTemplate) (string, error) {
	if err := validateTemplateForSave(template); err != nil {
		return "", err
	}
	if template.TemplateID == "" {
		template.TemplateID = uuid.New().String()
	}
	now := time.Now().UTC()
	template.CreatedAt = now
	template.UpdatedAt = now
	if template.Version == "" {
		template.Version = "1.0.0"
	}
	if template.Status == 0 {
		template.Status = types.WorkflowTemplateStatusDraft
	}
	if template.VisibleScope == 0 {
		template.VisibleScope = types.WorkflowTemplateVisibleScopeCompany
	}

	if err := s.repo.CreateWorkflowTemplate(template); err != nil {
		return "", fmt.Errorf("import workflow template: %w", err)
	}
	_ = s.auditLogs.RecordLog(NewAuditLog(
		types.AuditOpTemplateImport, template.Author, "",
		WithDetailf("导入流程模板 %s", template.TemplateName),
	))
	return template.TemplateID, nil
}

func (s *templateServiceImpl) ExportTemplate(templateID string) (*types.WorkflowTemplate, error) {
	template, err := s.repo.GetWorkflowTemplate(templateID)
	if err != nil {
		return nil, fmt.Errorf("export workflow template: %w", err)
	}
	_ = s.auditLogs.RecordLog(NewAuditLog(
		types.AuditOpTemplateExport, "", "",
		WithDetailf("导出流程模板 %s", template.TemplateName),
	))
	return template, nil
}

func (s *templateServiceImpl) ListCategories() ([]*types.WorkflowTemplateCategory, error) {
	return s.repo.ListWorkflowTemplateCategories()
}

func validateTemplateForSave(template *types.WorkflowTemplate) error {
	if template == nil {
		return fmt.Errorf("template is required")
	}
	if template.TemplateName == "" {
		return fmt.Errorf("template name is required")
	}
	if template.WorkflowDef == nil {
		return fmt.Errorf("workflow definition is required")
	}
	return nil
}

func cloneWorkflowDef(def *types.WorkflowDef) (*types.WorkflowDef, error) {
	data, err := json.Marshal(def)
	if err != nil {
		return nil, fmt.Errorf("marshal workflow definition: %w", err)
	}
	var copied types.WorkflowDef
	if err = json.Unmarshal(data, &copied); err != nil {
		return nil, fmt.Errorf("unmarshal workflow definition: %w", err)
	}
	return &copied, nil
}


