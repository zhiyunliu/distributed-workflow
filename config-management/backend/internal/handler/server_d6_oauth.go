package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	types "github.com/zhiyunliu/distributed-workflow/runtime-execution/internal/common/types"
	api "github.com/zhiyunliu/distributed-workflow/runtime-execution/pkg/api"
)

// SetD6OAuthService 注入OAuth服务，并注册OAuth路由
func (s *Server) SetD6OAuthService(oauthSvc api.OAuthService) {
	s.oauthSvc = oauthSvc
	s.registerD6OAuthRoutes(s.engine)
}

func (s *Server) registerD6OAuthRoutes(r *gin.Engine) {
	oauth := r.Group("/api/oauth")
	{
		// 钉钉
		oauth.GET("/dingtalk/auth-url", s.getDingTalkAuthURL)
		oauth.POST("/dingtalk/callback", s.handleDingTalkCallback)
		// 企业微信
		oauth.GET("/wechat/auth-url", s.getWechatWorkAuthURL)
		oauth.POST("/wechat/callback", s.handleWechatWorkCallback)
		// 绑定与查询（需要JWT认证）
		authGroup := oauth.Group("")
		authGroup.Use(s.jwtAuthMiddleware())
		{
			authGroup.POST("/bind", s.bindOAuthUser)
		}
		// 查询绑定列表（开放，供管理员查询）
		oauth.GET("/bindings/:sysUserId", s.getOAuthBindings)
		// OAuth登录
		oauth.POST("/login", s.oauthLogin)
	}
}

// ─── OAuth处理函数 ────────────────────────────────────────────────────────────

type authURLQuery struct {
	RedirectURI string `form:"redirectUri" binding:"required"`
	State       string `form:"state"`
}

// getDingTalkAuthURL 获取钉钉授权URL
func (s *Server) getDingTalkAuthURL(c *gin.Context) {
	if s.oauthSvc == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"code": 503, "message": "OAuth服务不可用"})
		return
	}
	var q authURLQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	url, err := s.oauthSvc.GetDingTalkAuthURL(c.Request.Context(), q.RedirectURI, q.State)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "data": gin.H{"authUrl": url}})
}

type callbackBody struct {
	Code string `json:"code" binding:"required"`
}

// handleDingTalkCallback 处理钉钉OAuth回调
func (s *Server) handleDingTalkCallback(c *gin.Context) {
	if s.oauthSvc == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"code": 503, "message": "OAuth服务不可用"})
		return
	}
	var body callbackBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	userInfo, err := s.oauthSvc.HandleDingTalkCallback(c.Request.Context(), body.Code)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "data": userInfo})
}

// getWechatWorkAuthURL 获取企业微信授权URL
func (s *Server) getWechatWorkAuthURL(c *gin.Context) {
	if s.oauthSvc == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"code": 503, "message": "OAuth服务不可用"})
		return
	}
	var q authURLQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	url, err := s.oauthSvc.GetWechatWorkAuthURL(c.Request.Context(), q.RedirectURI, q.State)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "data": gin.H{"authUrl": url}})
}

// handleWechatWorkCallback 处理企业微信OAuth回调
func (s *Server) handleWechatWorkCallback(c *gin.Context) {
	if s.oauthSvc == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"code": 503, "message": "OAuth服务不可用"})
		return
	}
	var body callbackBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	userInfo, err := s.oauthSvc.HandleWechatWorkCallback(c.Request.Context(), body.Code)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "data": userInfo})
}

type bindOAuthBody struct {
	SysUserID string               `json:"sysUserId" binding:"required"`
	UserInfo  *types.OAuthUserInfo `json:"userInfo" binding:"required"`
}

// bindOAuthUser 绑定OAuth账号到系统用户（需要JWT认证）
func (s *Server) bindOAuthUser(c *gin.Context) {
	if s.oauthSvc == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"code": 503, "message": "OAuth服务不可用"})
		return
	}
	var body bindOAuthBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	if err := s.oauthSvc.BindUser(c.Request.Context(), body.SysUserID, body.UserInfo); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "绑定成功"})
}

// getOAuthBindings 获取用户的所有平台绑定列表
func (s *Server) getOAuthBindings(c *gin.Context) {
	if s.oauthSvc == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"code": 503, "message": "OAuth服务不可用"})
		return
	}
	sysUserID := c.Param("sysUserId")
	list, err := s.oauthSvc.GetBindingBySysUser(c.Request.Context(), sysUserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "data": list})
}

type oauthLoginBody struct {
	Platform string `json:"platform" binding:"required"`
	Code     string `json:"code" binding:"required"`
}

// oauthLogin OAuth一键登录：根据平台换取用户信息并查询绑定关系
func (s *Server) oauthLogin(c *gin.Context) {
	if s.oauthSvc == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"code": 503, "message": "OAuth服务不可用"})
		return
	}
	var body oauthLoginBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	var (
		userInfo *types.OAuthUserInfo
		err      error
	)
	switch body.Platform {
	case types.OAuthPlatformDingTalk:
		userInfo, err = s.oauthSvc.HandleDingTalkCallback(c.Request.Context(), body.Code)
	case types.OAuthPlatformWechatWork:
		userInfo, err = s.oauthSvc.HandleWechatWorkCallback(c.Request.Context(), body.Code)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "不支持的平台: " + body.Platform})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	// 查询系统用户绑定关系
	binding, err := s.oauthSvc.GetBindingByOpenID(c.Request.Context(), body.Platform, userInfo.OpenID)
	if err != nil {
		// 未绑定，返回 OAuth 用户信息供前端引导绑定
		c.JSON(http.StatusOK, gin.H{
			"code":    202,
			"message": "账号未绑定，请先绑定系统账号",
			"data":    userInfo,
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "data": binding})
}
