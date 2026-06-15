package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	api "github.com/zhiyunliu/distributed-workflow/runtime-execution/pkg/api"
	"github.com/zhiyunliu/distributed-workflow/runtime-execution/pkg/jwtutil"
	types "github.com/zhiyunliu/distributed-workflow/runtime-execution/pkg/types"
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
		s.respondError(c, http.StatusServiceUnavailable, 503, "表单服务不可用")
		return
	}
	var req types.FormDefinition
	if err := c.ShouldBindJSON(&req); err != nil {
		s.respondError(c, http.StatusBadRequest, 400, err.Error())
		return
	}
	if err := req.NormalizeSchema(req.Schema, req.FormSchema); err != nil {
		s.respondError(c, http.StatusBadRequest, 400, err.Error())
		return
	}
	if err := s.formSvc.CreateForm(c.Request.Context(), &req); err != nil {
		s.respondError(c, http.StatusInternalServerError, 500, err.Error())
		return
	}
	s.respondJSON(c, http.StatusCreated, 200, "创建成功", req, nil)
}

type listFormsQuery struct {
	Page     int    `form:"page"`
	PageSize int    `form:"pageSize"`
	Keyword  string `form:"keyword"`
}

func (s *Server) listForms(c *gin.Context) {
	if s.formSvc == nil {
		s.respondError(c, http.StatusServiceUnavailable, 503, "表单服务不可用")
		return
	}
	var q listFormsQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		s.respondError(c, http.StatusBadRequest, 400, err.Error())
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
		s.respondError(c, http.StatusInternalServerError, 500, err.Error())
		return
	}
	s.respondJSON(c, http.StatusOK, 200, "", gin.H{"list": list, "total": total}, nil)
}

func (s *Server) getForm(c *gin.Context) {
	if s.formSvc == nil {
		s.respondError(c, http.StatusServiceUnavailable, 503, "表单服务不可用")
		return
	}
	formID := c.Param("formId")
	form, err := s.formSvc.GetForm(c.Request.Context(), formID)
	if err != nil {
		s.respondError(c, http.StatusNotFound, 404, err.Error())
		return
	}
	s.respondJSON(c, http.StatusOK, 200, "", form, nil)
}

func (s *Server) updateForm(c *gin.Context) {
	if s.formSvc == nil {
		s.respondError(c, http.StatusServiceUnavailable, 503, "表单服务不可用")
		return
	}
	var req types.FormDefinition
	if err := c.ShouldBindJSON(&req); err != nil {
		s.respondError(c, http.StatusBadRequest, 400, err.Error())
		return
	}
	if err := req.NormalizeSchema(req.Schema, req.FormSchema); err != nil {
		s.respondError(c, http.StatusBadRequest, 400, err.Error())
		return
	}
	req.FormID = c.Param("formId")
	if err := s.formSvc.UpdateForm(c.Request.Context(), &req); err != nil {
		s.respondError(c, http.StatusInternalServerError, 500, err.Error())
		return
	}
	s.respondJSON(c, http.StatusOK, 200, "更新成功", nil, nil)
}

type publishFormReq struct {
	ChangeLog  string `json:"changeLog"`
	OperatorID string `json:"operatorId"`
}

func resolveD6OperatorID(c *gin.Context, jwtSecret string) string {
	if username, ok := c.Get("username"); ok {
		if operator, ok := username.(string); ok && operator != "" {
			return operator
		}
	}
	if userID, ok := c.Get("userID"); ok {
		switch v := userID.(type) {
		case int64:
			if v != 0 {
				return strconv.FormatInt(v, 10)
			}
		case int:
			if v != 0 {
				return strconv.Itoa(v)
			}
		case string:
			if v != "" {
				return v
			}
		}
	}
	if token := extractBearerToken(c); token != "" {
		if claims, err := jwtutil.Parse(token, jwtSecret); err == nil {
			if claims.Username != "" {
				return claims.Username
			}
			if claims.UserID != 0 {
				return strconv.FormatInt(claims.UserID, 10)
			}
		}
	}
	return ""
}

