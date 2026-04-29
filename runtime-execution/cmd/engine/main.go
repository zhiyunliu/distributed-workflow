// Package main Engine 运行时服务入口
//
// 环境变量：
//   - DB_DSN     SQL Server 连接字符串（必填）
//   - REDIS_ADDR Redis 地址（默认 localhost:6379）
//   - GRPC_ADDR  gRPC 监听地址（默认 0.0.0.0:9090）
//   - HTTP_ADDR  HTTP 触发接口监听地址（默认 0.0.0.0:8080）
package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/zhiyunliu/distributed-workflow/runtime-execution/internal/engine"
	"github.com/zhiyunliu/distributed-workflow/runtime-execution/internal/storage/redis"
	httpendpoint "github.com/zhiyunliu/distributed-workflow/runtime-execution/pkg/endpoint/http"
	"github.com/zhiyunliu/distributed-workflow/runtime-execution/pkg/storage/sqlserver"
)

func main() {
	dsn := envOrFatal("DB_DSN")
	redisAddr := envOr("REDIS_ADDR", "localhost:6379")
	grpcAddr := envOr("GRPC_ADDR", "0.0.0.0:9090")
	httpAddr := envOr("HTTP_ADDR", "0.0.0.0:8080")

	// 工作流仓储（sqlserver 实现）
	repo, err := sqlserver.NewRepository(sqlserver.Config{DSN: dsn})
	if err != nil {
		log.Fatalf("create workflow repository: %v", err)
	}
	defer repo.Close()

	// Redis 客户端（分布式锁 + 任务队列）
	redisClient, err := redis.NewClient(redis.Config{Addr: redisAddr})
	if err != nil {
		log.Fatalf("connect redis: %v", err)
	}
	defer redisClient.Close()

	// 创建引擎（D6 可选仓储传 nil，待后续注入）
	eng := engine.New(
		engine.Config{GRPCAddr: grpcAddr},
		repo,                    // api.WorkflowRepository
		redisClient,             // api.RedisRepository
		redisClient.RawClient(), // *redis.Client（快照 + failover）
		nil,                     // formRepo
		nil,                     // advApprovalRepo
		nil,                     // analyticsRepo
		nil,                     // pluginRepo
		nil,                     // oauthRepo
	)
	if err := eng.Start(); err != nil {
		log.Fatalf("start engine: %v", err)
	}

	// HTTP 触发路由
	httpServer := httpendpoint.NewServer(eng.WorkflowService())

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		if err := httpServer.Start(httpAddr); err != nil {
			log.Printf("http server stopped: %v", err)
		}
	}()

	log.Printf("engine started — gRPC: %s  HTTP: %s", grpcAddr, httpAddr)
	<-quit
	log.Println("shutting down...")
	eng.Stop()
	_ = httpServer.Stop()
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
