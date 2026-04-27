// 分支工作流示例：A → (成功走 B，失败走 C) → D
//
// 此示例展示：
//   - ConnectionTypeSuccess 和 ConnectionTypeFailure 条件路由
//   - D 使用 DependencyModeAny，任意一条路径到达即可执行
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
	_ = container.Register(&executors.LogExecutor{})
	_ = container.Register(&executors.SleepExecutor{})

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

	// A → B（成功路径）→ D
	//   → C（失败路径）→ D
	// D 使用 AnyMode：B 或 C 任意一个完成即可触发 D
	def := &types.WorkflowDef{
		ID:          "branch-example",
		Name:        "分支示例工作流",
		StartNodeID: "node-a",
		EndNodeID:   "node-d",
		Nodes: map[string]*types.WorkflowNode{
			"node-a": {
				ID: "node-a", Type: "log", Name: "决策节点A",
				Config: map[string]interface{}{"message": "节点 A：根据结果路由"},
			},
			"node-b": {
				ID: "node-b", Type: "log", Name: "成功分支B",
				Config: map[string]interface{}{"message": "走成功路径 B"},
			},
			"node-c": {
				ID: "node-c", Type: "log", Name: "失败分支C",
				Config: map[string]interface{}{"message": "走失败路径 C"},
			},
			"node-d": {
				ID: "node-d", Type: "log", Name: "汇聚节点D",
				Config:         map[string]interface{}{"message": "任意分支到达，执行 D"},
				DependencyMode: types.DependencyModeAny,
			},
		},
		Connections: []*types.Connection{
			{ID: "c1", SourceID: "node-a", TargetID: "node-b", Type: types.ConnectionTypeSuccess},
			{ID: "c2", SourceID: "node-a", TargetID: "node-c", Type: types.ConnectionTypeFailure},
			{ID: "c3", SourceID: "node-b", TargetID: "node-d", Type: types.ConnectionTypeSuccess},
			{ID: "c4", SourceID: "node-c", TargetID: "node-d", Type: types.ConnectionTypeSuccess},
		},
	}

	svc := eng.WorkflowService()
	if _, err := svc.CreateWorkflow(def); err != nil {
		log.Error().Err(err).Msg("create workflow failed")
	}

	instanceID, err := svc.StartWorkflow("branch-example", map[string]interface{}{}, "demo-user")
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
