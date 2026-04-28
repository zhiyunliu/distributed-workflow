package http

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"github.com/zhiyunliu/distributed-workflow/interfaces"
	"github.com/zhiyunliu/distributed-workflow/sysmanager"
	"github.com/zhiyunliu/distributed-workflow/types"
)

// Server HTTP API 服务器
type Server struct {
	svc         interfaces.WorkflowService
	approvalSvc interfaces.ApprovalService
	auditSvc    interfaces.AuditLogService
	endpointSvc interfaces.EndpointManagerService
	templateSvc interfaces.TemplateService
	// D4: 系统管理
	sysMgr    sysmanager.Manager
	jwtSecret string
	engine    *gin.Engine
	server    *http.Server
}

// NewServer 创建 HTTP 服务器
func NewServer(svc interfaces.WorkflowService) *Server {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())

	s := &Server{svc: svc, engine: r}
	s.registerRoutes(r)
	return s
}

func (s *Server) registerRoutes(r *gin.Engine) {
	v1 := r.Group("/api/v1")
	{
		// 工作流定义管理
		v1.POST("/workflows", s.createWorkflow)
		v1.GET("/workflows/:workflowId", s.getWorkflow)
		v1.PUT("/workflows/:workflowId", s.updateWorkflow)
		v1.DELETE("/workflows/:workflowId", s.deleteWorkflow)
		v1.POST("/workflows/:workflowId/start", s.startWorkflow)

		// 版本管理（D2）
		v1.GET("/workflows/:workflowId/versions", s.listWorkflowVersions)
		v1.GET("/workflows/:workflowId/versions/:version", s.getWorkflowVersion)
		v1.POST("/workflows/:workflowId/versions/:version/rollback", s.rollbackWorkflow)
		v1.PUT("/workflows/:workflowId/versions/:version/gray", s.updateGrayConfig)

		// 实例管理
		v1.GET("/instances/:instanceId", s.getInstance)
		v1.GET("/instances", s.listInstances)

		// 生命周期管理（D2）
		v1.POST("/instances/:instanceId/pause", s.pauseInstance)
		v1.POST("/instances/:instanceId/resume", s.resumeInstance)
		v1.POST("/instances/:instanceId/cancel", s.cancelInstance)
		v1.POST("/instances/:instanceId/retry", s.retryInstance)
		v1.POST("/instances/:instanceId/nodes/:nodeId/retry", s.retryNode)

		// 死信队列（D2）
		v1.GET("/instances/:instanceId/deadletter", s.listDeadLetterTasks)
	}
}

// Start 启动 HTTP 服务
func (s *Server) Start(addr string) error {
	s.server = &http.Server{Addr: addr, Handler: s.engine}
	log.Info().Str("addr", addr).Msg("HTTP server starting")
	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error().Err(err).Msg("HTTP server error")
		}
	}()
	return nil
}

// Stop 优雅停止 HTTP 服务
func (s *Server) Stop() error {
	if s.server == nil {
		return nil
	}
	return s.server.Shutdown(context.Background())
}

// ─── 处理函数 ────────────────────────────────────────────────────────────────

func (s *Server) createWorkflow(c *gin.Context) {
	var req types.WorkflowDef
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	id, err := s.svc.CreateWorkflow(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"workflow_id": id})
}

func (s *Server) getWorkflow(c *gin.Context) {
	def, err := s.svc.GetWorkflow(c.Param("workflowId"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, def)
}

func (s *Server) updateWorkflow(c *gin.Context) {
	var req types.WorkflowDef
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.ID = c.Param("workflowId")
	newVersion, err := s.svc.UpdateWorkflow(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "version": newVersion})
}

func (s *Server) deleteWorkflow(c *gin.Context) {
	if err := s.svc.DeleteWorkflow(c.Param("workflowId")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

type startWorkflowReq struct {
	InputData map[string]interface{} `json:"inputData"`
	CreatedBy string                 `json:"createdBy"`
}

func (s *Server) startWorkflow(c *gin.Context) {
	var req startWorkflowReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	instanceID, err := s.svc.StartWorkflow(c.Param("workflowId"), req.InputData, req.CreatedBy)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"instance_id": instanceID})
}

func (s *Server) getInstance(c *gin.Context) {
	instance, err := s.svc.GetWorkflowInstance(c.Param("instanceId"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, instance)
}

// ─── D2 新增处理函数 ─────────────────────────────────────────────────────────

func (s *Server) listWorkflowVersions(c *gin.Context) {
	versions, err := s.svc.ListWorkflowVersions(c.Param("workflowId"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, versions)
}

func (s *Server) getWorkflowVersion(c *gin.Context) {
	var versionNum int
	if _, err := fmt.Sscanf(c.Param("version"), "%d", &versionNum); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid version number"})
		return
	}
	ver, err := s.svc.GetWorkflowVersion(c.Param("workflowId"), versionNum)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, ver)
}

type rollbackReq struct {
	Operator string `json:"operator"`
}

func (s *Server) rollbackWorkflow(c *gin.Context) {
	var versionNum int
	if _, err := fmt.Sscanf(c.Param("version"), "%d", &versionNum); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid version number"})
		return
	}
	var req rollbackReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	newVer, err := s.svc.RollbackWorkflow(c.Param("workflowId"), versionNum, req.Operator)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"new_version": newVer})
}

func (s *Server) updateGrayConfig(c *gin.Context) {
	var versionNum int
	if _, err := fmt.Sscanf(c.Param("version"), "%d", &versionNum); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid version number"})
		return
	}
	var cfg types.GrayReleaseConfig
	if err := c.ShouldBindJSON(&cfg); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := s.svc.UpdateGrayConfig(c.Param("workflowId"), versionNum, &cfg); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

type listInstancesReq struct {
	WorkflowID string `form:"workflowId"`
	Status     string `form:"status"`
	PageSize   int    `form:"pageSize,default=20"`
	PageNum    int    `form:"pageNum,default=1"`
}

func (s *Server) listInstances(c *gin.Context) {
	var req listInstancesReq
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	instances, err := s.svc.ListWorkflowInstances(req.WorkflowID, types.WorkflowStatus(req.Status), req.PageSize, req.PageNum)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, instances)
}

type operatorReq struct {
	Operator string `json:"operator"`
}

func (s *Server) pauseInstance(c *gin.Context) {
	var req operatorReq
	_ = c.ShouldBindJSON(&req)
	if err := s.svc.PauseInstance(c.Param("instanceId"), req.Operator); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (s *Server) resumeInstance(c *gin.Context) {
	var req operatorReq
	_ = c.ShouldBindJSON(&req)
	if err := s.svc.ResumeInstance(c.Param("instanceId"), req.Operator); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (s *Server) cancelInstance(c *gin.Context) {
	var req operatorReq
	_ = c.ShouldBindJSON(&req)
	if err := s.svc.CancelInstance(c.Param("instanceId"), req.Operator); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (s *Server) retryInstance(c *gin.Context) {
	var req operatorReq
	_ = c.ShouldBindJSON(&req)
	if err := s.svc.RetryInstance(c.Param("instanceId"), req.Operator); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (s *Server) retryNode(c *gin.Context) {
	var req operatorReq
	_ = c.ShouldBindJSON(&req)
	if err := s.svc.RetryNode(c.Param("instanceId"), c.Param("nodeId"), req.Operator); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (s *Server) listDeadLetterTasks(c *gin.Context) {
	tasks, err := s.svc.ListDeadLetterTasks(c.Param("instanceId"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, tasks)
}
