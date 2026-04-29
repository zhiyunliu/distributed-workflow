package service

import (
	"context"

	"github.com/zhiyunliu/distributed-workflow/config-management/backend/internal/sysmanager"
	"github.com/zhiyunliu/distributed-workflow/config-management/backend/internal/sysmodel"
	"github.com/zhiyunliu/distributed-workflow/config-management/backend/internal/sysrepo"
)

// ─── PermissionService 内置实现 ───────────────────────────────────────────────

type permissionService struct {
	userRepo sysrepo.UserRepo
	roleRepo sysrepo.RoleRepo
	menuRepo sysrepo.MenuRepo
}

// NewPermissionService 创建内置权限校验服务
func NewPermissionService(userRepo sysrepo.UserRepo, roleRepo sysrepo.RoleRepo, menuRepo sysrepo.MenuRepo) sysmanager.PermissionService {
	return &permissionService{userRepo: userRepo, roleRepo: roleRepo, menuRepo: menuRepo}
}

func (s *permissionService) HasPermission(ctx context.Context, userID int64, permission string) (bool, error) {
	perms, err := s.getUserPermsInternal(ctx, userID)
	if err != nil {
		return false, err
	}
	for _, p := range perms {
		if p == "*:*:*" || p == permission {
			return true, nil
		}
	}
	return false, nil
}

func (s *permissionService) HasAnyPermission(ctx context.Context, userID int64, permissions []string) (bool, error) {
	perms, err := s.getUserPermsInternal(ctx, userID)
	if err != nil {
		return false, err
	}
	permSet := toSet(perms)
	if permSet["*:*:*"] {
		return true, nil
	}
	for _, p := range permissions {
		if permSet[p] {
			return true, nil
		}
	}
	return false, nil
}

func (s *permissionService) HasAllPermissions(ctx context.Context, userID int64, permissions []string) (bool, error) {
	perms, err := s.getUserPermsInternal(ctx, userID)
	if err != nil {
		return false, err
	}
	permSet := toSet(perms)
	if permSet["*:*:*"] {
		return true, nil
	}
	for _, p := range permissions {
		if !permSet[p] {
			return false, nil
		}
	}
	return true, nil
}

func (s *permissionService) HasRole(ctx context.Context, userID int64, roleCode string) (bool, error) {
	roles, err := s.roleRepo.GetRolesByUserID(ctx, userID)
	if err != nil {
		return false, err
	}
	for _, r := range roles {
		if r.RoleCode == roleCode {
			return true, nil
		}
	}
	return false, nil
}

func (s *permissionService) HasAnyRole(ctx context.Context, userID int64, roleCodes []string) (bool, error) {
	roles, err := s.roleRepo.GetRolesByUserID(ctx, userID)
	if err != nil {
		return false, err
	}
	codeSet := toSet(roleCodes)
	for _, r := range roles {
		if codeSet[r.RoleCode] {
			return true, nil
		}
	}
	return false, nil
}

func (s *permissionService) CheckDataScope(ctx context.Context, userID int64) (*sysmodel.DataScope, error) {
	roles, err := s.roleRepo.GetRolesByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	// 取最宽松的数据权限：1=全量 > 2=本部门 > 3=个人
	minScope := 3
	for _, r := range roles {
		if r.DataScope < minScope {
			minScope = r.DataScope
		}
	}
	return &sysmodel.DataScope{
		ScopeType: minScope,
		UserID:    userID,
	}, nil
}

func (s *permissionService) getUserPermsInternal(ctx context.Context, userID int64) ([]string, error) {
	roles, err := s.roleRepo.GetRolesByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return getUserPerms(ctx, userID, roles, s.menuRepo, s.roleRepo)
}

func toSet(items []string) map[string]bool {
	m := make(map[string]bool, len(items))
	for _, v := range items {
		m[v] = true
	}
	return m
}

