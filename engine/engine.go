package engine

import (
	"fmt"
	"net"

	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"

	"github.com/zhiyunliu/distributed-workflow/interfaces"
	"github.com/zhiyunliu/distributed-workflow/rpc"
)

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
}

// New 创建引擎实例
// repo: SQL Server 存储
// redis: Redis 存储
func New(
	cfg Config,
	repo interfaces.WorkflowRepository,
	redis interfaces.RedisRepository,
) *Engine {
	// Worker gRPC 连接池
	workerClient := rpc.NewWorkerClient()

	// 依次创建各服务（注意循环依赖解耦）
	workerMgr := NewWorkerManagerService(workerClient)

	stateImpl := NewInstanceStateService(repo)

	sched := NewSchedulerService(repo, stateImpl, workerMgr, redis)

	// 注入调度器（解决循环依赖）
	stateImpl.SetScheduler(sched)

	svc := NewWorkflowService(repo, sched, stateImpl)

	engineSrv := NewEngineServer(workerMgr, stateImpl)

	grpcServer := grpc.NewServer()
	engineSrv.RegisterServer(grpcServer)

	return &Engine{
		cfg:        cfg,
		grpcServer: grpcServer,
		svc:        svc,
		sched:      sched,
		workerMgr:  workerMgr,
		state:      stateImpl,
		engineSrv:  engineSrv,
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

// Start 启动引擎：监听 gRPC 端口 + 恢复未完成实例
func (e *Engine) Start() error {
	lis, err := net.Listen("tcp", e.cfg.GRPCAddr)
	if err != nil {
		return fmt.Errorf("listen '%s': %w", e.cfg.GRPCAddr, err)
	}
	e.listener = lis

	// 恢复未完成实例
	if err := e.sched.RecoverUnfinishedInstances(); err != nil {
		log.Error().Err(err).Msg("recover unfinished instances failed")
	}

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
	e.grpcServer.GracefulStop()
}
