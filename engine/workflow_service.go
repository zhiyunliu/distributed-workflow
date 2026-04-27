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
	repo  interfaces.WorkflowRepository
	sched interfaces.SchedulerService
	state interfaces.InstanceStateService
}

// NewWorkflowService 创建 WorkflowService 实例
func NewWorkflowService(
	repo interfaces.WorkflowRepository,
	sched interfaces.SchedulerService,
	state interfaces.InstanceStateService,
) interfaces.WorkflowService {
	return &WorkflowServiceImpl{repo: repo, sched: sched, state: state}
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

func (s *WorkflowServiceImpl) UpdateWorkflow(def *types.WorkflowDef) error {
	return s.repo.UpdateWorkflowDef(def)
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
