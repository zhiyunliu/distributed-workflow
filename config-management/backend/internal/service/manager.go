package service

import (
	"github.com/zhiyunliu/distributed-workflow/config-management/backend/internal/sysmanager"
	"github.com/zhiyunliu/distributed-workflow/config-management/backend/internal/sysmodel"
	"github.com/zhiyunliu/distributed-workflow/config-management/backend/internal/sysrepo"
)

// manager 系统管理服务聚合（内置实现）
type manager struct {
	userSvc sysmanager.UserService
	roleSvc sysmanager.RoleService
	menuSvc sysmanager.MenuService
	authSvc sysmanager.AuthService
	permSvc sysmanager.PermissionService
	dictSvc sysmanager.DictionaryService
}

// NewManager 创建内置系统管理服务聚合体
func NewManager(
	userRepo sysrepo.UserRepo,
	roleRepo sysrepo.RoleRepo,
	menuRepo sysrepo.MenuRepo,
	dictRepo sysrepo.DictRepo,
	cfg sysmodel.AuthConfig,
) sysmanager.Manager {
	return &manager{
		userSvc: NewUserService(userRepo, roleRepo),
		roleSvc: NewRoleService(roleRepo),
		menuSvc: NewMenuService(menuRepo),
		authSvc: NewAuthService(userRepo, roleRepo, menuRepo, cfg),
		permSvc: NewPermissionService(userRepo, roleRepo, menuRepo),
		dictSvc: NewDictionaryService(dictRepo),
	}
}

func (m *manager) User() sysmanager.UserService             { return m.userSvc }
func (m *manager) Role() sysmanager.RoleService             { return m.roleSvc }
func (m *manager) Menu() sysmanager.MenuService             { return m.menuSvc }
func (m *manager) Auth() sysmanager.AuthService             { return m.authSvc }
func (m *manager) Permission() sysmanager.PermissionService { return m.permSvc }
func (m *manager) Dictionary() sysmanager.DictionaryService { return m.dictSvc }
