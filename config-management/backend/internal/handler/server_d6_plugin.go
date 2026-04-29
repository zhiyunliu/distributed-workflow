package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	types "github.com/zhiyunliu/distributed-workflow/runtime-execution/pkg/types"
	api "github.com/zhiyunliu/distributed-workflow/runtime-execution/pkg/api"
)

// SetD6PluginService 注入插件管理服务，并注册插件路由
func (s *Server) SetD6PluginService(pluginSvc api.PluginService) {
	s.pluginSvc = pluginSvc
	s.registerD6PluginRoutes(s.engine)
}

func (s *Server) registerD6PluginRoutes(r *gin.Engine) {
	plugins := r.Group("/api/plugins")
	{
		plugins.GET("/market", s.listPluginMarket)
		plugins.GET("/installed", s.listInstalledPlugins)
		// 静态路径 /market 和 /installed 须先注册，优先级高于参数路由 /:pluginId
		plugins.GET("/:pluginId", s.getPlugin)
		plugins.POST("/:pluginId/install", s.installPlugin)
		plugins.POST("/:pluginId/uninstall", s.uninstallPlugin)
		plugins.POST("/:pluginId/enable", s.enablePlugin)
		plugins.POST("/:pluginId/disable", s.disablePlugin)
		plugins.GET("/:pluginId/config", s.getPluginConfig)
		plugins.PUT("/:pluginId/config", s.savePluginConfig)
	}
}

// ─── 插件管理处理函数 ─────────────────────────────────────────────────────────

type pluginMarketQuery struct {
	Page     int `form:"page"`
	PageSize int `form:"pageSize"`
}

// listPluginMarket 插件市场列表
func (s *Server) listPluginMarket(c *gin.Context) {
	if s.pluginSvc == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"code": 503, "message": "插件服务不可用"})
		return
	}
	var q pluginMarketQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	page, pageSize := normalizePage(q.Page, q.PageSize)
	list, total, err := s.pluginSvc.ListPluginMarket(c.Request.Context(), page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "data": list, "total": total})
}

// listInstalledPlugins 已安装插件列表
func (s *Server) listInstalledPlugins(c *gin.Context) {
	if s.pluginSvc == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"code": 503, "message": "插件服务不可用"})
		return
	}
	list, err := s.pluginSvc.ListInstalledPlugins(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "data": list})
}

// getPlugin 插件详情
func (s *Server) getPlugin(c *gin.Context) {
	if s.pluginSvc == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"code": 503, "message": "插件服务不可用"})
		return
	}
	pluginID := c.Param("pluginId")
	info, err := s.pluginSvc.GetPlugin(c.Request.Context(), pluginID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	if info == nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "插件不存在"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "data": info})
}

// installPlugin 安装插件
func (s *Server) installPlugin(c *gin.Context) {
	if s.pluginSvc == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"code": 503, "message": "插件服务不可用"})
		return
	}
	pluginID := c.Param("pluginId")
	var req types.InstallPluginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	req.PluginID = pluginID
	if err := s.pluginSvc.InstallPlugin(c.Request.Context(), req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "安装成功"})
}

// uninstallPlugin 卸载插件
func (s *Server) uninstallPlugin(c *gin.Context) {
	if s.pluginSvc == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"code": 503, "message": "插件服务不可用"})
		return
	}
	pluginID := c.Param("pluginId")
	if err := s.pluginSvc.UninstallPlugin(c.Request.Context(), pluginID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "卸载成功"})
}

// enablePlugin 启用插件
func (s *Server) enablePlugin(c *gin.Context) {
	if s.pluginSvc == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"code": 503, "message": "插件服务不可用"})
		return
	}
	pluginID := c.Param("pluginId")
	if err := s.pluginSvc.EnablePlugin(c.Request.Context(), pluginID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "启用成功"})
}

// disablePlugin 停用插件
func (s *Server) disablePlugin(c *gin.Context) {
	if s.pluginSvc == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"code": 503, "message": "插件服务不可用"})
		return
	}
	pluginID := c.Param("pluginId")
	if err := s.pluginSvc.DisablePlugin(c.Request.Context(), pluginID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "停用成功"})
}

// getPluginConfig 获取插件配置（从已安装插件列表中查找）
func (s *Server) getPluginConfig(c *gin.Context) {
	if s.pluginSvc == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"code": 503, "message": "插件服务不可用"})
		return
	}
	pluginID := c.Param("pluginId")
	list, err := s.pluginSvc.ListInstalledPlugins(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	for _, cfg := range list {
		if cfg.PluginID == pluginID {
			c.JSON(http.StatusOK, gin.H{"code": 200, "data": cfg})
			return
		}
	}
	c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "插件未安装或配置不存在"})
}

type savePluginConfigBody struct {
	Config string `json:"config" binding:"required"`
}

// savePluginConfig 保存插件配置
func (s *Server) savePluginConfig(c *gin.Context) {
	if s.pluginSvc == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"code": 503, "message": "插件服务不可用"})
		return
	}
	pluginID := c.Param("pluginId")
	var body savePluginConfigBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	if err := s.pluginSvc.UpdatePluginConfig(c.Request.Context(), pluginID, body.Config); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "保存成功"})
}

// normalizePage 规范化分页参数
func normalizePage(page, pageSize int) (int, int) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return page, pageSize
}


