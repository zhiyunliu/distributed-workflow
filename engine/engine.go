package engine

import (
	"database/sql"
	"fmt"
	"net"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"

	"github.com/zhiyunliu/distributed-workflow/interfaces"
	"github.com/zhiyunliu/distributed-workflow/plugin"
	"github.com/zhiyunliu/distributed-workflow/rpc"
	"github.com/zhiyunliu/distributed-workflow/storage/sqlserver"
)

// dbProvider 定义了能够暴露底层 *sql.DB 的 repository 接口，供 D6 子仓库复用
type dbProvider interface {
	DB() *sql.DB
}

// D6ServiceInjector 定义了 HTTP 服务器注入 D6 服务的接口，避免 engine 直接依赖 endpoint/http
type D6ServiceInjector interface {
	SetD6FormService(interfaces.FormService)
	SetD6ApprovalService(interfaces.AdvancedApprovalService)
	SetD6AnalyticsService(interfaces.AnalyticsService)
	SetD6PluginService(interfaces.PluginService)
	SetD6OAuthService(interfaces.OAuthService)
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
	svc        interfaces.WorkflowService
	sched      interfaces.SchedulerService
	workerMgr  interfaces.WorkerManagerService
	state      *InstanceStateServiceImpl
	engineSrv  *EngineServer
	failover   interfaces.FailoverManager

	// D3 组件
	auditLogMgr interfaces.AuditLogManager
	approvalSvc interfaces.ApprovalService
	auditSvc    interfaces.AuditLogService
	endpointMgr interfaces.EndpointManagerService
	templateSvc interfaces.TemplateService

	// D6 组件
	formSvc        interfaces.FormService
	advApprovalSvc interfaces.AdvancedApprovalService
	analyticsSvc   interfaces.AnalyticsService
	pluginSvc      interfaces.PluginService
	oauthSvc       interfaces.OAuthService
}

// New 创建引擎实例
// repo: SQL Server 存储
// redisRepo: Redis 存储（分布式锁）
// redisClient: Redis 客户端（上下文快照 + failover 锁）
func New(
	cfg Config,
	repo interfaces.WorkflowRepository,
	redis interfaces.RedisRepository,
	redisClient *redis.Client,
) *Engine {
	logger := log.Logger

	// Worker gRPC 客户端
	workerClient := rpc.NewWorkerClient()

	// 依次创建各服务（注意循环依赖解耦）
	workerMgrImpl := NewWorkerManagerService(workerClient).(*WorkerManagerServiceImpl)

	stateImpl := NewInstanceStateService(repo)

	sched := NewSchedulerService(repo, stateImpl, workerMgrImpl, redis)

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

	// D6 组件（需要底层 *sql.DB；若 repo 实现了 dbProvider 则自动初始化）
	var (
		formSvc        interfaces.FormService
		advApprovalSvc interfaces.AdvancedApprovalService
		analyticsSvc   interfaces.AnalyticsService
		pluginSvc      interfaces.PluginService
		oauthSvc       interfaces.OAuthService
	)
	if dbProv, ok := repo.(dbProvider); ok {
		db := dbProv.DB()
		pluginRegistry := plugin.NewPluginRegistry()
		formSvc = NewFormService(sqlserver.NewFormRepository(db))
		advApprovalSvc = NewAdvancedApprovalService(sqlserver.NewAdvancedApprovalRepository(db))
		analyticsSvc = NewAnalyticsService(sqlserver.NewAnalyticsRepository(db))
		pluginSvc = NewPluginService(sqlserver.NewPluginRepository(db), pluginRegistry)
		oauthSvc = NewOAuthService(sqlserver.NewOAuthRepository(db))
	} else {
		log.Warn().Msg("repo 未实现 dbProvider，D6 服务将不可用")
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
func (e *Engine) WorkflowService() interfaces.WorkflowService {
	return e.svc
}

// SchedulerService 暴露调度服务
func (e *Engine) SchedulerService() interfaces.SchedulerService {
	return e.sched
}

// ApprovalService 暴露审批服务（D3）
func (e *Engine) ApprovalService() interfaces.ApprovalService {
	return e.approvalSvc
}

// AuditLogService 暴露审计日志服务（D3）
func (e *Engine) AuditLogService() interfaces.AuditLogService {
	return e.auditSvc
}

// EndpointManagerService 暴露端点管理服务（D3）
func (e *Engine) EndpointManagerService() interfaces.EndpointManagerService {
	return e.endpointMgr
}

// TemplateService 暴露模板服务（D5）
func (e *Engine) TemplateService() interfaces.TemplateService {
	return e.templateSvc
}

// FormService 暴露表单服务（D6）
func (e *Engine) FormService() interfaces.FormService {
	return e.formSvc
}

// AdvancedApprovalService 暴露高级审批服务（D6）
func (e *Engine) AdvancedApprovalService() interfaces.AdvancedApprovalService {
	return e.advApprovalSvc
}

// AnalyticsService 暴露 BI 统计服务（D6）
func (e *Engine) AnalyticsService() interfaces.AnalyticsService {
	return e.analyticsSvc
}

// PluginService 暴露插件管理服务（D6）
func (e *Engine) PluginService() interfaces.PluginService {
	return e.pluginSvc
}

// OAuthService 暴露 OAuth 服务（D6）
func (e *Engine) OAuthService() interfaces.OAuthService {
	return e.oauthSvc
}

// InjectD6Services 将 D6 服务注入支持 D6ServiceInjector 接口的 HTTP 服务器并注册路由。
// 调用方示例：eng.InjectD6Services(httpServer)
func (e *Engine) InjectD6Services(injector D6ServiceInjector) {
	if e.formSvc != nil {
		injector.SetD6FormService(e.formSvc)
	}
	if e.advApprovalSvc != nil {
		injector.SetD6ApprovalService(e.advApprovalSvc)
	}
	if e.analyticsSvc != nil {
		injector.SetD6AnalyticsService(e.analyticsSvc)
	}
	if e.pluginSvc != nil {
		injector.SetD6PluginService(e.pluginSvc)
	}
	if e.oauthSvc != nil {
		injector.SetD6OAuthService(e.oauthSvc)
	}
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
