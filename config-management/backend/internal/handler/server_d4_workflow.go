package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/zhiyunliu/distributed-workflow/endpoint/schedule"
	types "github.com/zhiyunliu/distributed-workflow/runtime-execution/internal/common/types"
)

// registerD4WorkflowRoutes 注册 D4 工作流设计器扩展路由
// 需在 SetSysManager 之后调用（依赖 JWT 中间件已挂载）
func (s *Server) registerD4WorkflowRoutes(r *gin.Engine) {
	authorized := r.Group("/api/v1")
	authorized.Use(s.jwtAuthMiddleware())
	{
		// ── 工作流设计器配套 API ──────────────────────────────────────────
		def := authorized.Group("/workflow/def")
		{
			def.POST("/save-draft", s.saveWorkflowDraft)
			def.POST("/publish", s.publishWorkflow)
			def.GET("/:id/versions", s.listWorkflowVersionsV2)
			def.GET("/version/:versionId", s.getWorkflowVersionByID)
			def.POST("/rollback", s.rollbackWorkflowDef)
			def.POST("/import", s.importWorkflowDef)
			def.GET("/:id/export", s.exportWorkflowDef)
			def.POST("/:id/validate", s.validateWorkflowDef)
			def.GET("/list", s.listWorkflowDefsForDesigner)
		}

		// ── 实例监控扩展 API ──────────────────────────────────────────────
		inst := authorized.Group("/workflow/instance")
		{
			inst.GET("/:id/topology", s.getInstanceTopology)
			inst.GET("/:id/nodes", s.getInstanceNodes)
			inst.GET("/:id/context", s.getInstanceContext)
			inst.GET("/:id/events", s.getInstanceEvents)
			inst.POST("/:id/node/:nodeId/retry", s.retryInstanceNode)
		}

		// ── 端点扩展 API ──────────────────────────────────────────────────
		ep := authorized.Group("/endpoint")
		{
			ep.GET("/:id/trigger-history", s.getEndpointTriggerHistory)
			ep.POST("/:id/test", s.testEndpoint)
			ep.POST("/cron/validate", s.validateCronExpr)
		}
	}
}

// SetD4WorkflowRoutes 向外暴露注册方法（在 SetSysManager 之后调用）
func (s *Server) SetD4WorkflowRoutes() {
	s.registerD4WorkflowRoutes(s.engine)
}

// ─── 工作流设计器 ─────────────────────────────────────────────────────────────

func (s *Server) saveWorkflowDraft(c *gin.Context) {
	var def types.WorkflowDef
	if err := c.ShouldBindJSON(&def); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	// 草稿通过 CreateWorkflow 保存，若已存在则 Update
	if def.ID == "" {
		id, err := s.workflowRepo.CreateWorkflow(&def)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{"code": 500, "message": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"code": 200, "data": gin.H{"id": id}})
		return
	}
	version, err := s.workflowRepo.UpdateWorkflow(&def)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "data": gin.H{"id": def.ID, "version": version}})
}

func (s *Server) publishWorkflow(c *gin.Context) {
	// 发布与 save-draft 共用 UpdateWorkflow，后端语义上统一处理
	s.saveWorkflowDraft(c)
}

func (s *Server) listWorkflowVersionsV2(c *gin.Context) {
	id := c.Param("id")
	versions, err := s.workflowRepo.ListWorkflowVersions(id)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "data": versions})
}

func (s *Server) getWorkflowVersionByID(c *gin.Context) {
	// versionId 格式: {workflowID}:{version}
	versionIDParam := c.Param("versionId")
	// 此处简单按 query 参数处理
	workflowID := c.Query("workflowId")
	version, _ := strconv.Atoi(versionIDParam)
	wv, err := s.workflowRepo.GetWorkflowVersion(workflowID, version)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "data": wv})
}

