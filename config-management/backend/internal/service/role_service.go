package service

import (
	"context"
	"errors"

	"github.com/zhiyunliu/distributed-workflow/config-management/backend/internal/sysmanager"
	"github.com/zhiyunliu/distributed-workflow/config-management/backend/internal/sysmodel"
	"github.com/zhiyunliu/distributed-workflow/config-management/backend/internal/sysrepo"
)

// ─── RoleService 内置实现 ─────────────────────────────────────────────────────

type roleService struct {
	roleRepo sysrepo.RoleRepo
}

// NewRoleService 创建内置角色服务
func NewRoleService(roleRepo sysrepo.RoleRepo) sysmanager.RoleService {
	return &roleService{roleRepo: roleRepo}
}

func (s *roleService) GetRoleByID(ctx context.Context, roleID int64) (*sysmodel.SystemRole, error) {
	return s.roleRepo.GetByID(ctx, roleID)
}

func (s *roleService) GetRoleByCode(ctx context.Context, roleCode string) (*sysmodel.SystemRole, error) {
	return s.roleRepo.GetByCode(ctx, roleCode)
}

func (s *roleService) ListRoles(ctx context.Context, filter sysmodel.RoleFilter, page, pageSize int) ([]*sysmodel.SystemRole, int64, error) {
	return s.roleRepo.List(ctx, filter, page, pageSize)
}

func (s *roleService) CreateRole(ctx context.Context, role *sysmodel.SystemRole) (int64, error) {
	if role.RoleName == "" || role.RoleCode == "" {
		return 0, errors.New("角色名称和编码不能为空")
	}
	if role.DataScope == 0 {
		role.DataScope = 1
	}
	if role.Status == 0 {
		role.Status = 1
	}
	return s.roleRepo.Create(ctx, role)
}

func (s *roleService) UpdateRole(ctx context.Context, role *sysmodel.SystemRole) error {
	if role.ID <= 0 {
		return errors.New("角色ID不能为空")
	}
	return s.roleRepo.Update(ctx, role)
}

func (s *roleService) DeleteRole(ctx context.Context, roleID int64) error {
	return s.roleRepo.Delete(ctx, roleID)
}

func (s *roleService) GetRoleMenus(ctx context.Context, roleID int64) ([]int64, error) {
	return s.roleRepo.GetMenuIDsByRoleID(ctx, roleID)
}

func (s *roleService) AssignRoleMenus(ctx context.Context, roleID int64, menuIDs []int64) error {
	return s.roleRepo.AssignMenus(ctx, roleID, menuIDs)
}

