package worker

import (
	"fmt"
	"net"
	"time"

	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"

	"github.com/zhiyunliu/distributed-workflow/runtime-execution/internal/common/rpc"
	api "github.com/zhiyunliu/distributed-workflow/runtime-execution/pkg/api"
)

const (
	heartbeatInterval    = 5 * time.Second
	missedHeartbeatLimit = 3
)

// Config NodeWorker 配置
type Config struct {
	// WorkerID Worker 唯一标识
	WorkerID string
	// GRPCAddr Worker 自身 gRPC 监听地址，格式：ip:port
	GRPCAddr string
	// WorkerIP Worker 服务器 IP（用于 Engine 识别）
	WorkerIP string
	// EngineAddr Engine gRPC 地址
	EngineAddr string
	// Capabilities 支持的节点类型与最大并发数
	Capabilities map[string]int
}

// NodeWorker 分布式执行节点
type NodeWorker struct {
	cfg        Config
	container  api.NodeExecutorContainer
	taskMgr    api.TaskExecutionManager
	engineRPC  *rpc.EngineClient
	grpcServer *grpc.Server
	listener   net.Listener
	stopCh     chan struct{}
}

// New 创建 NodeWorker
func New(cfg Config, container api.NodeExecutorContainer) (*NodeWorker, error) {
	engineRPC, err := rpc.NewEngineClient(cfg.EngineAddr)
	if err != nil {
		return nil, fmt.Errorf("connect to engine: %w", err)
	}

	taskMgr := NewTaskManager(container, engineRPC, cfg.Capabilities)
	workerSrv := NewWorkerServer(taskMgr)
	grpcServer := grpc.NewServer()
	workerSrv.RegisterServer(grpcServer)

	return &NodeWorker{
		cfg:        cfg,
		container:  container,
		taskMgr:    taskMgr,
		engineRPC:  engineRPC,
		grpcServer: grpcServer,
		stopCh:     make(chan struct{}),
	}, nil
}

// Start 启动 Worker：监听 gRPC + 向 Engine 注册 + 启动心跳
func (w *NodeWorker) Start() error {
	lis, err := net.Listen("tcp", w.cfg.GRPCAddr)
	if err != nil {
		return fmt.Errorf("listen '%s': %w", w.cfg.GRPCAddr, err)
	}
	w.listener = lis

	log.Info().Str("addr", w.cfg.GRPCAddr).Str("worker_id", w.cfg.WorkerID).Msg("worker gRPC server starting")
	go func() {
		if err := w.grpcServer.Serve(lis); err != nil {
			log.Error().Err(err).Msg("worker gRPC server stopped")
		}
	}()

	// 向 Engine 注册
	if err := w.engineRPC.RegisterWorker(
		w.cfg.WorkerID,
		w.cfg.GRPCAddr,
		w.cfg.WorkerIP,
		w.cfg.Capabilities,
	); err != nil {
		return fmt.Errorf("register worker: %w", err)
	}
	log.Info().Str("worker_id", w.cfg.WorkerID).Msg("worker registered to engine")

	// 启动心跳 goroutine
	go w.heartbeatLoop()
	return nil
}

// Stop 优雅停止 Worker
func (w *NodeWorker) Stop() {
	close(w.stopCh)

	// 向 Engine 注销
	if err := w.engineRPC.UnregisterWorker(w.cfg.WorkerID); err != nil {
		log.Warn().Err(err).Str("worker_id", w.cfg.WorkerID).Msg("unregister worker failed")
	}

	w.grpcServer.GracefulStop()
	_ = w.engineRPC.Close()
	log.Info().Str("worker_id", w.cfg.WorkerID).Msg("worker stopped")
}

// RegisterExecutor 注册节点执行器（Start 之前调用）
func (w *NodeWorker) RegisterExecutor(executor api.NodeExecutor) error {
	return w.container.Register(executor)
}

// heartbeatLoop 定期向 Engine 发送心跳
func (w *NodeWorker) heartbeatLoop() {
	ticker := time.NewTicker(heartbeatInterval)
	defer ticker.Stop()
	for {
		select {
		case <-w.stopCh:
			return
		case <-ticker.C:
			load := w.taskMgr.GetCurrentLoad()
			if err := w.engineRPC.Heartbeat(w.cfg.WorkerID, load); err != nil {
				log.Warn().Err(err).Str("worker_id", w.cfg.WorkerID).Msg("heartbeat failed")
			}
		}
	}
}

