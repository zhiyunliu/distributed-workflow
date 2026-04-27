package http

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/zhiyunliu/distributed-workflow/jwtutil"
	"github.com/zhiyunliu/distributed-workflow/sysmanager"
	"github.com/zhiyunliu/distributed-workflow/sysmodel"
)

// ─── 注入与路由注册 ──────────────────────────────────────────────────────────

// SetSysManager 注入系统管理服务，注册 D4 路由（认证 + 系统管理）
func (s *Server) SetSysManager(mgr sysmanager.Manager, jwtSecret string) {
	s.sysMgr = mgr
	s.jwtSecret = jwtSecret
	s.registerD4Routes(s.engine)
}

func (s *Server) registerD4Routes(r *gin.Engine) {
	// ── 开放接口：登录 ─────────────────────────────────────────────────────
	auth := r.Group("/api/v1/auth")
	{
		auth.POST("/login", s.login)
	}

	// ── 需要认证的接口 ─────────────────────────────────────────────────────
	authorized := r.Group("/api/v1")
	authorized.Use(s.jwtAuthMiddleware())
	{
		// 认证相关
		authorized.POST("/auth/logout", s.logout)
		authorized.GET("/auth/user-info", s.getUserInfo)
		authorized.GET("/auth/menu", s.getUserMenu)
		authorized.PUT("/auth/reset-pwd", s.resetPassword)

		// 系统管理 - 用户
		authorized.GET("/system/user", s.listUsers)
		authorized.POST("/system/user", s.createUser)
		authorized.PUT("/system/user/:id", s.updateUser)
		authorized.DELETE("/system/user/:id", s.deleteUser)
		authorized.GET("/system/user/:id/roles", s.getUserRoles)
		authorized.PUT("/system/user/:id/roles", s.assignUserRoles)
		authorized.PUT("/system/user/:id/reset-pwd", s.adminResetUserPassword)

		// 系统管理 - 角色
		authorized.GET("/system/role", s.listRoles)
		authorized.POST("/system/role", s.createRole)
		authorized.PUT("/system/role/:id", s.updateRole)
		authorized.DELETE("/system/role/:id", s.deleteRole)
		authorized.GET("/system/role/:id/menus", s.getRoleMenus)
		authorized.PUT("/system/role/:id/menus", s.assignRoleMenus)

		// 系统管理 - 菜单
		authorized.GET("/system/menu", s.listMenus)
		authorized.GET("/system/menu/tree", s.getMenuTree)
		authorized.POST("/system/menu", s.createMenu)
		authorized.PUT("/system/menu/:id", s.updateMenu)
		authorized.DELETE("/system/menu/:id", s.deleteMenu)

		// 系统管理 - 数据字典
		authorized.GET("/system/dictionary/types", s.listDictTypes)
		authorized.GET("/system/dictionary/data/:type", s.listDictData)
		authorized.GET("/system/dictionary", s.pageDictData)
		authorized.POST("/system/dictionary", s.createDictData)
		authorized.PUT("/system/dictionary/:dicId", s.updateDictData)
		authorized.DELETE("/system/dictionary/:dicId", s.deleteDictData)
	}
}

// ─── JWT 中间件 ──────────────────────────────────────────────────────────────

func (s *Server) jwtAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := extractBearerToken(c)
		if token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "未登录或 token 已失效"})
			return
		}
		claims, err := jwtutil.Parse(token, s.jwtSecret)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "token 无效或已过期"})
			return
		}
		c.Set("userID", claims.UserID)
		c.Set("username", claims.Username)
		c.Next()
	}
}

func extractBearerToken(c *gin.Context) string {
	auth := c.GetHeader("Authorization")
	if auth == "" {
		return ""
	}
	parts := strings.SplitN(auth, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return strings.TrimSpace(parts[1])
}

func currentUserID(c *gin.Context) int64 {
	id, _ := c.Get("userID")
	uid, _ := id.(int64)
	return uid
}

// ─── 认证接口 ─────────────────────────────────────────────────────────────────

func (s *Server) login(c *gin.Context) {
	var req sysmodel.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误: " + err.Error()})
		return
	}
	resp, err := s.sysMgr.Auth().Login(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 401, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "登录成功", "data": resp})
}

