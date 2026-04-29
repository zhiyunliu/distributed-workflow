package httpendpoint

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/zhiyunliu/distributed-workflow/runtime-execution/pkg/types"
	api "github.com/zhiyunliu/distributed-workflow/runtime-execution/pkg/api"
)

// SetD3Services 注入 D3 所需服务，并注册 D3 路由
func (s *Server) SetD3Services(
	approvalSvc api.ApprovalService,
	auditSvc api.AuditLogService,
	endpointSvc api.EndpointManagerService,
) {
	s.approvalSvc = approvalSvc
	s.auditSvc = auditSvc
	s.endpointSvc = endpointSvc
	s.registerD3Routes(s.engine)
}

func (s *Server) registerD3Routes(r *gin.Engine) {
	v1 := r.Group("/api/v1")
	{
		// ── 审批 ────────────────────────────────────────────────────────────
		v1.GET("/approval/tasks", s.listApprovalTasks)
		v1.POST("/approval/approve", s.approveTask)
		v1.POST("/approval/reject", s.rejectTask)
		v1.GET("/approval/:instanceId/:nodeId/records", s.getApprovalRecords)

		// ── 端点管理 ─────────────────────────────────────────────────────────
		v1.GET("/endpoints", s.listEndpoints)
		v1.POST("/endpoints", s.createEndpoint)
		v1.GET("/endpoints/:id", s.getEndpoint)
		v1.PUT("/endpoints/:id", s.updateEndpoint)
		v1.DELETE("/endpoints/:id", s.deleteEndpoint)
		v1.POST("/endpoints/:id/start", s.startEndpoint)
		v1.POST("/endpoints/:id/stop", s.stopEndpoint)
		v1.POST("/endpoints/:id/trigger", s.triggerEndpoint)

		// ── 审计日志 ─────────────────────────────────────────────────────────
		v1.GET("/audit/logs", s.queryAuditLogs)
		v1.POST("/audit/archive", s.archiveAuditLogs)
	}
}

// ─── 审批处理函数 ────────────────────────────────────────────────────────────

type listApprovalTasksReq struct {
	Approver   string `form:"approver"`
	InstanceID string `form:"instanceId"`
	NodeID     string `form:"nodeId"`
	WorkflowID string `form:"workflowId"`
}

func (s *Server) listApprovalTasks(c *gin.Context) {
	if s.approvalSvc == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "approval service not available"})
		return
	}
	var req listApprovalTasksReq
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	filter := types.ApprovalTaskFilter{
		Approver:   req.Approver,
		InstanceID: req.InstanceID,
		NodeID:     req.NodeID,
		WorkflowID: req.WorkflowID,
	}
	tasks, err := s.approvalSvc.GetPendingTasks(filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": tasks, "total": len(tasks)})
}

type approvalActionReq struct {
	InstanceID string                 `json:"instanceId" binding:"required"`
	NodeID     string                 `json:"nodeId"     binding:"required"`
	Approver   string                 `json:"approver"   binding:"required"`
	Comment    string                 `json:"comment"`
	FormData   map[string]interface{} `json:"formData"`
	OperateIP  string                 `json:"operateIp"`
}

func (s *Server) approveTask(c *gin.Context) {
	if s.approvalSvc == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "approval service not available"})
		return
	}
	var req approvalActionReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ip := req.OperateIP
	if ip == "" {
		ip = c.ClientIP()
	}
	if err := s.approvalSvc.Approve(req.InstanceID, req.NodeID, req.Approver, req.Comment, req.FormData, ip); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (s *Server) rejectTask(c *gin.Context) {
	if s.approvalSvc == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "approval service not available"})
		return
	}
	var req approvalActionReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ip := req.OperateIP
	if ip == "" {
		ip = c.ClientIP()
	}
	if err := s.approvalSvc.Reject(req.InstanceID, req.NodeID, req.Approver, req.Comment, req.FormData, ip); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (s *Server) getApprovalRecords(c *gin.Context) {
	if s.approvalSvc == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "approval service not available"})
		return
	}
	records, err := s.approvalSvc.GetApprovalRecords(c.Param("instanceId"), c.Param("nodeId"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": records})
}

// ─── 端点管理处理函数 ────────────────────────────────────────────────────────

type listEndpointsReq struct {
	WorkflowID string `form:"workflowId"`
	Type       string `form:"type"`
	Page       int    `form:"page,default=1"`
	PageSize   int    `form:"pageSize,default=20"`
}

func (s *Server) listEndpoints(c *gin.Context) {
	if s.endpointSvc == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "endpoint service not available"})
		return
	}
	var req listEndpointsReq
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	filter := types.EndpointFilter{
		WorkflowID: req.WorkflowID,
	}
	if req.Type != "" {
		filter.Type = types.EndpointType(req.Type)
	}
	eps, total, err := s.endpointSvc.ListEndpoints(filter, req.Page, req.PageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": eps, "total": total})
}

