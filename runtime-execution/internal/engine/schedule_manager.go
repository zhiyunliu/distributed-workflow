package engine

import (
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/zhiyunliu/distributed-workflow/runtime-execution/pkg/types"
	api "github.com/zhiyunliu/distributed-workflow/runtime-execution/pkg/api"
	"github.com/zhiyunliu/distributed-workflow/runtime-execution/pkg/endpoint/schedule"
)

// scheduleEntry 定时端点调度条目
type scheduleEntry struct {
	endpoint *types.WorkflowEndpoint
	nextTime time.Time
}

// scheduleManagerImpl 定时端点调度管理器（内存实现，适合单机模式）
type scheduleManagerImpl struct {
	mu          sync.Mutex
	entries     map[string]*scheduleEntry // endpointID -> entry
	repo        api.WorkflowRepository
	workflowSvc api.WorkflowService
	stopCh      chan struct{}
	doneCh      chan struct{}
}

// newScheduleManager 创建调度管理器
func newScheduleManager(repo api.WorkflowRepository, workflowSvc api.WorkflowService) *scheduleManagerImpl {
	return &scheduleManagerImpl{
		entries:     make(map[string]*scheduleEntry),
		repo:        repo,
		workflowSvc: workflowSvc,
		stopCh:      make(chan struct{}),
		doneCh:      make(chan struct{}),
	}
}

// registerEndpoint 注册定时端点
func (m *scheduleManagerImpl) registerEndpoint(ep *types.WorkflowEndpoint) error {
	if ep.ScheduleConfig == nil || ep.ScheduleConfig.CronExpression == "" {
		return nil
	}
	nextTime, err := schedule.ParseNextTime(ep.ScheduleConfig.CronExpression, time.Now())
	if err != nil {
		return err
	}
	// 检查 StartTime 约束
	if ep.ScheduleConfig.StartTime != nil && nextTime.Before(*ep.ScheduleConfig.StartTime) {
		nextTime = *ep.ScheduleConfig.StartTime
	}
	m.mu.Lock()
	m.entries[ep.ID] = &scheduleEntry{endpoint: ep, nextTime: nextTime}
	m.mu.Unlock()
	log.Info().Str("endpointID", ep.ID).Time("nextTime", nextTime).Msg("schedule endpoint registered")
	return nil
}

// unregisterEndpoint 注销定时端点
func (m *scheduleManagerImpl) unregisterEndpoint(endpointID string) {
	m.mu.Lock()
	delete(m.entries, endpointID)
	m.mu.Unlock()
}

// start 启动调度循环
func (m *scheduleManagerImpl) start() {
	go m.run()
}

// stop 停止调度循环
func (m *scheduleManagerImpl) stop() {
	close(m.stopCh)
	<-m.doneCh
}

// run 调度主循环，每 10 秒检查一次（精度 10s）
func (m *scheduleManagerImpl) run() {
	defer close(m.doneCh)
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-m.stopCh:
			return
		case now := <-ticker.C:
			m.tick(now)
		}
	}
}

// tick 检查并触发到期的定时端点
func (m *scheduleManagerImpl) tick(now time.Time) {
	m.mu.Lock()
	var toTrigger []*scheduleEntry
	for _, e := range m.entries {
		if !now.Before(e.nextTime) {
			toTrigger = append(toTrigger, e)
		}
	}
	m.mu.Unlock()

	for _, entry := range toTrigger {
		ep := entry.endpoint
		cfg := ep.ScheduleConfig

		// 检查 EndTime 约束
		if cfg.EndTime != nil && now.After(*cfg.EndTime) {
			m.unregisterEndpoint(ep.ID)
			log.Info().Str("endpointID", ep.ID).Msg("schedule endpoint expired, unregistered")
			continue
		}

		// 检查最大触发次数
		if cfg.MaxTriggerCount > 0 && cfg.TriggeredCount >= cfg.MaxTriggerCount {
			m.unregisterEndpoint(ep.ID)
			log.Info().Str("endpointID", ep.ID).Msg("schedule endpoint reached max trigger count, unregistered")
			continue
		}

		// 触发工作流
		go m.triggerEndpoint(ep)

		// 计算下次时间
		nextTime, err := schedule.ParseNextTime(cfg.CronExpression, now)
		if err != nil {
			log.Error().Err(err).Str("endpointID", ep.ID).Msg("schedule parse next time failed")
			m.unregisterEndpoint(ep.ID)
			continue
		}
		m.mu.Lock()
		if e, ok := m.entries[ep.ID]; ok {
			e.nextTime = nextTime
		}
		m.mu.Unlock()
	}
}

// triggerEndpoint 触发端点（异步执行）
func (m *scheduleManagerImpl) triggerEndpoint(ep *types.WorkflowEndpoint) {
	cfg := ep.ScheduleConfig
	inputData := cfg.InputData
	if inputData == nil {
		inputData = map[string]interface{}{}
	}

	instanceID, err := m.workflowSvc.StartWorkflow(ep.WorkflowID, inputData, "endpoint:schedule")
	if err != nil {
		log.Error().Err(err).
			Str("endpointID", ep.ID).
			Str("workflowID", ep.WorkflowID).
			Msg("schedule trigger workflow failed")
		return
	}

	// 更新触发计数
	if err := m.repo.UpdateEndpointTriggeredCount(ep.ID, 1); err != nil {
		log.Warn().Err(err).Str("endpointID", ep.ID).Msg("update triggered count failed")
	}

	log.Info().
		Str("endpointID", ep.ID).
		Str("workflowID", ep.WorkflowID).
		Str("instanceID", instanceID).
		Msg("schedule endpoint triggered workflow")
}