func (s *Server) logout(c *gin.Context) {
	token := extractBearerToken(c)
	if err := s.sysMgr.Auth().Logout(c.Request.Context(), token); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "登出成功"})
}

func (s *Server) getUserInfo(c *gin.Context) {
	uid := currentUserID(c)
	user, err := s.sysMgr.User().GetUserByID(c.Request.Context(), uid)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": err.Error()})
		return
	}
	roles, _ := s.sysMgr.User().GetUserRoles(c.Request.Context(), uid)
	perms, _ := s.sysMgr.User().GetUserPermissions(c.Request.Context(), uid)
	c.JSON(http.StatusOK, gin.H{
		"code": 200, "data": gin.H{
			"user": user, "roles": roles, "perms": perms,
		},
	})
}

func (s *Server) getUserMenu(c *gin.Context) {
	uid := currentUserID(c)
	tree, err := s.sysMgr.Menu().GetUserMenuTree(c.Request.Context(), uid)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "data": tree})
}

func (s *Server) resetPassword(c *gin.Context) {
	var req struct {
		OldPassword string `json:"oldPassword" binding:"required"`
		NewPassword string `json:"newPassword" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	uid := currentUserID(c)
	if err := s.sysMgr.Auth().ChangePassword(c.Request.Context(), uid, req.OldPassword, req.NewPassword); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "密码修改成功"})
}

// ─── 用户管理接口 ─────────────────────────────────────────────────────────────

func (s *Server) listUsers(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	filter := sysmodel.UserFilter{
		Username: c.Query("username"),
		RealName: c.Query("realName"),
	}
	users, total, err := s.sysMgr.User().ListUsers(c.Request.Context(), filter, page, pageSize)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "data": gin.H{"list": users, "total": total}})
}

func (s *Server) createUser(c *gin.Context) {
	var user sysmodel.SystemUser
	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	id, err := s.sysMgr.User().CreateUser(c.Request.Context(), &user)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "data": gin.H{"id": id}})
}

func (s *Server) updateUser(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的用户ID"})
		return
	}
	var user sysmodel.SystemUser
	if err = c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	user.ID = id
	if err = s.sysMgr.User().UpdateUser(c.Request.Context(), &user); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "更新成功"})
}

func (s *Server) deleteUser(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的用户ID"})
		return
	}
	if err = s.sysMgr.User().DeleteUser(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "删除成功"})
}

func (s *Server) getUserRoles(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的用户ID"})
		return
	}
	roles, err := s.sysMgr.User().GetUserRoles(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "data": roles})
}

func (s *Server) assignUserRoles(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的用户ID"})
		return
	}
	var req struct {
		RoleIDs []int64 `json:"roleIds" binding:"required"`
	}
	if err = c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	if err = s.sysMgr.User().AssignRoles(c.Request.Context(), id, req.RoleIDs); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "分配角色成功"})
}

func (s *Server) adminResetUserPassword(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的用户ID"})
		return
	}
	var req struct {
		NewPassword string `json:"newPassword" binding:"required"`
	}
	if err = c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	if err = s.sysMgr.User().ResetPassword(c.Request.Context(), id, req.NewPassword); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "密码重置成功"})
}

// ─── 角色管理接口 ─────────────────────────────────────────────────────────────

func (s *Server) listRoles(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	filter := sysmodel.RoleFilter{RoleName: c.Query("roleName")}
	roles, total, err := s.sysMgr.Role().ListRoles(c.Request.Context(), filter, page, pageSize)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "data": gin.H{"list": roles, "total": total}})
}

func (s *Server) createRole(c *gin.Context) {
	var role sysmodel.SystemRole
	if err := c.ShouldBindJSON(&role); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	id, err := s.sysMgr.Role().CreateRole(c.Request.Context(), &role)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "data": gin.H{"id": id}})
}

func (s *Server) updateRole(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的角色ID"})
		return
	}
	var role sysmodel.SystemRole
	if err = c.ShouldBindJSON(&role); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	role.ID = id
	if err = s.sysMgr.Role().UpdateRole(c.Request.Context(), &role); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "更新成功"})
}

func (s *Server) deleteRole(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的角色ID"})
		return
	}
	if err = s.sysMgr.Role().DeleteRole(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "删除成功"})
}

func (s *Server) getRoleMenus(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的角色ID"})
		return
	}
	menuIDs, err := s.sysMgr.Role().GetRoleMenus(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "data": menuIDs})
}

func (s *Server) assignRoleMenus(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的角色ID"})
		return
	}
	var req struct {
		MenuIDs []int64 `json:"menuIds"`
	}
	if err = c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	if err = s.sysMgr.Role().AssignRoleMenus(c.Request.Context(), id, req.MenuIDs); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "分配菜单成功"})
}

// ─── 菜单管理接口 ─────────────────────────────────────────────────────────────

func (s *Server) listMenus(c *gin.Context) {
	filter := sysmodel.MenuFilter{MenuName: c.Query("menuName")}
	menus, err := s.sysMgr.Menu().ListMenus(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "data": menus})
}

func (s *Server) getMenuTree(c *gin.Context) {
	filter := sysmodel.MenuFilter{MenuName: c.Query("menuName")}
	tree, err := s.sysMgr.Menu().GetMenuTree(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "data": tree})
}

func (s *Server) createMenu(c *gin.Context) {
	var menu sysmodel.SystemMenu
	if err := c.ShouldBindJSON(&menu); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	id, err := s.sysMgr.Menu().CreateMenu(c.Request.Context(), &menu)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "data": gin.H{"id": id}})
}

func (s *Server) updateMenu(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的菜单ID"})
		return
	}
	var menu sysmodel.SystemMenu
	if err = c.ShouldBindJSON(&menu); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	menu.ID = id
	if err = s.sysMgr.Menu().UpdateMenu(c.Request.Context(), &menu); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "更新成功"})
}

func (s *Server) deleteMenu(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的菜单ID"})
		return
	}
	if err = s.sysMgr.Menu().DeleteMenu(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "删除成功"})
}

// ─── 数据字典接口 ─────────────────────────────────────────────────────────────

func (s *Server) listDictTypes(c *gin.Context) {
	types, err := s.sysMgr.Dictionary().ListDictTypes(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "data": types})
}

func (s *Server) listDictData(c *gin.Context) {
	dictType := c.Param("type")
	items, err := s.sysMgr.Dictionary().ListDictData(c.Request.Context(), dictType)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "data": items})
}

func (s *Server) pageDictData(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	filter := sysmodel.DictFilter{
		DictType:  c.Query("dictType"),
		DictGroup: c.Query("dictGroup"),
	}
	items, total, err := s.sysMgr.Dictionary().PageDictData(c.Request.Context(), filter, page, pageSize)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "data": gin.H{"list": items, "total": total}})
}

func (s *Server) createDictData(c *gin.Context) {
	var item sysmodel.DictionaryItem
	if err := c.ShouldBindJSON(&item); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	id, err := s.sysMgr.Dictionary().CreateDictData(c.Request.Context(), &item)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "data": gin.H{"dicId": id}})
}

func (s *Server) updateDictData(c *gin.Context) {
	dicID, err := strconv.ParseInt(c.Param("dicId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的字典ID"})
		return
	}
	var item sysmodel.DictionaryItem
	if err = c.ShouldBindJSON(&item); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	item.DicID = dicID
	if err = s.sysMgr.Dictionary().UpdateDictData(c.Request.Context(), &item); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "更新成功"})
}

func (s *Server) deleteDictData(c *gin.Context) {
	dicID, err := strconv.ParseInt(c.Param("dicId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的字典ID"})
		return
	}
	if err = s.sysMgr.Dictionary().DeleteDictData(c.Request.Context(), dicID); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "删除成功"})
}