func (s *Server) createEndpoint(c *gin.Context) {
	if s.endpointSvc == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "endpoint service not available"})
		return
	}
	var ep types.WorkflowEndpoint
	if err := c.ShouldBindJSON(&ep); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	id, err := s.endpointSvc.CreateEndpoint(&ep)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id})
}

func (s *Server) getEndpoint(c *gin.Context) {
	if s.endpointSvc == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "endpoint service not available"})
		return
	}
	ep, err := s.endpointSvc.GetEndpoint(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, ep)
}

func (s *Server) updateEndpoint(c *gin.Context) {
	if s.endpointSvc == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "endpoint service not available"})
		return
	}
	var ep types.WorkflowEndpoint
	if err := c.ShouldBindJSON(&ep); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ep.ID = c.Param("id")
	if err := s.endpointSvc.UpdateEndpoint(&ep); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (s *Server) deleteEndpoint(c *gin.Context) {
	if s.endpointSvc == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "endpoint service not available"})
		return
	}
	if err := s.endpointSvc.DeleteEndpoint(c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (s *Server) startEndpoint(c *gin.Context) {
	if s.endpointSvc == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "endpoint service not available"})
		return
	}
	if err := s.endpointSvc.StartEndpoint(c.Param("id")); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (s *Server) stopEndpoint(c *gin.Context) {
	if s.endpointSvc == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "endpoint service not available"})
		return
	}
	if err := s.endpointSvc.StopEndpoint(c.Param("id")); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

type triggerEndpointReq struct {
	InputData map[string]interface{} `json:"inputData"`
}

func (s *Server) triggerEndpoint(c *gin.Context) {
	if s.endpointSvc == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "endpoint service not available"})
		return
	}
	var req triggerEndpointReq
	_ = c.ShouldBindJSON(&req)
	instanceID, err := s.endpointSvc.TriggerEndpoint(c.Param("id"), req.InputData)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"instance_id": instanceID})
}

// ─── 审计日志处理函数 ────────────────────────────────────────────────────────

type queryAuditLogsReq struct {
	WorkflowID    string `form:"workflowId"`
	InstanceID    string `form:"instanceId"`
	NodeID        string `form:"nodeId"`
	EndpointID    string `form:"endpointId"`
	Operator      string `form:"operator"`
	OperationType string `form:"operationType"`
	Page          int    `form:"page,default=1"`
	PageSize      int    `form:"pageSize,default=20"`
}

func (s *Server) queryAuditLogs(c *gin.Context) {
	if s.auditSvc == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "audit service not available"})
		return
	}
	var req queryAuditLogsReq
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	filter := types.AuditLogFilter{
		WorkflowID: req.WorkflowID,
		InstanceID: req.InstanceID,
		NodeID:     req.NodeID,
		EndpointID: req.EndpointID,
		Operator:   req.Operator,
	}
	if req.OperationType != "" {
		filter.OperationType = []types.AuditOperationType{types.AuditOperationType(req.OperationType)}
	}
	logs, total, err := s.auditSvc.QueryLogs(filter, req.Page, req.PageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": logs, "total": total})
}

type archiveAuditLogsReq struct {
	BeforeDay int `json:"beforeDay" binding:"required,min=1"`
}

func (s *Server) archiveAuditLogs(c *gin.Context) {
	if s.auditSvc == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "audit service not available"})
		return
	}
	var req archiveAuditLogsReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	beforeTime := time.Now().AddDate(0, 0, -req.BeforeDay)
	if err := s.auditSvc.ArchiveLogs(beforeTime); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}


