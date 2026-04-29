// 并行工作流示例：A → (B, C 并行执行) → D
//
// 此示例展示：
//   - B 和 C 在 A 完成后并发执行
//   - D 使用 DependencyModeAll，等待 B 和 C 全部完成后才执行
package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/zhiyunliu/distributed-workflow/runtime-execution/pkg/types"
	"github.com/zhiyunliu/distributed-workflow/runtime-execution/internal/engine"
	"github.com/zhiyunliu/distributed-workflow/runtime-execution/internal/storage/redis"
	"github.com/zhiyunliu/distributed-workflow/runtime-execution/pkg/storage/sqlserver"
	"github.com/zhiyunliu/distributed-workflow/runtime-execution/internal/worker"
	endpointhttp "github.com/zhiyunliu/distributed-workflow/runtime-execution/pkg/endpoint/http"
	"github.com/zhiyunliu/distributed-workflow/runtime-execution/pkg/node"
)

const (
	engineGRPCAddr = "0.0.0.0:9090"
	workerGRPCAddr = "0.0.0.0:9091"
	httpAddr       = "0.0.0.0:8080"
	sqlServerDSN   = "sqlserver://sa:YourPassword@localhost:1433?database=distributed_workflow"
	redisAddr      = "localhost:6379"
)

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	repo, err := sqlserver.NewRepository(sqlserver.Config{DSN: sqlServerDSN})
	if err != nil {
		log.Fatal().Err(err).Msg("connect sql server failed")
	}
	redisClient, err := redis.NewClient(redis.Config{Addr: redisAddr})
	if err != nil {
		log.Fatal().Err(err).Msg("connect redis failed")
	}

	eng := engine.New(engine.Config{GRPCAddr: engineGRPCAddr}, repo, redisClient, redisClient.RawClient())
	if err := eng.Start(); err != nil {
		log.Fatal().Err(err).Msg("start engine failed")
	}

	container := worker.NewExecutorContainer()
	_ = container.Register(&node.LogExecutor{})
	_ = container.Register(&node.SleepExecutor{})

	w, err := worker.New(worker.Config{
		WorkerID:     "worker-1",
		GRPCAddr:     workerGRPCAddr,
		WorkerIP:     "127.0.0.1",
		EngineAddr:   "127.0.0.1:9090",
		Capabilities: map[string]int{"log": 10, "sleep": 5},
	}, container)
	if err != nil {
		log.Fatal().Err(err).Msg("create worker failed")
	}
	if err := w.Start(); err != nil {
		log.Fatal().Err(err).Msg("start worker failed")
	}

	httpServer := endpointhttp.NewServer(eng.WorkflowService())
	_ = httpServer.Start(httpAddr)

	time.Sleep(500 * time.Millisecond)

	// A → (B | C 并行) → D（全量依赖）
	def := &types.WorkflowDef{
		ID:          "parallel-example",
		Name:        "并行示例工作流 A→(B|C)→D",
		StartNodeID: "node-a",
		EndNodeID:   "node-d",
		Nodes: map[string]*types.WorkflowNode{
			"node-a": {ID: "node-a", Type: "log", Name: "步骤A",
				Config: map[string]interface{}{"message": "步骤 A 完成，触发并行执行"}},
			"node-b": {ID: "node-b", Type: "sleep", Name: "步骤B（并行）",
				Config: map[string]interface{}{"duration_ms": 800}},
			"node-c": {ID: "node-c", Type: "sleep", Name: "步骤C（并行）",
				Config: map[string]interface{}{"duration_ms": 500}},
			"node-d": {
				ID: "node-d", Type: "log", Name: "步骤D（汇聚）",
				Config:         map[string]interface{}{"message": "B 和 C 均完成，执行 D"},
				DependencyMode: types.DependencyModeAll,
			},
		},
		Connections: []*types.Connection{
			{ID: "c1", SourceID: "node-a", TargetID: "node-b", Type: types.ConnectionTypeSuccess},
			{ID: "c2", SourceID: "node-a", TargetID: "node-c", Type: types.ConnectionTypeSuccess},
			{ID: "c3", SourceID: "node-b", TargetID: "node-d", Type: types.ConnectionTypeSuccess},
			{ID: "c4", SourceID: "node-c", TargetID: "node-d", Type: types.ConnectionTypeSuccess},
		},
	}

	svc := eng.WorkflowService()
	if _, err := svc.CreateWorkflow(def); err != nil {
		log.Error().Err(err).Msg("create workflow failed")
	}

	instanceID, err := svc.StartWorkflow("parallel-example", map[string]interface{}{}, "demo-user")
	if err != nil {
		log.Error().Err(err).Msg("start workflow failed")
	} else {
		fmt.Printf("workflow started, instance_id: %s\n", instanceID)
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	w.Stop()
	eng.Stop()
	_ = httpServer.Stop()
}


