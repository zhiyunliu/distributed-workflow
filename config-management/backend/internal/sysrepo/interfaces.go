// Package sysrepo 系统管理存储层接口（D4新增）
// 定义系统管理相关的数据库操作接口，与业务无关。
package sysrepo

import (
	"context"

	"github.com/zhiyunliu/distributed-workflow/config-management/backend/internal/sysmodel"
)

// UserRepo 用户存储接口
type UserRepo interface {
	GetByID(ctx context.Context, id int64) (*sysmodel.SystemUser, error)
	GetByUsername(ctx context.Context, username string) (*sysmodel.SystemUser, error)
	List(ctx context.Context, filter sysmodel.UserFilter, page, pageSize int) ([]*sysmodel.SystemUser, int64, error)
	Create(ctx context.Context, user *sysmodel.SystemUser) (int64, error)
	Update(ctx context.Context, user *sysmodel.SystemUser) error
	Delete(ctx context.Context, id int64) error
	UpdatePassword(ctx context.Context, id int64, passwordHash string) error
}

// RoleRepo 角色存储接口
type RoleRepo interface {
	GetByID(ctx context.Context, id int64) (*sysmodel.SystemRole, error)
	GetByCode(ctx context.Context, code string) (*sysmodel.SystemRole, error)
	List(ctx context.Context, filter sysmodel.RoleFilter, page, pageSize int) ([]*sysmodel.SystemRole, int64, error)
	Create(ctx context.Context, role *sysmodel.SystemRole) (int64, error)
	Update(ctx context.Context, role *sysmodel.SystemRole) error
	Delete(ctx context.Context, id int64) error
	GetRolesByUserID(ctx context.Context, userID int64) ([]*sysmodel.SystemRole, error)
	GetMenuIDsByRoleID(ctx context.Context, roleID int64) ([]int64, error)
	AssignMenus(ctx context.Context, roleID int64, menuIDs []int64) error
	AssignUserRoles(ctx context.Context, userID int64, roleIDs []int64) error
}

// MenuRepo 菜单存储接口
type MenuRepo interface {
	GetByID(ctx context.Context, id int64) (*sysmodel.SystemMenu, error)
	List(ctx context.Context, filter sysmodel.MenuFilter) ([]*sysmodel.SystemMenu, error)
	GetByUserID(ctx context.Context, userID int64) ([]*sysmodel.SystemMenu, error)
	Create(ctx context.Context, menu *sysmodel.SystemMenu) (int64, error)
	Update(ctx context.Context, menu *sysmodel.SystemMenu) error
	Delete(ctx context.Context, id int64) error
}

// DictRepo 字典存储接口
type DictRepo interface {
	ListTypes(ctx context.Context) ([]string, error)
	ListByType(ctx context.Context, dictType string) ([]*sysmodel.DictionaryItem, error)
	Page(ctx context.Context, filter sysmodel.DictFilter, page, pageSize int) ([]*sysmodel.DictionaryItem, int64, error)
	Create(ctx context.Context, item *sysmodel.DictionaryItem) (int64, error)
	Update(ctx context.Context, item *sysmodel.DictionaryItem) error
	Delete(ctx context.Context, dicID int64) error
}
