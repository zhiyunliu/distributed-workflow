package engine

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/zhiyunliu/distributed-workflow/interfaces"
	"github.com/zhiyunliu/distributed-workflow/types"
)

// WorkflowServiceImpl 实现 interfaces.WorkflowService
type WorkflowServiceImpl struct {
	repo        interfaces.WorkflowRepository
	sched       interfaces.SchedulerService
	state       interfaces.InstanceStateService
	versionMgr  interfaces.WorkflowVersionManager
	lifecycleMgr interfaces.LifecycleManager
}

// NewWorkflowService 创建 WorkflowService 实例
func NewWorkflowService(
	repo interfaces.WorkflowRepository,
	sched interfaces.SchedulerService,
	state interfaces.InstanceStateService,
	versionMgr interfaces.WorkflowVersionManager,
	lifecycleMgr interfaces.LifecycleManager,
) interfaces.WorkflowService {
	return &WorkflowServiceImpl{
		repo:         repo,
		sched:        sched,
		state:        state,
		versionMgr:   versionMgr,
		lifecycleMgr: lifecycleMgr,
	}
}

func (s *WorkflowServiceImpl) CreateWorkflow(def *types.WorkflowDef) (string, error) {
	if def.ID == "" {
		def.ID = uuid.New().String()
	}
	if err := s.repo.CreateWorkflowDef(def); err != nil {
		return "", fmt.Errorf("create workflow def: %w", err)
	}
	log.Info().Str("workflow_id", def.ID).Msg("workflow created")
	return def.ID, nil
}

func (s *WorkflowServiceImpl) GetWorkflow(workflowID string) (*types.WorkflowDef, error) {
	return s.repo.GetWorkflowDef(workflowID)
}

func (s *WorkflowServiceImpl) UpdateWorkflow(def *types.WorkflowDef) (int, error) {
	if err := s.repo.UpdateWorkflowDef(def); err != nil {
		return 0, fmt.Errorf("update workflow def: %w", err)
	}
	// 自动创建新版本快照
	ver, err := s.versionMgr.CreateVersion(def.ID, def, "workflow updated", def.CreatedBy)
	if err != nil {
		// 版本创建失败不阻断更新（记录日志即可）
		log.Error().Err(err).Str("workflow_id", def.ID).Msg("create version snapshot failed")
		return 0, nil
	}
	return ver.Version, nil
}

func (s *WorkflowServiceImpl) DeleteWorkflow(workflowID string) error {
	return s.repo.DeleteWorkflowDef(workflowID)
}

// StartWorkflow 创建并启动一个工作流实例
func (s *WorkflowServiceImpl) StartWorkflow(workflowID string, inputData map[string]interface{}, createdBy string) (string, error) {
	def, err := s.repo.GetWorkflowDef(workflowID)
	if err != nil {
		return "", fmt.Errorf("get workflow def '%s': %w", workflowID, err)
	}
	if def.Disabled {
		return "", fmt.Errorf("workflow '%s' is disabled", workflowID)
	}

	instance := &types.WorkflowInstance{
		ID:         uuid.New().String(),
		WorkflowID: workflowID,
		Status:     types.WorkflowStatusPending,
		InputData:  inputData,
		CreatedBy:  createdBy,
	}

	if err := s.state.CreateInstance(instance); err != nil {
		return "", fmt.Errorf("create workflow instance: %w", err)
	}

	if err := s.sched.StartExecution(instance.ID); err != nil {
		log.Error().Err(err).Str("instance_id", instance.ID).Msg("start execution failed")
		return instance.ID, fmt.Errorf("start execution: %w", err)
	}

	log.Info().Str("workflow_id", workflowID).Str("instance_id", instance.ID).Msg("workflow started")
	return instance.ID, nil
}

func (s *WorkflowServiceImpl) GetWorkflowInstance(instanceID string) (*types.WorkflowInstance, error) {
	return s.repo.GetWorkflowInstance(instanceID)
}

// ─────────────────────────────────────────────────────────────────────────────
// D2 新增：版本管理
// ─────────────────────────────────────────────────────────────────────────────

func (s *WorkflowServiceImpl) GetWorkflowVersion(workflowID string, version int) (*types.WorkflowVersion, error) {
	return s.versionMgr.GetVersion(workflowID, version)
}

func (s *WorkflowServiceImpl) ListWorkflowVersions(workflowID string) ([]*types.WorkflowVersion, error) {
	return s.versionMgr.ListVersions(workflowID)
}

// RollbackWorkflow 回滚到指定版本：基于目标版本定义创建新版本并设为当前版本
func (s *WorkflowServiceImpl) RollbackWorkflow(workflowID string, targetVersion int, operator string) (int, error) {
	ver, err := s.versionMgr.GetVersion(workflowID, targetVersion)
	if err != nil {
		return 0, fmt.Errorf("get target version %d: %w", targetVersion, err)
	}
	newVer, err := s.versionMgr.CreateVersion(workflowID, ver.Definition, fmt.Sprintf("rollback to v%d by %s", targetVersion, operator), operator)
	if err != nil {
		return 0, fmt.Errorf("create rollback version: %w", err)
	}
	// 同步更新工作流定义
	if err := s.repo.UpdateWorkflowDef(ver.Definition); err != nil {
		log.Error().Err(err).Str("workflow_id", workflowID).Msg("sync rollback def failed")
	}
	return newVer.Version, nil
}

func (s *WorkflowServiceImpl) UpdateGrayConfig(workflowID string, version int, cfg *types.GrayReleaseConfig) error {
	return s.repo.UpdateVersionGrayConfig(workflowID, version, cfg)
}

// ─────────────────────────────────────────────────────────────────────────────
// D2 新增：生命周期管理
// ─────────────────────────────────────────────────────────────────────────────

func (s *WorkflowServiceImpl) PauseInstance(instanceID string, operator string) error {
	if err := s.lifecycleMgr.Pause(instanceID, operator); err != nil {
		return err
	}
	return s.sched.PauseExecution(instanceID)
}

func (s *WorkflowServiceImpl) ResumeInstance(instanceID string, operator string) error {
	if err := s.lifecycleMgr.Resume(instanceID, operator); err != nil {
		return err
	}
	return s.sched.ResumeExecution(instanceID)
}

func (s *WorkflowServiceImpl) CancelInstance(instanceID string, operator string) error {
	if err := s.lifecycleMgr.Cancel(instanceID, operator); err != nil {
		return err
	}
	return s.sched.CancelExecution(instanceID)
}

func (s *WorkflowServiceImpl) RetryInstance(instanceID string, operator string) error {
	if err := s.lifecycleMgr.Retry(instanceID, operator); err != nil {
		return err
	}
	return s.sched.RetryExecution(instanceID)
}

func (s *WorkflowServiceImpl) RetryNode(instanceID string, nodeID string, operator string) error {
	if err := s.lifecycleMgr.RetryNode(instanceID, nodeID, operator); err != nil {
		return err
	}
	return s.sched.RetrySingleNode(instanceID, nodeID)
}

func (s *WorkflowServiceImpl) ListWorkflowInstances(workflowID string, status types.WorkflowStatus, pageSize, pageNum int) ([]*types.WorkflowInstance, error) {
	return s.repo.ListWorkflowInstances(workflowID, status, pageSize, pageNum)
}

func (s *WorkflowServiceImpl) ListDeadLetterTasks(instanceID string) ([]*types.DeadLetterTask, error) {
	return s.repo.ListDeadLetterTasks(instanceID)
}
