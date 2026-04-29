package engine

import (
	"fmt"
	"net"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"

	"github.com/zhiyunliu/distributed-workflow/runtime-execution/internal/common/plugin"
	"github.com/zhiyunliu/distributed-workflow/runtime-execution/internal/common/rpc"
	api "github.com/zhiyunliu/distributed-workflow/runtime-execution/pkg/api"
)

// D6ServiceInjector 定义了 HTTP 服务器注入 D6 服务的接口，避免 engine 直接依赖 endpoint/http
type D6ServiceInjector interface {
	SetD6FormService(api.FormService)
	SetD6ApprovalService(api.AdvancedApprovalService)
	SetD6AnalyticsService(api.AnalyticsService)
	SetD6PluginService(api.PluginService)
	SetD6OAuthService(api.OAuthService)
}

// Config 引擎配置
type Config struct {
	// GRPCAddr gRPC 服务监听地址，格式：ip:port
	GRPCAddr string
}

// Engine 工作流引擎，组装所有服务并管理生命周期
type Engine struct {
	cfg        Config
	grpcServer *grpc.Server
	listener   net.Listener
	svc        api.WorkflowService
	sched      api.SchedulerService
	workerMgr  api.WorkerManagerService
	state      *InstanceStateServiceImpl
	engineSrv  *EngineServer
	failover   api.FailoverManager

	// D3 组件
	auditLogMgr api.AuditLogManager
	approvalSvc api.ApprovalService
	auditSvc    api.AuditLogService
	endpointMgr api.EndpointManagerService
	templateSvc api.TemplateService

	// D6 组件
	formSvc        api.FormService
	advApprovalSvc api.AdvancedApprovalService
	analyticsSvc   api.AnalyticsService
	pluginSvc      api.PluginService
	oauthSvc       api.OAuthService
}

// New 创建引擎实例
// repo: 工作流主仓库
// redis: Redis 存储（分布式锁）
// redisClient: Redis 客户端（上下文快照 + failover 锁）
// D6 repos: 各 D6 子模块仓库接口，由调用方注入（通常来自 storage/sqlserver 实现）
func New(
	cfg Config,
	repo api.WorkflowRepository,
	redisRepo api.RedisRepository,
	redisClient *redis.Client,
	formRepo api.FormRepository,
	advApprovalRepo api.AdvancedApprovalRepository,
	analyticsRepo api.AnalyticsRepository,
	pluginRepo api.PluginRepository,
	oauthRepo api.OAuthRepository,
) *Engine {
	logger := log.Logger

	// Worker gRPC 客户端
	workerClient := rpc.NewWorkerClient()

	// 依次创建各服务（注意循环依赖解耦）
	workerMgrImpl := NewWorkerManagerService(workerClient).(*WorkerManagerServiceImpl)

	stateImpl := NewInstanceStateService(repo)

	sched := NewSchedulerService(repo, stateImpl, workerMgrImpl, redisRepo)

	// 注入调度器（解决循环依赖）
	stateImpl.SetScheduler(sched)

	// D2 新增组件
	versionMgr := NewVersionManager(repo)
	lifecycleMgr := NewLifecycleManager(repo, stateImpl)

	svc := NewWorkflowService(repo, sched, stateImpl, versionMgr, lifecycleMgr)

	failoverMgr := NewFailoverManager(workerMgrImpl, repo, stateImpl, workerClient, redisClient, logger)

	engineSrv := NewEngineServer(workerMgrImpl, stateImpl)

	// D3 组件
	auditLogMgr := NewAuditLogManager(repo)
	auditSvc := NewAuditLogService(auditLogMgr)
	approvalSvcImpl := NewApprovalService(repo, stateImpl, auditLogMgr).(*approvalServiceImpl)
	approvalSvcImpl.SetScheduler(sched)
	endpointMgr := NewEndpointManager(repo, auditLogMgr, svc)
	templateSvc := NewTemplateService(repo, svc, auditLogMgr)

	grpcServer := grpc.NewServer()
	engineSrv.RegisterServer(grpcServer)

	// D6 组件：仓库由调用方注入，engine 不直接依赖 storage/sqlserver
	pluginRegistry := plugin.NewPluginRegistry()
	var (
		formSvc        api.FormService
		advApprovalSvc api.AdvancedApprovalService
		analyticsSvc   api.AnalyticsService
		pluginSvc      api.PluginService
		oauthSvc       api.OAuthService
	)
	if formRepo != nil {
		formSvc = NewFormService(formRepo)
	}
	if advApprovalRepo != nil {
		advApprovalSvc = NewAdvancedApprovalService(advApprovalRepo)
	}
	if analyticsRepo != nil {
		analyticsSvc = NewAnalyticsService(analyticsRepo)
	}
	if pluginRepo != nil {
		pluginSvc = NewPluginService(pluginRepo, pluginRegistry)
	}
	if oauthRepo != nil {
		oauthSvc = NewOAuthService(oauthRepo)
	}
	if formRepo == nil && advApprovalRepo == nil && analyticsRepo == nil && pluginRepo == nil && oauthRepo == nil {
		log.Warn().Msg("D6 repos 均未注入，D6 服务将不可用")
	}

	return &Engine{
		cfg:        cfg,
		grpcServer: grpcServer,
		svc:        svc,
		sched:      sched,
		workerMgr:  workerMgrImpl,
		state:      stateImpl,
		engineSrv:  engineSrv,
		failover:   failoverMgr,

		// D3
		auditLogMgr: auditLogMgr,
		approvalSvc: approvalSvcImpl,
		auditSvc:    auditSvc,
		endpointMgr: endpointMgr,
		templateSvc: templateSvc,

		// D6
		formSvc:        formSvc,
		advApprovalSvc: advApprovalSvc,
		analyticsSvc:   analyticsSvc,
		pluginSvc:      pluginSvc,
		oauthSvc:       oauthSvc,
	}
}