func (s *Server) publishForm(c *gin.Context) {
	if s.formSvc == nil {
		s.respondError(c, http.StatusServiceUnavailable, 503, "表单服务不可用")
		return
	}
	var req publishFormReq
	if err := c.ShouldBindJSON(&req); err != nil {
		s.respondError(c, http.StatusBadRequest, 400, err.Error())
		return
	}
	operatorID := resolveD6OperatorID(c, s.jwtSecret)
	if operatorID == "" {
		s.respondError(c, http.StatusUnauthorized, 401, "未登录或 token 已失效")
		return
	}
	formID := c.Param("formId")
	if err := s.formSvc.PublishForm(c.Request.Context(), formID, req.ChangeLog, operatorID); err != nil {
		s.respondError(c, http.StatusInternalServerError, 500, err.Error())
		return
	}
	s.respondJSON(c, http.StatusOK, 200, "发布成功", nil, nil)
}

type rollbackFormReq struct {
	TargetVersion int    `json:"targetVersion" binding:"required,min=1"`
	OperatorID    string `json:"operatorId"`
}

func (s *Server) rollbackForm(c *gin.Context) {
	if s.formSvc == nil {
		s.respondError(c, http.StatusServiceUnavailable, 503, "表单服务不可用")
		return
	}
	var req rollbackFormReq
	if err := c.ShouldBindJSON(&req); err != nil {
		s.respondError(c, http.StatusBadRequest, 400, err.Error())
		return
	}
	operatorID := resolveD6OperatorID(c, s.jwtSecret)
	if operatorID == "" {
		s.respondError(c, http.StatusUnauthorized, 401, "未登录或 token 已失效")
		return
	}
	formID := c.Param("formId")
	if err := s.formSvc.RollbackForm(c.Request.Context(), formID, req.TargetVersion, operatorID); err != nil {
		s.respondError(c, http.StatusInternalServerError, 500, err.Error())
		return
	}
	s.respondJSON(c, http.StatusOK, 200, "版本回滚成功", nil, nil)
}

func (s *Server) getFormVersions(c *gin.Context) {
	if s.formSvc == nil {
		s.respondError(c, http.StatusServiceUnavailable, 503, "表单服务不可用")
		return
	}
	formID := c.Param("formId")
	versions, err := s.formSvc.GetFormVersions(c.Request.Context(), formID)
	if err != nil {
		s.respondError(c, http.StatusInternalServerError, 500, err.Error())
		return
	}
	s.respondJSON(c, http.StatusOK, 200, "", versions, nil)
}

// ─── 表单实例处理函数 ─────────────────────────────────────────────────────────

func (s *Server) saveFormInstance(c *gin.Context) {
	if s.formSvc == nil {
		s.respondError(c, http.StatusServiceUnavailable, 503, "表单服务不可用")
		return
	}
	var req types.FormInstance
	if err := c.ShouldBindJSON(&req); err != nil {
		s.respondError(c, http.StatusBadRequest, 400, err.Error())
		return
	}
	if err := s.formSvc.SaveFormInstance(c.Request.Context(), &req); err != nil {
		s.respondError(c, http.StatusInternalServerError, 500, err.Error())
		return
	}
	s.respondJSON(c, http.StatusOK, 200, "保存成功", req, nil)
}

func (s *Server) getFormInstance(c *gin.Context) {
	if s.formSvc == nil {
		s.respondError(c, http.StatusServiceUnavailable, 503, "表单服务不可用")
		return
	}
	instanceID := c.Param("instanceId")
	inst, err := s.formSvc.GetFormInstance(c.Request.Context(), instanceID)
	if err != nil {
		s.respondError(c, http.StatusNotFound, 404, err.Error())
		return
	}
	s.respondJSON(c, http.StatusOK, 200, "", inst, nil)
}

func (s *Server) getFormInstanceByWorkflow(c *gin.Context) {
	if s.formSvc == nil {
		s.respondError(c, http.StatusServiceUnavailable, 503, "表单服务不可用")
		return
	}
	workflowInstanceID := c.Param("workflowInstanceId")
	nodeID := c.Query("nodeId")
	inst, err := s.formSvc.GetFormInstanceByWorkflow(c.Request.Context(), workflowInstanceID, nodeID)
	if err != nil {
		s.respondError(c, http.StatusNotFound, 404, err.Error())
		return
	}
	s.respondJSON(c, http.StatusOK, 200, "", inst, nil)
}
