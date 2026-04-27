package http

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"github.com/zhiyunliu/distributed-workflow/interfaces"
	"github.com/zhiyunliu/distributed-workflow/types"
)

// Server HTTP API 服务器
type Server struct {
	svc    interfaces.WorkflowService
	engine *gin.Engine
	server *http.Server
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
		v1.POST("/workflows", s.createWorkflow)
		v1.GET("/workflows/:workflowId", s.getWorkflow)
		v1.PUT("/workflows/:workflowId", s.updateWorkflow)
		v1.DELETE("/workflows/:workflowId", s.deleteWorkflow)
		v1.POST("/workflows/:workflowId/start", s.startWorkflow)
		v1.GET("/instances/:instanceId", s.getInstance)
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
	if err := s.svc.UpdateWorkflow(&req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
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