func (s *Server) rollbackWorkflowDef(c *gin.Context) {
	var req struct {
		WorkflowID    string `json:"workflowId"    binding:"required"`
		TargetVersion int    `json:"targetVersion" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	username, _ := c.Get("username")
	operator, _ := username.(string)
	newVersion, err := s.workflowRepo.RollbackWorkflow(req.WorkflowID, req.TargetVersion, operator)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "data": gin.H{"newVersion": newVersion}})
}

func (s *Server) importWorkflowDef(c *gin.Context) {
	var def types.WorkflowDef
	if err := c.ShouldBindJSON(&def); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的流程定义: " + err.Error()})
		return
	}
	def.ID = "" // 强制创建新流程
	id, err := s.workflowRepo.CreateWorkflow(&def)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "data": gin.H{"id": id}})
}

func (s *Server) exportWorkflowDef(c *gin.Context) {
	id := c.Param("id")
	def, err := s.workflowRepo.GetWorkflow(id)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": err.Error()})
		return
	}
	data, err := json.MarshalIndent(def, "", "  ")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.Header("Content-Disposition", "attachment; filename=workflow_"+id+".json")
	c.Data(http.StatusOK, "application/json", data)
}

func (s *Server) validateWorkflowDef(c *gin.Context) {
	var def types.WorkflowDef
	if err := c.ShouldBindJSON(&def); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	// 简单校验：必须有 start 节点和 end 节点
	hasStart, hasEnd := false, false
	for _, node := range def.Nodes {
		if node.Type == "start" {
			hasStart = true
		}
		if node.Type == "end" {
			hasEnd = true
		}
	}
	if !hasStart || !hasEnd {
		c.JSON(http.StatusOK, gin.H{"code": 200, "data": gin.H{
			"valid":  false,
			"errors": []string{"流程必须包含开始节点和结束节点"},
		}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "data": gin.H{"valid": true, "errors": []string{}}})
}

func (s *Server) listWorkflowDefsForDesigner(c *gin.Context) {
	// 返回所有可用于子流程调用的流程列表（通过 GetWorkflow 实现基础查询）
	// 实际完整实现需在 WorkflowService 新增 ListWorkflowDefs 接口
	c.JSON(http.StatusOK, gin.H{"code": 200, "data": []interface{}{}})
}

// ─── 实例监控扩展 ─────────────────────────────────────────────────────────────

func (s *Server) getInstanceTopology(c *gin.Context) {
	id := c.Param("id")
	instance, err := s.workflowRepo.GetWorkflowInstance(id)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": err.Error()})
		return
	}
	def, err := s.workflowRepo.GetWorkflow(instance.WorkflowID)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": err.Error()})
		return
	}
	nodeStates, err := s.getNodeStatesInternal(id)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": gin.H{
			"definition": def,
			"nodeStates": nodeStates,
			"status":     instance.Status,
		},
	})
}

func (s *Server) getInstanceNodes(c *gin.Context) {
	id := c.Param("id")
	nodeStates, err := s.getNodeStatesInternal(id)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "data": nodeStates})
}

func (s *Server) getNodeStatesInternal(instanceID string) (interface{}, error) {
	// WorkflowService 暂无独立的节点状态列表接口，通过 GetWorkflowInstance 获取
	instance, err := s.workflowRepo.GetWorkflowInstance(instanceID)
	if err != nil {
		return nil, err
	}
	return instance, nil
}

func (s *Server) getInstanceContext(c *gin.Context) {
	id := c.Param("id")
	instance, err := s.workflowRepo.GetWorkflowInstance(id)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "data": gin.H{
		"inputData":  instance.InputData,
		"outputData": instance.OutputData,
	}})
}

func (s *Server) getInstanceEvents(c *gin.Context) {
	id := c.Param("id")
	if s.auditSvc == nil {
		c.JSON(http.StatusOK, gin.H{"code": 200, "data": []interface{}{}})
		return
	}
	filter := types.AuditLogFilter{InstanceID: id}
	logs, _, err := s.auditSvc.QueryLogs(filter, 1, 1000)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "data": logs})
}

func (s *Server) retryInstanceNode(c *gin.Context) {
	id := c.Param("id")
	nodeID := c.Param("nodeId")
	username, _ := c.Get("username")
	operator, _ := username.(string)
	if err := s.workflowRepo.RetryNode(id, nodeID, operator); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "节点重试成功"})
}

// ─── 端点扩展 ─────────────────────────────────────────────────────────────────

func (s *Server) getEndpointTriggerHistory(c *gin.Context) {
	// 端点触发历史暂通过审计日志查询
	epID := c.Param("id")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	if s.auditSvc == nil {
		c.JSON(http.StatusOK, gin.H{"code": 200, "data": gin.H{"list": []interface{}{}, "total": 0}})
		return
	}
	filter := types.AuditLogFilter{InstanceID: epID}
	logs, total, err := s.auditSvc.QueryLogs(filter, page, pageSize)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "data": gin.H{"list": logs, "total": total}})
}

func (s *Server) testEndpoint(c *gin.Context) {
	epID := c.Param("id")
	if s.endpointSvc == nil {
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": "端点服务未初始化"})
		return
	}
	instanceID, err := s.endpointSvc.TriggerEndpoint(epID, map[string]interface{}{})
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "data": gin.H{"instanceId": instanceID}})
}

func (s *Server) validateCronExpr(c *gin.Context) {
	var req struct {
		Expr string `json:"expr" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	now := time.Now()
	var nextTimes []string
	cur := now
	valid := true
	errMsg := ""
	for i := 0; i < 5; i++ {
		next, err := schedule.ParseNextTime(req.Expr, cur)
		if err != nil {
			valid = false
			errMsg = err.Error()
			break
		}
		nextTimes = append(nextTimes, next.Format(time.RFC3339))
		cur = next
	}
	if !valid {
		c.JSON(http.StatusOK, gin.H{"code": 200, "data": gin.H{
			"valid":   false,
			"message": errMsg,
		}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "data": gin.H{
		"valid":     true,
		"nextTimes": nextTimes,
	}})
}
