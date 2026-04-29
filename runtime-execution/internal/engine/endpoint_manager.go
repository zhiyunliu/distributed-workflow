package engine

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	api "github.com/zhiyunliu/distributed-workflow/runtime-execution/pkg/api"
	"github.com/zhiyunliu/distributed-workflow/runtime-execution/pkg/types"
)

// endpointManagerImpl 端点管理服务实现
type endpointManagerImpl struct {
	repo        api.WorkflowRepository
	auditMgr    api.AuditLogManager
	schedMgr    *scheduleManagerImpl
	workflowSvc api.WorkflowService
}

// NewEndpointManager 创建端点管理服务
func NewEndpointManager(
	repo api.WorkflowRepository,
	auditMgr api.AuditLogManager,
	workflowSvc api.WorkflowService,
) api.EndpointManagerService {
	mgr := &endpointManagerImpl{
		repo:        repo,
		auditMgr:    auditMgr,
		workflowSvc: workflowSvc,
	}
	mgr.schedMgr = newScheduleManager(repo, workflowSvc)
	return mgr
}

// CreateEndpoint 创建端点
func (m *endpointManagerImpl) CreateEndpoint(endpoint *types.WorkflowEndpoint) (string, error) {
	if endpoint.ID == "" {
		endpoint.ID = uuid.New().String()
	}
	now := time.Now().UTC()
	endpoint.CreatedAt = now
	endpoint.UpdatedAt = now

	if err := m.repo.CreateEndpoint(endpoint); err != nil {
		return "", fmt.Errorf("create endpoint: %w", err)
	}

	_ = m.auditMgr.RecordLog(NewAuditLog(
		types.AuditOpEndpointCreate, endpoint.CreatedBy, "",
		WithEndpoint(endpoint.ID), WithWorkflow(endpoint.WorkflowID),
		WithDetailf("创建端点 %s (type=%s)", endpoint.Name, endpoint.Type),
	))
	return endpoint.ID, nil
}

// UpdateEndpoint 更新端点
func (m *endpointManagerImpl) UpdateEndpoint(endpoint *types.WorkflowEndpoint) error {
	endpoint.UpdatedAt = time.Now().UTC()
	if err := m.repo.UpdateEndpoint(endpoint); err != nil {
		return fmt.Errorf("update endpoint: %w", err)
	}
	_ = m.auditMgr.RecordLog(NewAuditLog(
		types.AuditOpEndpointUpdate, "", "",
		WithEndpoint(endpoint.ID), WithWorkflow(endpoint.WorkflowID),
		WithDetailf("更新端点 %s", endpoint.Name),
	))
	return nil
}

// GetEndpoint 获取端点详情
func (m *endpointManagerImpl) GetEndpoint(endpointID string) (*types.WorkflowEndpoint, error) {
	return m.repo.GetEndpoint(endpointID)
}

// ListEndpoints 查询端点列表
func (m *endpointManagerImpl) ListEndpoints(filter types.EndpointFilter, page, pageSize int) ([]*types.WorkflowEndpoint, int64, error) {
	return m.repo.ListEndpoints(filter, page, pageSize)
}

// StartEndpoint 启用端点
func (m *endpointManagerImpl) StartEndpoint(endpointID string) error {
	ep, err := m.repo.GetEndpoint(endpointID)
	if err != nil {
		return err
	}
	if !ep.Disabled {
		return nil // 已启用
	}
	ep.Disabled = false
	ep.UpdatedAt = time.Now().UTC()
	if err := m.repo.UpdateEndpoint(ep); err != nil {
		return fmt.Errorf("start endpoint: %w", err)
	}

	// 如果是定时端点，注册到调度器
	if ep.Type == types.EndpointTypeSchedule && ep.ScheduleConfig != nil {
		if err := m.schedMgr.registerEndpoint(ep); err != nil {
			log.Warn().Err(err).Str("endpointID", endpointID).Msg("register schedule endpoint failed")
		}
	}

	_ = m.auditMgr.RecordLog(NewAuditLog(
		types.AuditOpEndpointStart, "", "",
		WithEndpoint(endpointID),
		WithDetailf("启用端点 %s", ep.Name),
	))
	return nil
}

