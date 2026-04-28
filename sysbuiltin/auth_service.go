// Package builtin 系统管理内置实现（D4新增）
// 基于 SQL Server + bcrypt + HS256 JWT 实现完整的用户、角色、菜单、权限管理。
package builtin

import (
	"context"
	"errors"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/zhiyunliu/distributed-workflow/jwtutil"
	"github.com/zhiyunliu/distributed-workflow/sysmanager"
	"github.com/zhiyunliu/distributed-workflow/sysmodel"
	"github.com/zhiyunliu/distributed-workflow/sysrepo"
)

// ─── AuthService 内置实现 ─────────────────────────────────────────────────────

type authService struct {
	userRepo sysrepo.UserRepo
	roleRepo sysrepo.RoleRepo
	menuRepo sysrepo.MenuRepo
	cfg      sysmodel.AuthConfig
}

// NewAuthService 创建内置认证服务
func NewAuthService(
	userRepo sysrepo.UserRepo,
	roleRepo sysrepo.RoleRepo,
	menuRepo sysrepo.MenuRepo,
	cfg sysmodel.AuthConfig,
) sysmanager.AuthService {
	if cfg.ExpireHours <= 0 {
		cfg.ExpireHours = sysmodel.DefaultAuthConfig.ExpireHours
	}
	return &authService{userRepo: userRepo, roleRepo: roleRepo, menuRepo: menuRepo, cfg: cfg}
}

// resolveSecret 获取有效JWT密钥，优先使用配置，其次环境变量 JWT_SECRET
func (s *authService) resolveSecret() (string, error) {
	return jwtutil.ResolveSecret(s.cfg.Secret)
}

func (s *authService) Login(ctx context.Context, req *sysmodel.LoginRequest) (*sysmodel.LoginResponse, error) {
	if req.Username == "" || req.Password == "" {
		return nil, errors.New("用户名和密码不能为空")
	}

	// 检查账号是否被锁定
	if GlobalLoginTracker.IsLocked(req.Username) {
		return nil, errors.New("账号已被锁定，请15分钟后重试")
	}

	user, err := s.userRepo.GetByUsername(ctx, req.Username)
	if err != nil {
		return nil, fmt.Errorf("查询用户失败: %w", err)
	}
	if user == nil {
		GlobalLoginTracker.RecordFailedAttempt(req.Username)
		return nil, errors.New("用户名或密码错误")
	}
	if user.Status == 0 {
		return nil, errors.New("账号已禁用")
	}

	if err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		GlobalLoginTracker.RecordFailedAttempt(req.Username)
		return nil, errors.New("用户名或密码错误")
	}

	secret, err := s.resolveSecret()
	if err != nil {
		return nil, err
	}

	expireSeconds := int64(s.cfg.ExpireHours) * 3600
	claims := jwtutil.Claims{
		UserID:   user.ID,
		Username: user.Username,
		IssuedAt: time.Now().Unix(),
		ExpireAt: time.Now().Unix() + expireSeconds,
	}
	token, err := jwtutil.Sign(claims, secret)
	if err != nil {
		return nil, fmt.Errorf("签发token失败: %w", err)
	}

	GlobalLoginTracker.RecordSuccessAttempt(req.Username)

	roles, err := s.roleRepo.GetRolesByUserID(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("获取角色失败: %w", err)
	}

	perms, err := getUserPerms(ctx, user.ID, roles, s.menuRepo, s.roleRepo)
	if err != nil {
		return nil, fmt.Errorf("获取权限失败: %w", err)
	}

	user.Password = "" // 不返回密码
	return &sysmodel.LoginResponse{
		Token:     token,
		ExpiresIn: expireSeconds,
		User:      user,
		Roles:     roles,
		Perms:     perms,
	}, nil
}

func (s *authService) Logout(_ context.Context, _ string) error {
	// 无状态 JWT，登出时客户端删除 token 即可
	// 若需要黑名单机制，可在此处将 token 加入 Redis 黑名单（预留扩展点）
	return nil
}

func (s *authService) ValidateToken(ctx context.Context, token string) (*sysmodel.SystemUser, error) {
	secret, err := s.resolveSecret()
	if err != nil {
		return nil, err
	}
	claims, err := jwtutil.Parse(token, secret)
	if err != nil {
		return nil, err
	}
	user, err := s.userRepo.GetByID(ctx, claims.UserID)
	if err != nil {
		return nil, err
	}
	if user == nil || user.Status == 0 {
		return nil, errors.New("用户不存在或已禁用")
	}
	user.Password = ""
	return user, nil
}

func (s *authService) RefreshToken(ctx context.Context, token string) (*sysmodel.LoginResponse, error) {
	secret, err := s.resolveSecret()
	if err != nil {
		return nil, err
	}
	claims, err := jwtutil.Parse(token, secret)
	if err != nil {
		// token 过期时仍允许刷新（5分钟宽限期）
		if !errors.Is(err, jwtutil.ErrTokenExpired) {
			return nil, err
		}
	}
	user, err := s.userRepo.GetByID(ctx, claims.UserID)
	if err != nil {
		return nil, err
	}
	if user == nil || user.Status == 0 {
		return nil, errors.New("用户不存在或已禁用")
	}
	// 重新签发 token
	expireSeconds := int64(s.cfg.ExpireHours) * 3600
	newClaims := jwtutil.Claims{
		UserID:   user.ID,
		Username: user.Username,
		IssuedAt: time.Now().Unix(),
		ExpireAt: time.Now().Unix() + expireSeconds,
	}
	newToken, err := jwtutil.Sign(newClaims, secret)
	if err != nil {
		return nil, err
	}
	roles, _ := s.roleRepo.GetRolesByUserID(ctx, user.ID)
	perms, _ := getUserPerms(ctx, user.ID, roles, s.menuRepo, s.roleRepo)
	user.Password = ""
	return &sysmodel.LoginResponse{
		Token:     newToken,
		ExpiresIn: expireSeconds,
		User:      user,
		Roles:     roles,
		Perms:     perms,
	}, nil
}

func (s *authService) ChangePassword(ctx context.Context, userID int64, oldPassword, newPassword string) error {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return err
	}
	if user == nil {
		return errors.New("用户不存在")
	}
	if err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(oldPassword)); err != nil {
		return errors.New("原密码错误")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	return s.userRepo.UpdatePassword(ctx, userID, string(hash))
}

// ─── 辅助：获取用户权限标识 ─────────────────────────────────────────────────

func getUserPerms(ctx context.Context, userID int64, roles []*sysmodel.SystemRole, menuRepo sysrepo.MenuRepo, roleRepo sysrepo.RoleRepo) ([]string, error) {
	// 超级管理员拥有所有权限
	for _, r := range roles {
		if r.RoleCode == "super_admin" {
			return []string{"*:*:*"}, nil
		}
	}

	menus, err := menuRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	seen := make(map[string]bool)
	var perms []string
	for _, m := range menus {
		if m.Perms != "" && !seen[m.Perms] {
			seen[m.Perms] = true
			perms = append(perms, m.Perms)
		}
	}
	return perms, nil
}
