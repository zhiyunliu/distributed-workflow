package handler

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/zhiyunliu/distributed-workflow/config-management/backend/internal/sysmanager"
	api "github.com/zhiyunliu/distributed-workflow/runtime-execution/pkg/api"
)

// Server 管理侧 HTTP API 服务器
type Server struct {
	// D4: 系统管理
	sysMgr    sysmanager.Manager
	jwtSecret string
	// D6: 表单引擎 & 高级审批
	formSvc        api.FormService
	advApprovalSvc api.AdvancedApprovalService
	// D6: BI统计、插件管理、OAuth登录
	analyticsSvc api.AnalyticsService
	pluginSvc    api.PluginService
	oauthSvc     api.OAuthService
	// D5: 模板服务
	templateSvc api.TemplateService
	// D4: 工作流管理路由
	workflowRepo api.WorkflowService
	auditSvc     api.AuditLogService
	endpointSvc  api.EndpointManagerService

	engine *gin.Engine
	server *http.Server
}

// NewServer 创建管理侧 HTTP 服务器
func NewServer() *Server {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	return &Server{engine: r}
}

// Start 启动 HTTP 服务
func (s *Server) Start(addr string) error {
	s.server = &http.Server{Addr: addr, Handler: s.engine}
	return s.server.ListenAndServe()
}

// Stop 优雅停止 HTTP 服务
func (s *Server) Stop() error {
	if s.server == nil {
		return nil
	}
	return s.server.Shutdown(context.Background())
}