// WorkflowService 暴露工作流服务供 HTTP 层使用
func (e *Engine) WorkflowService() api.WorkflowService {
	return e.svc
}

// SchedulerService 暴露调度服务
func (e *Engine) SchedulerService() api.SchedulerService {
	return e.sched
}

// ApprovalService 暴露审批服务（D3）
func (e *Engine) ApprovalService() api.ApprovalService {
	return e.approvalSvc
}

// AuditLogService 暴露审计日志服务（D3）
func (e *Engine) AuditLogService() api.AuditLogService {
	return e.auditSvc
}

// EndpointManagerService 暴露端点管理服务（D3）
func (e *Engine) EndpointManagerService() api.EndpointManagerService {
	return e.endpointMgr
}

// TemplateService 暴露模板服务（D5）
func (e *Engine) TemplateService() api.TemplateService {
	return e.templateSvc
}

// FormService 暴露表单服务（D6）
func (e *Engine) FormService() api.FormService {
	return e.formSvc
}

// AdvancedApprovalService 暴露高级审批服务（D6）
func (e *Engine) AdvancedApprovalService() api.AdvancedApprovalService {
	return e.advApprovalSvc
}

// AnalyticsService 暴露 BI 统计服务（D6）
func (e *Engine) AnalyticsService() api.AnalyticsService {
	return e.analyticsSvc
}

// PluginService 暴露插件服务（D6）
func (e *Engine) PluginService() api.PluginService {
	return e.pluginSvc
}

// OAuthService 暴露 OAuth 服务（D6）
func (e *Engine) OAuthService() api.OAuthService {
	return e.oauthSvc
}

// InjectD6Services 将 D6 服务注入 HTTP 服务器（如果实现了 D6ServiceInjector 接口）
func (e *Engine) InjectD6Services(srv D6ServiceInjector) {
	srv.SetD6FormService(e.formSvc)
	srv.SetD6ApprovalService(e.advApprovalSvc)
	srv.SetD6AnalyticsService(e.analyticsSvc)
	srv.SetD6PluginService(e.pluginSvc)
	srv.SetD6OAuthService(e.oauthSvc)
}

// Start 启动引擎：监听 gRPC 端口 + 恢复未完成实例 + 启动健康检查
func (e *Engine) Start() error {
	lis, err := net.Listen("tcp", e.cfg.GRPCAddr)
	if err != nil {
		return fmt.Errorf("listen '%s': %w", e.cfg.GRPCAddr, err)
	}
	e.listener = lis

	// 启动 D3 审计日志管理器
	e.auditLogMgr.Start()

	// 启动 D3 定时端点调度器
	if err := e.endpointMgr.StartScheduleManager(); err != nil {
		log.Error().Err(err).Msg("start schedule manager failed")
	}

	// 恢复未完成实例
	if err := e.sched.RecoverUnfinishedInstances(); err != nil {
		log.Error().Err(err).Msg("recover unfinished instances failed")
	}

	// 启动 Worker 健康检查循环
	e.failover.StartHealthCheckLoop()

	log.Info().Str("addr", e.cfg.GRPCAddr).Msg("engine gRPC server starting")
	go func() {
		if err := e.grpcServer.Serve(lis); err != nil {
			log.Error().Err(err).Msg("engine gRPC server stopped")
		}
	}()
	return nil
}

// Stop 优雅停止引擎
func (e *Engine) Stop() {
	log.Info().Msg("engine stopping")
	e.failover.StopHealthCheckLoop()
	_ = e.endpointMgr.StopScheduleManager()
	e.auditLogMgr.Stop()
	e.grpcServer.GracefulStop()
}

// suppress unused import warning
var _ zerolog.Logger

