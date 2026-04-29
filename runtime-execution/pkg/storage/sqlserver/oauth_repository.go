package sqlserver

import (
	"context"
	"database/sql"

	api "github.com/zhiyunliu/distributed-workflow/runtime-execution/pkg/api"
	"github.com/zhiyunliu/distributed-workflow/runtime-execution/pkg/types"
)

// 编译时接口断言
var _ api.OAuthRepository = (*OAuthRepository)(nil)

// OAuthRepository SQL Server OAuth绑定数据仓库实现
type OAuthRepository struct {
	db *sql.DB
}

// NewOAuthRepository 创建OAuth绑定数据仓库实例，复用已有 *sql.DB 连接池
func NewOAuthRepository(db *sql.DB) *OAuthRepository {
	return &OAuthRepository{db: db}
}

// toNullStr 将空字符串转为无效的 sql.NullString，非空转为有效
func toNullStr(s string) sql.NullString {
	return sql.NullString{String: s, Valid: s != ""}
}

// CreateBinding 创建OAuth平台绑定关系
// access_token/refresh_token 由上层调用方加密后通过独立渠道持久化，此处不存储
func (r *OAuthRepository) CreateBinding(ctx context.Context, binding *types.OAuthBinding) error {
	ctx, cancel := context.WithTimeout(ctx, defaultQueryTimeout)
	defer cancel()

	const q = `
INSERT INTO oauth_bindings
    (sys_user_id, platform, open_id, union_id, nickname, avatar_url, token_expire_at, created_at, updated_at)
VALUES
    (@p1, @p2, @p3, @p4, @p5, @p6, @p7, GETDATE(), GETDATE())`

	var tokenExpireAt sql.NullTime
	if binding.TokenExpireAt != nil {
		tokenExpireAt = sql.NullTime{Time: *binding.TokenExpireAt, Valid: true}
	}

	_, err := r.db.ExecContext(ctx, q,
		sql.Named("p1", binding.SysUserID),
		sql.Named("p2", binding.Platform),
		sql.Named("p3", binding.OpenID),
		sql.Named("p4", toNullStr(binding.UnionID)),
		sql.Named("p5", toNullStr(binding.Nickname)),
		sql.Named("p6", toNullStr(binding.AvatarURL)),
		sql.Named("p7", tokenExpireAt),
	)
	return wrapDBErr(err, "CreateBinding")
}

// GetBindingByOpenID 按平台+OpenID查询绑定关系，无结果返回 nil, nil
// 出于安全考虑不返回 access_token/refresh_token
func (r *OAuthRepository) GetBindingByOpenID(ctx context.Context, platform, openID string) (*types.OAuthBinding, error) {
	ctx, cancel := context.WithTimeout(ctx, defaultQueryTimeout)
	defer cancel()

	const q = `
SELECT id, sys_user_id, platform, open_id, union_id, nickname, avatar_url, token_expire_at, created_at, updated_at
FROM oauth_bindings
WHERE platform = @p1 AND open_id = @p2`

	row := r.db.QueryRowContext(ctx, q,
		sql.Named("p1", platform),
		sql.Named("p2", openID),
	)

	b := &types.OAuthBinding{}
	var unionID, nickname, avatarURL sql.NullString
	var tokenExpireAt sql.NullTime
	err := row.Scan(
		&b.ID, &b.SysUserID, &b.Platform, &b.OpenID,
		&unionID, &nickname, &avatarURL,
		&tokenExpireAt, &b.CreatedAt, &b.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, wrapDBErr(err, "GetBindingByOpenID")
	}
	b.UnionID = unionID.String
	b.Nickname = nickname.String
	b.AvatarURL = avatarURL.String
	if tokenExpireAt.Valid {
		b.TokenExpireAt = &tokenExpireAt.Time
	}
	return b, nil
}

// GetBindingBySysUser 查询系统用户的所有平台绑定，不返回 access_token/refresh_token
func (r *OAuthRepository) GetBindingBySysUser(ctx context.Context, sysUserID string) ([]*types.OAuthBinding, error) {
	ctx, cancel := context.WithTimeout(ctx, defaultQueryTimeout)
	defer cancel()

	const q = `
SELECT id, sys_user_id, platform, open_id, union_id, nickname, avatar_url, token_expire_at, created_at, updated_at
FROM oauth_bindings
WHERE sys_user_id = @p1`

	rows, err := r.db.QueryContext(ctx, q, sql.Named("p1", sysUserID))
	if err != nil {
		return nil, wrapDBErr(err, "GetBindingBySysUser query")
	}
	defer rows.Close()

	var list []*types.OAuthBinding
	for rows.Next() {
		b := &types.OAuthBinding{}
		var unionID, nickname, avatarURL sql.NullString
		var tokenExpireAt sql.NullTime
		if err := rows.Scan(
			&b.ID, &b.SysUserID, &b.Platform, &b.OpenID,
			&unionID, &nickname, &avatarURL,
			&tokenExpireAt, &b.CreatedAt, &b.UpdatedAt,
		); err != nil {
			return nil, wrapDBErr(err, "GetBindingBySysUser scan")
		}
		b.UnionID = unionID.String
		b.Nickname = nickname.String
		b.AvatarURL = avatarURL.String
		if tokenExpireAt.Valid {
			b.TokenExpireAt = &tokenExpireAt.Time
		}
		list = append(list, b)
	}
	if err := rows.Err(); err != nil {
		return nil, wrapDBErr(err, "GetBindingBySysUser rows")
	}
	return list, nil
}

// UpdateBinding 更新绑定的展示信息和Token过期时间（以 platform+open_id 定位）
// access_token/refresh_token 字段不在 OAuthBinding 结构体中，不参与更新
func (r *OAuthRepository) UpdateBinding(ctx context.Context, binding *types.OAuthBinding) error {
	ctx, cancel := context.WithTimeout(ctx, defaultQueryTimeout)
	defer cancel()

	const q = `
UPDATE oauth_bindings
SET nickname        = @p1,
    avatar_url      = @p2,
    token_expire_at = @p3,
    updated_at      = GETDATE()
WHERE platform = @p4 AND open_id = @p5`

	var tokenExpireAt sql.NullTime
	if binding.TokenExpireAt != nil {
		tokenExpireAt = sql.NullTime{Time: *binding.TokenExpireAt, Valid: true}
	}

	_, err := r.db.ExecContext(ctx, q,
		sql.Named("p1", toNullStr(binding.Nickname)),
		sql.Named("p2", toNullStr(binding.AvatarURL)),
		sql.Named("p3", tokenExpireAt),
		sql.Named("p4", binding.Platform),
		sql.Named("p5", binding.OpenID),
	)
	return wrapDBErr(err, "UpdateBinding")
}


