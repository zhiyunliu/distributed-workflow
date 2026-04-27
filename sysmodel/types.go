// Package sysmodel 系统管理通用数据模型（D4新增）
// 定义与具体实现无关的系统用户、角色、菜单、权限等通用结构，
// 供接口层、内置实现和扩展实现共同使用。
package sysmodel

import "time"

// ─── 用户 ─────────────────────────────────────────────────────────────────────

// SystemUser 系统用户通用结构
type SystemUser struct {
	ID         int64                  `json:"id"`
	Username   string                 `json:"username"`
	Password   string                 `json:"-"` // 不序列化到响应
	RealName   string                 `json:"realName"`
	Email      string                 `json:"email"`
	Phone      string                 `json:"phone"`
	Avatar     string                 `json:"avatar"`
	Status     int                    `json:"status"` // 1=启用 0=禁用
	DeptID     int64                  `json:"deptId"`
	CreateTime time.Time              `json:"createTime"`
	UpdateTime time.Time              `json:"updateTime"`
	Extra      map[string]interface{} `json:"extra,omitempty"` // 扩展字段
}

// UserFilter 用户查询过滤器
type UserFilter struct {
	Username string
	RealName string
	Status   *int
	DeptID   *int64
}

// ─── 角色 ─────────────────────────────────────────────────────────────────────

// SystemRole 系统角色通用结构
type SystemRole struct {
	ID          int64     `json:"id"`
	RoleName    string    `json:"roleName"`
	RoleCode    string    `json:"roleCode"`
	Description string    `json:"description"`
	Status      int       `json:"status"`    // 1=启用 0=禁用
	DataScope   int       `json:"dataScope"` // 1=全量 2=本部门 3=个人
	CreateTime  time.Time `json:"createTime"`
}

// RoleFilter 角色查询过滤器
type RoleFilter struct {
	RoleName string
	Status   *int
}

// ─── 菜单 ─────────────────────────────────────────────────────────────────────

// SystemMenu 系统菜单通用结构
type SystemMenu struct {
	ID        int64         `json:"id"`
	ParentID  int64         `json:"parentId"`
	MenuName  string        `json:"menuName"`
	MenuType  string        `json:"menuType"` // M=目录 C=菜单 F=按钮
	Path      string        `json:"path"`
	Component string        `json:"component"`
	Perms     string        `json:"perms"` // 权限标识
	Icon      string        `json:"icon"`
	Sort      int           `json:"sort"`
	Visible   int           `json:"visible"` // 1=显示 0=隐藏
	IsFrame   int           `json:"isFrame"` // 1=外链 0=内链
	Children  []*SystemMenu `json:"children,omitempty"`
}

// MenuFilter 菜单查询过滤器
type MenuFilter struct {
	MenuName string
	Visible  *int
}

// ─── 认证 ─────────────────────────────────────────────────────────────────────

// LoginRequest 登录请求通用结构
type LoginRequest struct {
	Username  string                 `json:"username"  binding:"required"`
	Password  string                 `json:"password"  binding:"required"`
	GrantType string                 `json:"grantType"` // password/authorization_code/client_credentials
	Extra     map[string]interface{} `json:"extra,omitempty"`
}

// LoginResponse 登录响应通用结构
type LoginResponse struct {
	Token     string        `json:"token"`
	ExpiresIn int64         `json:"expiresIn"` // 过期时间，单位秒
	User      *SystemUser   `json:"user"`
	Roles     []*SystemRole `json:"roles"`
	Perms     []string      `json:"perms"`
}

// ─── 权限 ─────────────────────────────────────────────────────────────────────

// DataScope 数据权限范围
type DataScope struct {
	ScopeType int     `json:"scopeType"` // 1=全量 2=本部门 3=个人
	DeptIDs   []int64 `json:"deptIds"`
	UserID    int64   `json:"userId"`
}

// ─── 数据字典 ─────────────────────────────────────────────────────────────────

// DictionaryItem 数据字典项
type DictionaryItem struct {
	DicID      int64     `json:"dicId"`
	DictType   string    `json:"dictType"`  // 字典类型
	DictName   string    `json:"dictName"`  // 显示名称
	DictValue  string    `json:"dictValue"` // 字典值
	DictGroup  string    `json:"dictGroup"` // 分组，默认 *
	Sort       int       `json:"sort"`
	Status     int       `json:"status"` // 1=启用 0=禁用
	Remark     string    `json:"remark"`
	CreateTime time.Time `json:"createTime"`
}

// DictFilter 字典查询过滤器
type DictFilter struct {
	DictType  string
	DictGroup string
	Status    *int
}

// ─── 系统配置 ──────────────────────────────────────────────────────────────────

// AuthConfig 认证配置
type AuthConfig struct {
	// Type 实现类型：builtin/ldap/oauth2/custom
	Type string `json:"type"`
	// Secret JWT 签名密钥
	Secret string `json:"secret"`
	// ExpireHours Token 有效时长（小时）
	ExpireHours int `json:"expireHours"`
}

// DefaultAuthConfig 默认认证配置
var DefaultAuthConfig = AuthConfig{
	Type:        "builtin",
	Secret:      "distributed-workflow-default-secret-change-me",
	ExpireHours: 24,
}
