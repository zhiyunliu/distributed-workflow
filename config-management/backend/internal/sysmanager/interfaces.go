// Package sysmanager 系统管理抽象层接口定义（D4新增）
// 定义 UserService、RoleService、MenuService、AuthService、PermissionService
// 以及字典服务接口，支持内置实现和外部扩展。
package sysmanager

import (
	"context"

	"github.com/zhiyunliu/distributed-workflow/config-management/backend/internal/sysmodel"
)

// UserService 用户服务接口
type UserService interface {
	// GetUserByID 根据用户ID获取用户信息
	GetUserByID(ctx context.Context, userID int64) (*sysmodel.SystemUser, error)
	// GetUserByUsername 根据用户名获取用户信息
	GetUserByUsername(ctx context.Context, username string) (*sysmodel.SystemUser, error)
	// ListUsers 分页查询用户列表
	ListUsers(ctx context.Context, filter sysmodel.UserFilter, page, pageSize int) ([]*sysmodel.SystemUser, int64, error)
	// CreateUser 创建用户（仅内置实现支持）
	CreateUser(ctx context.Context, user *sysmodel.SystemUser) (int64, error)
	// UpdateUser 更新用户（仅内置实现支持）
	UpdateUser(ctx context.Context, user *sysmodel.SystemUser) error
	// DeleteUser 删除用户（仅内置实现支持）
	DeleteUser(ctx context.Context, userID int64) error
	// GetUserRoles 获取用户的角色列表
	GetUserRoles(ctx context.Context, userID int64) ([]*sysmodel.SystemRole, error)
	// GetUserPermissions 获取用户的权限标识列表
	GetUserPermissions(ctx context.Context, userID int64) ([]string, error)
	// ResetPassword 重置用户密码（仅内置实现支持）
	ResetPassword(ctx context.Context, userID int64, newPassword string) error
	// AssignRoles 为用户分配角色（仅内置实现支持）
	AssignRoles(ctx context.Context, userID int64, roleIDs []int64) error
}

// RoleService 角色服务接口
type RoleService interface {
	// GetRoleByID 根据角色ID获取角色信息
	GetRoleByID(ctx context.Context, roleID int64) (*sysmodel.SystemRole, error)
	// GetRoleByCode 根据角色编码获取角色信息
	GetRoleByCode(ctx context.Context, roleCode string) (*sysmodel.SystemRole, error)
	// ListRoles 分页查询角色列表
	ListRoles(ctx context.Context, filter sysmodel.RoleFilter, page, pageSize int) ([]*sysmodel.SystemRole, int64, error)
	// CreateRole 创建角色（仅内置实现支持）
	CreateRole(ctx context.Context, role *sysmodel.SystemRole) (int64, error)
	// UpdateRole 更新角色（仅内置实现支持）
	UpdateRole(ctx context.Context, role *sysmodel.SystemRole) error
	// DeleteRole 删除角色（仅内置实现支持）
	DeleteRole(ctx context.Context, roleID int64) error
	// GetRoleMenus 获取角色的菜单ID列表
	GetRoleMenus(ctx context.Context, roleID int64) ([]int64, error)
	// AssignRoleMenus 为角色分配菜单（仅内置实现支持）
	AssignRoleMenus(ctx context.Context, roleID int64, menuIDs []int64) error
}

// MenuService 菜单服务接口
type MenuService interface {
	// GetMenuByID 根据菜单ID获取菜单信息
	GetMenuByID(ctx context.Context, menuID int64) (*sysmodel.SystemMenu, error)
	// ListMenus 查询菜单列表（平铺）
	ListMenus(ctx context.Context, filter sysmodel.MenuFilter) ([]*sysmodel.SystemMenu, error)
	// GetMenuTree 获取菜单树
	GetMenuTree(ctx context.Context, filter sysmodel.MenuFilter) ([]*sysmodel.SystemMenu, error)
	// GetUserMenuTree 获取用户的菜单树（根据角色权限过滤）
	GetUserMenuTree(ctx context.Context, userID int64) ([]*sysmodel.SystemMenu, error)
	// CreateMenu 创建菜单（仅内置实现支持）
	CreateMenu(ctx context.Context, menu *sysmodel.SystemMenu) (int64, error)
	// UpdateMenu 更新菜单（仅内置实现支持）
	UpdateMenu(ctx context.Context, menu *sysmodel.SystemMenu) error
	// DeleteMenu 删除菜单（仅内置实现支持）
	DeleteMenu(ctx context.Context, menuID int64) error
}

// AuthService 认证服务接口
type AuthService interface {
	// Login 用户登录，返回token和用户信息
	Login(ctx context.Context, req *sysmodel.LoginRequest) (*sysmodel.LoginResponse, error)
	// Logout 用户登出，销毁token
	Logout(ctx context.Context, token string) error
	// ValidateToken 验证token有效性，返回用户信息
	ValidateToken(ctx context.Context, token string) (*sysmodel.SystemUser, error)
	// RefreshToken 刷新token
	RefreshToken(ctx context.Context, token string) (*sysmodel.LoginResponse, error)
	// ChangePassword 修改密码（仅内置实现支持）
	ChangePassword(ctx context.Context, userID int64, oldPassword, newPassword string) error
}

// PermissionService 权限校验服务接口
type PermissionService interface {
	// HasPermission 校验用户是否有指定权限
	HasPermission(ctx context.Context, userID int64, permission string) (bool, error)
	// HasAnyPermission 校验用户是否有任意一个指定权限
	HasAnyPermission(ctx context.Context, userID int64, permissions []string) (bool, error)
	// HasAllPermissions 校验用户是否有所有指定权限
	HasAllPermissions(ctx context.Context, userID int64, permissions []string) (bool, error)
	// HasRole 校验用户是否有指定角色
	HasRole(ctx context.Context, userID int64, roleCode string) (bool, error)
	// HasAnyRole 校验用户是否有任意一个指定角色
	HasAnyRole(ctx context.Context, userID int64, roleCodes []string) (bool, error)
	// CheckDataScope 校验数据权限，返回用户可访问的数据范围
	CheckDataScope(ctx context.Context, userID int64) (*sysmodel.DataScope, error)
}

// DictionaryService 数据字典服务接口
type DictionaryService interface {
	// ListDictTypes 查询字典类型列表（去重）
	ListDictTypes(ctx context.Context) ([]string, error)
	// ListDictData 查询指定类型的字典数据列表
	ListDictData(ctx context.Context, dictType string) ([]*sysmodel.DictionaryItem, error)
	// PageDictData 分页查询字典数据
	PageDictData(ctx context.Context, filter sysmodel.DictFilter, page, pageSize int) ([]*sysmodel.DictionaryItem, int64, error)
	// CreateDictData 新增字典数据
	CreateDictData(ctx context.Context, item *sysmodel.DictionaryItem) (int64, error)
	// UpdateDictData 修改字典数据
	UpdateDictData(ctx context.Context, item *sysmodel.DictionaryItem) error
	// DeleteDictData 删除字典数据
	DeleteDictData(ctx context.Context, dicID int64) error
}

// Manager 系统管理服务聚合接口（方便整体注入）
type Manager interface {
	User() UserService
	Role() RoleService
	Menu() MenuService
	Auth() AuthService
	Permission() PermissionService
	Dictionary() DictionaryService
}

