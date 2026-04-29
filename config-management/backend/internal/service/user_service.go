package service

import (
	"context"
	"errors"
	"fmt"

	"golang.org/x/crypto/bcrypt"

	"github.com/zhiyunliu/distributed-workflow/config-management/backend/internal/sysmanager"
	"github.com/zhiyunliu/distributed-workflow/config-management/backend/internal/sysmodel"
	"github.com/zhiyunliu/distributed-workflow/config-management/backend/internal/sysrepo"
)

// ─── UserService 内置实现 ─────────────────────────────────────────────────────

type userService struct {
	userRepo sysrepo.UserRepo
	roleRepo sysrepo.RoleRepo
}

// NewUserService 创建内置用户服务
func NewUserService(userRepo sysrepo.UserRepo, roleRepo sysrepo.RoleRepo) sysmanager.UserService {
	return &userService{userRepo: userRepo, roleRepo: roleRepo}
}

func (s *userService) GetUserByID(ctx context.Context, userID int64) (*sysmodel.SystemUser, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, fmt.Errorf("user %d not found", userID)
	}
	user.Password = ""
	return user, nil
}

func (s *userService) GetUserByUsername(ctx context.Context, username string) (*sysmodel.SystemUser, error) {
	user, err := s.userRepo.GetByUsername(ctx, username)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, fmt.Errorf("user %q not found", username)
	}
	user.Password = ""
	return user, nil
}

func (s *userService) ListUsers(ctx context.Context, filter sysmodel.UserFilter, page, pageSize int) ([]*sysmodel.SystemUser, int64, error) {
	users, total, err := s.userRepo.List(ctx, filter, page, pageSize)
	if err != nil {
		return nil, 0, err
	}
	for _, u := range users {
		u.Password = ""
	}
	return users, total, nil
}

func (s *userService) CreateUser(ctx context.Context, user *sysmodel.SystemUser) (int64, error) {
	if user.Username == "" {
		return 0, errors.New("用户名不能为空")
	}
	if user.Password == "" {
		return 0, errors.New("密码不能为空")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return 0, fmt.Errorf("hash password: %w", err)
	}
	user.Password = string(hash)
	if user.Status == 0 {
		user.Status = 1 // 默认启用
	}
	return s.userRepo.Create(ctx, user)
}

func (s *userService) UpdateUser(ctx context.Context, user *sysmodel.SystemUser) error {
	if user.ID <= 0 {
		return errors.New("用户ID不能为空")
	}
	return s.userRepo.Update(ctx, user)
}

func (s *userService) DeleteUser(ctx context.Context, userID int64) error {
	return s.userRepo.Delete(ctx, userID)
}

func (s *userService) GetUserRoles(ctx context.Context, userID int64) ([]*sysmodel.SystemRole, error) {
	return s.roleRepo.GetRolesByUserID(ctx, userID)
}

func (s *userService) GetUserPermissions(ctx context.Context, userID int64) ([]string, error) {
	return nil, errors.New("请使用 AuthService.Login 获取权限列表")
}

func (s *userService) ResetPassword(ctx context.Context, userID int64, newPassword string) error {
	if newPassword == "" {
		return errors.New("新密码不能为空")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	return s.userRepo.UpdatePassword(ctx, userID, string(hash))
}

func (s *userService) AssignRoles(ctx context.Context, userID int64, roleIDs []int64) error {
	return s.roleRepo.AssignUserRoles(ctx, userID, roleIDs)
}

