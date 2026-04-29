package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	types "github.com/zhiyunliu/distributed-workflow/runtime-execution/pkg/types"
	api "github.com/zhiyunliu/distributed-workflow/runtime-execution/pkg/api"
)

// SetD6AnalyticsService 注入BI统计服务，并注册统计路由
func (s *Server) SetD6AnalyticsService(analyticsSvc api.AnalyticsService) {
	s.analyticsSvc = analyticsSvc
	s.registerD6StatsRoutes(s.engine)
}

func (s *Server) registerD6StatsRoutes(r *gin.Engine) {
	stats := r.Group("/api/stats")
	{
		stats.GET("/workflow/overview", s.getWorkflowOverview)
		stats.GET("/workflow/efficiency", s.getWorkflowEfficiency)
		stats.GET("/approval/performance", s.getApprovalPerformance)
		stats.GET("/instance/trend", s.getInstanceTrend)
		stats.POST("/refresh", s.refreshDailyStats)
	}
}

// ─── BI统计处理函数 ────────────────────────────────────────────────────────────

// getWorkflowOverview 工作流总览统计
func (s *Server) getWorkflowOverview(c *gin.Context) {
	if s.analyticsSvc == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"code": 503, "message": "统计服务不可用"})
		return
	}
	data, err := s.analyticsSvc.GetWorkflowOverview(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "data": data})
}

type statsEfficiencyQuery struct {
	StartDate     string `form:"startDate"`
	EndDate       string `form:"endDate"`
	WorkflowDefID string `form:"workflowDefID"`
}

// getWorkflowEfficiency 工作流效率分析
func (s *Server) getWorkflowEfficiency(c *gin.Context) {
	if s.analyticsSvc == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"code": 503, "message": "统计服务不可用"})
		return
	}
	var q statsEfficiencyQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	params := types.StatsQueryParams{
		StartDate:  q.StartDate,
		EndDate:    q.EndDate,
		WorkflowID: q.WorkflowDefID,
	}
	data, err := s.analyticsSvc.GetWorkflowEfficiency(c.Request.Context(), params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "data": data})
}

type statsDateRangeQuery struct {
	StartDate string `form:"startDate"`
	EndDate   string `form:"endDate"`
}

// getApprovalPerformance 审批绩效统计
func (s *Server) getApprovalPerformance(c *gin.Context) {
	if s.analyticsSvc == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"code": 503, "message": "统计服务不可用"})
		return
	}
	var q statsDateRangeQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	params := types.StatsQueryParams{
		StartDate: q.StartDate,
		EndDate:   q.EndDate,
	}
	data, err := s.analyticsSvc.GetApprovalPerformance(c.Request.Context(), params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "data": data})
}

type statsTrendQuery struct {
	StartDate string `form:"startDate"`
	EndDate   string `form:"endDate"`
	GroupBy   string `form:"groupBy"`
}

// getInstanceTrend 实例趋势数据
func (s *Server) getInstanceTrend(c *gin.Context) {
	if s.analyticsSvc == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"code": 503, "message": "统计服务不可用"})
		return
	}
	var q statsTrendQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	params := types.StatsQueryParams{
		StartDate: q.StartDate,
		EndDate:   q.EndDate,
		GroupBy:   q.GroupBy,
	}
	data, err := s.analyticsSvc.GetInstanceTrend(c.Request.Context(), params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "data": data})
}

type refreshStatsBody struct {
	Date string `json:"date" binding:"required"`
}

// refreshDailyStats 手动刷新日统计
func (s *Server) refreshDailyStats(c *gin.Context) {
	if s.analyticsSvc == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"code": 503, "message": "统计服务不可用"})
		return
	}
	var body refreshStatsBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	if err := s.analyticsSvc.RefreshDailyStats(c.Request.Context(), body.Date); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "刷新成功"})
}


