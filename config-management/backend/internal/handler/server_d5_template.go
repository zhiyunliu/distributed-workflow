package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	api "github.com/zhiyunliu/distributed-workflow/runtime-execution/pkg/api"
	types "github.com/zhiyunliu/distributed-workflow/runtime-execution/pkg/types"
)

// SetD5TemplateService 注入 D5 模板服务并注册路由
func (s *Server) SetD5TemplateService(templateSvc api.TemplateService) {
	s.templateSvc = templateSvc
	s.registerD5TemplateRoutes(s.engine)
}

func (s *Server) registerD5TemplateRoutes(r *gin.Engine) {
	authorized := r.Group("/api/v1")
	authorized.Use(s.jwtAuthMiddleware())
	{
		tpl := authorized.Group("/template")
		{
			tpl.GET("/market", s.listTemplateMarket)
			tpl.POST("/create", s.createTemplate)
			tpl.GET("/categories", s.listTemplateCategories)
			tpl.POST("/import", s.importTemplate)
			tpl.GET("/:templateId", s.getTemplateDetail)
			tpl.POST("/:templateId/install", s.installTemplate)
			tpl.PUT("/:templateId/update", s.updateTemplate)
			tpl.POST("/:templateId/publish", s.publishTemplate)
			tpl.GET("/:templateId/export", s.exportTemplate)
		}
	}
}

func (s *Server) listTemplateMarket(c *gin.Context) {
	if s.templateSvc == nil {
		s.respondError(c, http.StatusServiceUnavailable, 503, "模板服务不可用")
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	categoryID, _ := strconv.ParseInt(c.DefaultQuery("categoryId", "0"), 10, 64)

	filter := types.WorkflowTemplateFilter{
		Keyword:       c.Query("keyword"),
		CategoryID:    categoryID,
		OnlyPublished: true,
	}
	templates, total, err := s.templateSvc.ListTemplates(filter, page, pageSize)
	if err != nil {
		s.respondError(c, http.StatusOK, 500, err.Error())
		return
	}
	s.respondJSON(c, http.StatusOK, 200, "", gin.H{"list": templates, "total": total}, nil)
}

func (s *Server) getTemplateDetail(c *gin.Context) {
	if s.templateSvc == nil {
		s.respondError(c, http.StatusServiceUnavailable, 503, "模板服务不可用")
		return
	}
	template, err := s.templateSvc.GetTemplate(c.Param("templateId"))
	if err != nil {
		s.respondError(c, http.StatusOK, 404, err.Error())
		return
	}
	s.respondJSON(c, http.StatusOK, 200, "", template, nil)
}

func (s *Server) installTemplate(c *gin.Context) {
	if s.templateSvc == nil {
		s.respondError(c, http.StatusServiceUnavailable, 503, "模板服务不可用")
		return
	}
	username, _ := c.Get("username")
	operator, _ := username.(string)
	result, err := s.templateSvc.InstallTemplate(c.Param("templateId"), operator)
	if err != nil {
		s.respondError(c, http.StatusOK, 500, err.Error())
		return
	}
	s.respondJSON(c, http.StatusOK, 200, "安装成功", result, nil)
}

func (s *Server) createTemplate(c *gin.Context) {
	if s.templateSvc == nil {
		s.respondError(c, http.StatusServiceUnavailable, 503, "模板服务不可用")
		return
	}
	var template types.WorkflowTemplate
	if err := c.ShouldBindJSON(&template); err != nil {
		s.respondError(c, http.StatusBadRequest, 400, err.Error())
		return
	}
	username, _ := c.Get("username")
	operator, _ := username.(string)
	template.Author = operator

	id, err := s.templateSvc.CreateTemplate(&template)
	if err != nil {
		s.respondError(c, http.StatusOK, 500, err.Error())
		return
	}
	s.respondJSON(c, http.StatusOK, 200, "", gin.H{"templateId": id}, nil)
}

func (s *Server) updateTemplate(c *gin.Context) {
	if s.templateSvc == nil {
		s.respondError(c, http.StatusServiceUnavailable, 503, "模板服务不可用")
		return
	}
	var template types.WorkflowTemplate
	if err := c.ShouldBindJSON(&template); err != nil {
		s.respondError(c, http.StatusBadRequest, 400, err.Error())
		return
	}
	template.TemplateID = c.Param("templateId")
	if err := s.templateSvc.UpdateTemplate(&template); err != nil {
		s.respondError(c, http.StatusOK, 500, err.Error())
		return
	}
	s.respondError(c, http.StatusOK, 200, "更新成功")
}

func (s *Server) publishTemplate(c *gin.Context) {
	if s.templateSvc == nil {
		s.respondError(c, http.StatusServiceUnavailable, 503, "模板服务不可用")
		return
	}
	if err := s.templateSvc.PublishTemplate(c.Param("templateId")); err != nil {
		s.respondError(c, http.StatusOK, 500, err.Error())
		return
	}
	s.respondError(c, http.StatusOK, 200, "发布成功")
}

func (s *Server) listTemplateCategories(c *gin.Context) {
	if s.templateSvc == nil {
		s.respondError(c, http.StatusServiceUnavailable, 503, "模板服务不可用")
		return
	}
	categories, err := s.templateSvc.ListCategories()
	if err != nil {
		s.respondError(c, http.StatusOK, 500, err.Error())
		return
	}
	s.respondJSON(c, http.StatusOK, 200, "", categories, nil)
}

func (s *Server) importTemplate(c *gin.Context) {
	if s.templateSvc == nil {
		s.respondError(c, http.StatusServiceUnavailable, 503, "模板服务不可用")
		return
	}
	var template types.WorkflowTemplate
	if err := c.ShouldBindJSON(&template); err != nil {
		s.respondError(c, http.StatusBadRequest, 400, err.Error())
		return
	}
	username, _ := c.Get("username")
	operator, _ := username.(string)
	template.Author = operator

	id, err := s.templateSvc.ImportTemplate(&template)
	if err != nil {
		s.respondError(c, http.StatusOK, 500, err.Error())
		return
	}
	s.respondJSON(c, http.StatusOK, 200, "", gin.H{"templateId": id}, nil)
}

func (s *Server) exportTemplate(c *gin.Context) {
	if s.templateSvc == nil {
		s.respondError(c, http.StatusServiceUnavailable, 503, "模板服务不可用")
		return
	}
	template, err := s.templateSvc.ExportTemplate(c.Param("templateId"))
	if err != nil {
		s.respondError(c, http.StatusOK, 500, err.Error())
		return
	}
	data, err := json.MarshalIndent(template, "", "  ")
	if err != nil {
		s.respondError(c, http.StatusInternalServerError, 500, err.Error())
		return
	}
	c.Header("Content-Disposition", "attachment; filename=template_"+template.TemplateID+".json")
	c.Data(http.StatusOK, "application/json", data)
}
