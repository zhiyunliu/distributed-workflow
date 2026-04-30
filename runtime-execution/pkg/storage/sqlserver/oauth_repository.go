package sqlserver

import (
	"context"
	"database/sql"

	"github.com/zhiyunliu/distributed-workflow/runtime-execution/pkg/types"
)

type OauthRepository struct {
	db *sql.DB
}

func NewOauthRepository(db *sql.DB) *OauthRepository {
	return &OauthRepository{db: db}
}

func (r *OauthRepository) BindUser(ctx context.Context, binding *types.OAuthBinding) error {
	query := `INSERT INTO workflow_oauth_bindings 
	(sys_user_id, platform, open_id, union_id, nickname, avatar_url, token_expire_at, created_at, updated_at) 
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`
	_, err := r.db.ExecContext(ctx, query,
		binding.SysUserID, binding.Platform, binding.OpenID, binding.UnionID,
		binding.Nickname, binding.AvatarURL, binding.TokenExpireAt,
		binding.CreatedAt, binding.UpdatedAt)
	return err
}

func (r *OauthRepository) GetBindingByOpenID(ctx context.Context, platform, openID string) (*types.OAuthBinding, error) {
	var binding types.OAuthBinding
	query := `SELECT id, sys_user_id, platform, open_id, union_id, nickname, avatar_url, token_expire_at, created_at, updated_at
	FROM workflow_oauth_bindings WHERE platform = ? AND open_id = ?`
	err := r.db.QueryRowContext(ctx, query, platform, openID).Scan(
		&binding.ID, &binding.SysUserID, &binding.Platform, &binding.OpenID,
		&binding.UnionID, &binding.Nickname, &binding.AvatarURL, &binding.TokenExpireAt,
		&binding.CreatedAt, &binding.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &binding, nil
}

func (r *OauthRepository) GetBindingBySysUserID(ctx context.Context, sysUserID, platform string) (*types.OAuthBinding, error) {
	var binding types.OAuthBinding
	query := `SELECT id, sys_user_id, platform, open_id, union_id, nickname, avatar_url, token_expire_at, created_at, updated_at
	FROM workflow_oauth_bindings WHERE sys_user_id = ? AND platform = ?`
	err := r.db.QueryRowContext(ctx, query, sysUserID, platform).Scan(
		&binding.ID, &binding.SysUserID, &binding.Platform, &binding.OpenID,
		&binding.UnionID, &binding.Nickname, &binding.AvatarURL, &binding.TokenExpireAt,
		&binding.CreatedAt, &binding.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &binding, nil
}

func (r *OauthRepository) UpdateBinding(ctx context.Context, binding *types.OAuthBinding) error {
	query := `UPDATE workflow_oauth_bindings SET 
	sys_user_id=?, platform=?, open_id=?, union_id=?, nickname=?, 
	avatar_url=?, token_expire_at=?, updated_at=? WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query,
		binding.SysUserID, binding.Platform, binding.OpenID, binding.UnionID,
		binding.Nickname, binding.AvatarURL, binding.TokenExpireAt,
		binding.UpdatedAt, binding.ID)
	return err
}

func (r *OauthRepository) UnbindUser(ctx context.Context, sysUserID, platform string) error {
	query := `DELETE FROM workflow_oauth_bindings WHERE sys_user_id = ? AND platform = ?`
	_, err := r.db.ExecContext(ctx, query, sysUserID, platform)
	return err
}

func (r *OauthRepository) ListBindings(ctx context.Context, sysUserID string) ([]*types.OAuthBinding, error) {
	query := `SELECT id, sys_user_id, platform, open_id, union_id, nickname, avatar_url, token_expire_at, created_at, updated_at
	FROM workflow_oauth_bindings WHERE sys_user_id = ?`
	rows, err := r.db.QueryContext(ctx, query, sysUserID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var bindings []*types.OAuthBinding
	for rows.Next() {
		var binding types.OAuthBinding
		err := rows.Scan(
			&binding.ID, &binding.SysUserID, &binding.Platform, &binding.OpenID,
			&binding.UnionID, &binding.Nickname, &binding.AvatarURL, &binding.TokenExpireAt,
			&binding.CreatedAt, &binding.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		bindings = append(bindings, &binding)
	}
	return bindings, nil
}