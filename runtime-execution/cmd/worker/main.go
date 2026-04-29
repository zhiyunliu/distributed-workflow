// Package main NodeWorker 入口
//
// 环境变量：
//   - ENGINE_ADDR  Engine gRPC 地址（必填）
//   - WORKER_ID    Worker 唯一标识（默认 worker-1）
//   - WORKER_IP    Worker 服务器 IP（默认 127.0.0.1）
//   - GRPC_ADDR    Worker gRPC 监听地址（默认 0.0.0.0:9091）
package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/zhiyunliu/distributed-workflow/runtime-execution/internal/worker"
	"github.com/zhiyunliu/distributed-workflow/runtime-execution/pkg/node"
)

func main() {
	engineAddr := envOrFatal("ENGINE_ADDR")
	workerID := envOr("WORKER_ID", "worker-1")
	workerIP := envOr("WORKER_IP", "127.0.0.1")
	grpcAddr := envOr("GRPC_ADDR", "0.0.0.0:9091")

	// 注册内置节点执行器
	container := worker.NewExecutorContainer()
	_ = container.Register(&node.LogExecutor{})
	_ = container.Register(&node.SleepExecutor{})
	_ = container.Register(node.NewHTTPExecutor())

	cfg := worker.Config{
		WorkerID:   workerID,
		GRPCAddr:   grpcAddr,
		WorkerIP:   workerIP,
		EngineAddr: engineAddr,
		Capabilities: map[string]int{
			"log":   10,
			"sleep": 5,
			"http":  5,
		},
	}

	w, err := worker.New(cfg, container)
	if err != nil {
		log.Fatalf("create worker: %v", err)
	}
	if err := w.Start(); err != nil {
		log.Fatalf("start worker: %v", err)
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	log.Printf("worker %s started — gRPC: %s → engine: %s", workerID, grpcAddr, engineAddr)
	<-quit
	log.Println("shutting down...")
	w.Stop()
}

func envOrFatal(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("environment variable %s is required", key)
	}
	return v
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
