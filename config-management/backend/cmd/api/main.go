// Package main 配置管理后端服务入口
//
// 环境变量：
//   - DB_DSN     SQL Server 连接字符串（必填）
//   - JWT_SECRET JWT 签名密钥（必填）
//   - HTTP_ADDR  HTTP 监听地址（默认 :8080）
package main

import (
	"database/sql"
	"log"
	"os"
	"os/signal"
	"syscall"

	_ "github.com/microsoft/go-mssqldb"

	"github.com/zhiyunliu/distributed-workflow/config-management/backend/internal/dao"
	"github.com/zhiyunliu/distributed-workflow/config-management/backend/internal/handler"
	"github.com/zhiyunliu/distributed-workflow/config-management/backend/internal/service"
	"github.com/zhiyunliu/distributed-workflow/config-management/backend/internal/sysmodel"
	workflowengine "github.com/zhiyunliu/distributed-workflow/runtime-execution/pkg/engine"
	wfstore "github.com/zhiyunliu/distributed-workflow/runtime-execution/pkg/storage/sqlserver"
)

func main() {
	dsn := envOr("DB_DSN", "")
	jwtSecret := envOr("JWT_SECRET", "")
	httpAddr := envOr("HTTP_ADDR", ":7080")

	// 初始化系统管理数据库连接
	db, err := sql.Open("sqlserver", dsn)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()
	if err := db.Ping(); err != nil {
		log.Fatalf("ping db: %v", err)
	}

	// 系统管理 DAO
	sysDB := dao.NewDB(db)

	// 初始化系统数据
	initService := service.NewInitService(sysDB)
	if err := initService.InitializeSystem(nil); err != nil {
		log.Printf("初始化系统数据失败: %v", err)
	} else {
		log.Println("系统数据初始化完成")
	}

	// 系统管理 Service
	mgr := service.NewManager(
		sysDB.UserRepo(),
		sysDB.RoleRepo(),
		sysDB.MenuRepo(),
		sysDB.DictRepo(),
		sysmodel.AuthConfig{Secret: jwtSecret, ExpireHours: 24},
	)

	// 工作流仓储（sqlserver 实现）
	wfRepo, err := wfstore.NewRepository(wfstore.Config{DSN: dsn})
	if err != nil {
		log.Fatalf("create workflow repository: %v", err)
	}
	defer wfRepo.Close()

	// HTTP 服务：注入依赖并注册路由
	formRepo := wfstore.NewFormRepository(wfRepo.DB())
	workflowEngine := workflowengine.New(
		workflowengine.Config{GRPCAddr: "127.0.0.1:0"},
		wfRepo,
		nil,
		nil,
		formRepo,
		nil,
		nil,
		nil,
		nil,
	)

	srv := handler.NewServer()
	srv.SetSysManager(mgr, jwtSecret)
	srv.SetD4WorkflowRoutes()
	srv.SetD6FormService(workflowEngine.FormService())
	srv.RegisterFrontendRoutes() // 注册前端路由
	if workflowEngine.FormService() == nil {
		log.Println("D6 form service injection missing")
	} else {
		log.Println("D6 form service injected")
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		if err := srv.Start(httpAddr); err != nil {
			log.Printf("http server stopped: %v", err)
		}
	}()

	log.Printf("config-management api started on %s", httpAddr)
	<-quit
	log.Println("shutting down...")
	_ = srv.Stop()
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