// StopEndpoint 禁用端点
func (m *endpointManagerImpl) StopEndpoint(endpointID string) error {
	ep, err := m.repo.GetEndpoint(endpointID)
	if err != nil {
		return err
	}
	if ep.Disabled {
		return nil // 已禁用
	}
	ep.Disabled = true
	ep.UpdatedAt = time.Now().UTC()
	if err := m.repo.UpdateEndpoint(ep); err != nil {
		return fmt.Errorf("stop endpoint: %w", err)
	}

	// 如果是定时端点，从调度器注销
	if ep.Type == types.EndpointTypeSchedule {
		m.schedMgr.unregisterEndpoint(endpointID)
	}

	_ = m.auditMgr.RecordLog(NewAuditLog(
		types.AuditOpEndpointStop, "", "",
		WithEndpoint(endpointID),
		WithDetailf("禁用端点 %s", ep.Name),
	))
	return nil
}

// DeleteEndpoint 删除端点
func (m *endpointManagerImpl) DeleteEndpoint(endpointID string) error {
	ep, err := m.repo.GetEndpoint(endpointID)
	if err != nil {
		return err
	}

	// 先停止
	if ep.Type == types.EndpointTypeSchedule {
		m.schedMgr.unregisterEndpoint(endpointID)
	}

	if err := m.repo.DeleteEndpoint(endpointID); err != nil {
		return fmt.Errorf("delete endpoint: %w", err)
	}
	_ = m.auditMgr.RecordLog(NewAuditLog(
		types.AuditOpEndpointDelete, "", "",
		WithEndpoint(endpointID),
		WithDetailf("删除端点 %s", ep.Name),
	))
	return nil
}

// TriggerEndpoint 手动触发端点（测试用）
func (m *endpointManagerImpl) TriggerEndpoint(endpointID string, inputData map[string]interface{}) (string, error) {
	ep, err := m.repo.GetEndpoint(endpointID)
	if err != nil {
		return "", err
	}
	if ep.Disabled {
		return "", fmt.Errorf("endpoint %s is disabled", endpointID)
	}
	instanceID, err := m.workflowSvc.StartWorkflow(ep.WorkflowID, inputData, "endpoint:trigger")
	if err != nil {
		return "", fmt.Errorf("trigger endpoint start workflow: %w", err)
	}
	_ = m.auditMgr.RecordLog(NewAuditLog(
		types.AuditOpEndpointTrigger, "", "",
		WithEndpoint(endpointID), WithWorkflow(ep.WorkflowID), WithInstance(instanceID),
		WithDetailf("手动触发端点 %s，启动实例 %s", ep.Name, instanceID),
	))
	return instanceID, nil
}

// StartScheduleManager 启动定时调度器
func (m *endpointManagerImpl) StartScheduleManager() error {
	// 加载所有未禁用的 schedule 端点
	disabled := false
	eps, _, err := m.repo.ListEndpoints(types.EndpointFilter{
		Type:     types.EndpointTypeSchedule,
		Disabled: &disabled,
	}, 1, 1000)
	if err != nil {
		return fmt.Errorf("load schedule endpoints: %w", err)
	}
	for _, ep := range eps {
		if ep.ScheduleConfig != nil {
			if regErr := m.schedMgr.registerEndpoint(ep); regErr != nil {
				log.Warn().Err(regErr).Str("endpointID", ep.ID).Msg("register schedule endpoint failed on start")
			}
		}
	}
	m.schedMgr.start()
	return nil
}

// StopScheduleManager 停止定时调度器
func (m *endpointManagerImpl) StopScheduleManager() error {
	m.schedMgr.stop()
	return nil
}


