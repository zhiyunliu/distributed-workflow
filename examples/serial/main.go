// 串行工作流示例：A → B → C
//
// 运行前需要：
//  1. 启动 SQL Server，执行 storage/sqlserver/schema.sql
//  2. 启动 Redis
//  3. 修改下方配置常量
//
// 此示例展示如何：
//   - 构建 Engine 和 Worker
//   - 注册内置执行器
//   - 定义串行工作流
//   - 启动工作流实例
package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	endpointhttp "github.com/zhiyunliu/distributed-workflow/endpoint/http"
	"github.com/zhiyunliu/distributed-workflow/engine"
	"github.com/zhiyunliu/distributed-workflow/storage/redis"
	"github.com/zhiyunliu/distributed-workflow/storage/sqlserver"
	"github.com/zhiyunliu/distributed-workflow/types"
	"github.com/zhiyunliu/distributed-workflow/worker"
	"github.com/zhiyunliu/distributed-workflow/worker/executors"
)

const (
	engineGRPCAddr = "0.0.0.0:9090"
	workerGRPCAddr = "0.0.0.0:9091"
	httpAddr       = "0.0.0.0:8080"

	sqlServerDSN = "sqlserver://sa:YourPassword@localhost:1433?database=distributed_workflow"
	redisAddr    = "localhost:6379"
)

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	// ─── 存储层 ────────────────────────────────────────────────────────────────

	repo, err := sqlserver.NewRepository(sqlserver.Config{DSN: sqlServerDSN})
	if err != nil {
		log.Fatal().Err(err).Msg("connect sql server failed")
	}

	redisClient, err := redis.NewClient(redis.Config{Addr: redisAddr})
	if err != nil {
		log.Fatal().Err(err).Msg("connect redis failed")
	}

	// ─── Engine ────────────────────────────────────────────────────────────────

	eng := engine.New(engine.Config{GRPCAddr: engineGRPCAddr}, repo, redisClient, redisClient.RawClient())
	if err := eng.Start(); err != nil {
		log.Fatal().Err(err).Msg("start engine failed")
	}

	// ─── Worker ────────────────────────────────────────────────────────────────

	container := worker.NewExecutorContainer()
	_ = container.Register(&executors.LogExecutor{})
	_ = container.Register(&executors.SleepExecutor{})
	_ = container.Register(executors.NewHTTPExecutor())

	w, err := worker.New(worker.Config{
		WorkerID:   "worker-1",
		GRPCAddr:   workerGRPCAddr,
		WorkerIP:   "127.0.0.1",
		EngineAddr: "127.0.0.1:9090",
		Capabilities: map[string]int{
			"log":   10,
			"sleep": 5,
			"http":  5,
		},
	}, container)
	if err != nil {
		log.Fatal().Err(err).Msg("create worker failed")
	}
	if err := w.Start(); err != nil {
		log.Fatal().Err(err).Msg("start worker failed")
	}

	// ─── HTTP 服务 ─────────────────────────────────────────────────────────────

	httpServer := endpointhttp.NewServer(eng.WorkflowService())
	if err := httpServer.Start(httpAddr); err != nil {
		log.Fatal().Err(err).Msg("start http server failed")
	}

	// ─── 创建并启动串行工作流 ──────────────────────────────────────────────────

	time.Sleep(500 * time.Millisecond) // 等待组件就绪

	def := &types.WorkflowDef{
		ID:          "serial-example",
		Name:        "串行示例工作流 A→B→C",
		StartNodeID: "node-a",
		EndNodeID:   "node-c",
		Nodes: map[string]*types.WorkflowNode{
			"node-a": {ID: "node-a", Type: "log", Name: "步骤A", Config: map[string]interface{}{"message": "执行步骤 A"}},
			"node-b": {ID: "node-b", Type: "sleep", Name: "步骤B", Config: map[string]interface{}{"duration_ms": 500}},
			"node-c": {ID: "node-c", Type: "log", Name: "步骤C", Config: map[string]interface{}{"message": "执行步骤 C"}},
		},
		Connections: []*types.Connection{
			{ID: "c1", SourceID: "node-a", TargetID: "node-b", Type: types.ConnectionTypeSuccess},
			{ID: "c2", SourceID: "node-b", TargetID: "node-c", Type: types.ConnectionTypeSuccess},
		},
	}

	svc := eng.WorkflowService()
	if _, err := svc.CreateWorkflow(def); err != nil {
		log.Error().Err(err).Msg("create workflow failed")
	}

	instanceID, err := svc.StartWorkflow("serial-example", map[string]interface{}{"env": "demo"}, "demo-user")
	if err != nil {
		log.Error().Err(err).Msg("start workflow failed")
	} else {
		fmt.Printf("workflow started, instance_id: %s\n", instanceID)
	}

	// ─── 等待退出信号 ──────────────────────────────────────────────────────────

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info().Msg("shutting down...")
	w.Stop()
	eng.Stop()
	_ = httpServer.Stop()
}
