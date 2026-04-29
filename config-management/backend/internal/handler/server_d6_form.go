package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	types "github.com/zhiyunliu/distributed-workflow/runtime-execution/internal/common/types"
	api "github.com/zhiyunliu/distributed-workflow/runtime-execution/pkg/api"
)

// SetD6FormService 注入表单服务，并注册 D6 表单路由
func (s *Server) SetD6FormService(formSvc api.FormService) {
	s.formSvc = formSvc
	s.registerD6FormRoutes(s.engine)
}

func (s *Server) registerD6FormRoutes(r *gin.Engine) {
	forms := r.Group("/api/forms")
	{
		forms.POST("", s.createForm)
		forms.GET("", s.listForms)
		forms.GET("/:formId", s.getForm)
		forms.PUT("/:formId", s.updateForm)
		forms.POST("/:formId/publish", s.publishForm)
		forms.POST("/:formId/rollback", s.rollbackForm)
		forms.GET("/:formId/versions", s.getFormVersions)
	}

	instances := r.Group("/api/form-instances")
	{
		instances.POST("", s.saveFormInstance)
		// 静态前缀 /workflow/ 须先注册，优先级高于参数路由 /:instanceId
		instances.GET("/workflow/:workflowInstanceId", s.getFormInstanceByWorkflow)
		instances.GET("/:instanceId", s.getFormInstance)
	}
}

// ─── 表单定义处理函数 ─────────────────────────────────────────────────────────

func (s *Server) createForm(c *gin.Context) {
	if s.formSvc == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"code": 503, "message": "表单服务不可用"})
		return
	}
	var req types.FormDefinition
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	if err := s.formSvc.CreateForm(c.Request.Context(), &req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"code": 200, "message": "创建成功", "data": req})
}

type listFormsQuery struct {
	Page     int    `form:"page"`
	PageSize int    `form:"pageSize"`
	Keyword  string `form:"keyword"`
}

func (s *Server) listForms(c *gin.Context) {
	if s.formSvc == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"code": 503, "message": "表单服务不可用"})
		return
	}
	var q listFormsQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	if q.Page <= 0 {
		q.Page = 1
	}
	if q.PageSize <= 0 {
		q.PageSize = 20
	}
	params := types.FormListParams{
		Page:     q.Page,
		PageSize: q.PageSize,
		Keyword:  q.Keyword,
	}
	list, total, err := s.formSvc.ListForms(c.Request.Context(), params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "data": list, "total": total})
}

func (s *Server) getForm(c *gin.Context) {
	if s.formSvc == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"code": 503, "message": "表单服务不可用"})
		return
	}
	formID := c.Param("formId")
	form, err := s.formSvc.GetForm(c.Request.Context(), formID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "data": form})
}

func (s *Server) updateForm(c *gin.Context) {
	if s.formSvc == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"code": 503, "message": "表单服务不可用"})
		return
	}
	var req types.FormDefinition
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	req.FormID = c.Param("formId")
	if err := s.formSvc.UpdateForm(c.Request.Context(), &req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "更新成功"})
}

type publishFormReq struct {
	ChangeLog  string `json:"changeLog"`
	OperatorID string `json:"operatorId" binding:"required"`
}

func (s *Server) publishForm(c *gin.Context) {
	if s.formSvc == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"code": 503, "message": "表单服务不可用"})
		return
	}
	var req publishFormReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	formID := c.Param("formId")
	if err := s.formSvc.PublishForm(c.Request.Context(), formID, req.ChangeLog, req.OperatorID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "发布成功"})
}

type rollbackFormReq struct {
	TargetVersion int    `json:"targetVersion" binding:"required,min=1"`
	OperatorID    string `json:"operatorId"    binding:"required"`
}

func (s *Server) rollbackForm(c *gin.Context) {
	if s.formSvc == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"code": 503, "message": "表单服务不可用"})
		return
	}
	var req rollbackFormReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	formID := c.Param("formId")
	if err := s.formSvc.RollbackForm(c.Request.Context(), formID, req.TargetVersion, req.OperatorID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "版本回滚成功"})
}

func (s *Server) getFormVersions(c *gin.Context) {
	if s.formSvc == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"code": 503, "message": "表单服务不可用"})
		return
	}
	formID := c.Param("formId")
	versions, err := s.formSvc.GetFormVersions(c.Request.Context(), formID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "data": versions})
}

// ─── 表单实例处理函数 ─────────────────────────────────────────────────────────

func (s *Server) saveFormInstance(c *gin.Context) {
	if s.formSvc == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"code": 503, "message": "表单服务不可用"})
		return
	}
	var req types.FormInstance
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	if err := s.formSvc.SaveFormInstance(c.Request.Context(), &req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "保存成功", "data": req})
}

func (s *Server) getFormInstance(c *gin.Context) {
	if s.formSvc == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"code": 503, "message": "表单服务不可用"})
		return
	}
	instanceID := c.Param("instanceId")
	inst, err := s.formSvc.GetFormInstance(c.Request.Context(), instanceID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "data": inst})
}

func (s *Server) getFormInstanceByWorkflow(c *gin.Context) {
	if s.formSvc == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"code": 503, "message": "表单服务不可用"})
		return
	}
	workflowInstanceID := c.Param("workflowInstanceId")
	nodeID := c.Query("nodeId")
	inst, err := s.formSvc.GetFormInstanceByWorkflow(c.Request.Context(), workflowInstanceID, nodeID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "data": inst})
}
