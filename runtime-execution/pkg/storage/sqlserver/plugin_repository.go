package sqlserver

import (
	"context"
	"database/sql"

	api "github.com/zhiyunliu/distributed-workflow/runtime-execution/pkg/api"
	"github.com/zhiyunliu/distributed-workflow/runtime-execution/pkg/types"
)

// 编译时接口断言
var _ api.PluginRepository = (*PluginRepository)(nil)

// PluginRepository SQL Server 插件数据仓库实现
type PluginRepository struct {
	db *sql.DB
}

// NewPluginRepository 创建插件数据仓库实例，复用已有 *sql.DB 连接池
func NewPluginRepository(db *sql.DB) *PluginRepository {
	return &PluginRepository{db: db}
}

// ListPluginInfos 分页查询插件市场列表，返回列表、总数、error
func (r *PluginRepository) ListPluginInfos(ctx context.Context, page, pageSize int) ([]*types.PluginInfo, int64, error) {
	ctx, cancel := context.WithTimeout(ctx, defaultQueryTimeout)
	defer cancel()

	var total int64
	const countQ = `SELECT COUNT(*) FROM workflow_plugin_info`
	if err := r.db.QueryRowContext(ctx, countQ).Scan(&total); err != nil {
		return nil, 0, wrapDBErr(err, "ListPluginInfos count")
	}
	if total == 0 {
		return []*types.PluginInfo{}, 0, nil
	}

	offset := (page - 1) * pageSize
	const q = `
SELECT plugin_id, plugin_name, description, author, version, plugin_type, plugin_package,
       config_schema, status, is_official, download_count, rating, created_at, updated_at
FROM workflow_plugin_info
ORDER BY is_official DESC, download_count DESC
OFFSET @offset ROWS FETCH NEXT @pageSize ROWS ONLY`

	rows, err := r.db.QueryContext(ctx, q,
		sql.Named("offset", offset),
		sql.Named("pageSize", pageSize),
	)
	if err != nil {
		return nil, 0, wrapDBErr(err, "ListPluginInfos query")
	}
	defer rows.Close()

	var list []*types.PluginInfo
	for rows.Next() {
		p := &types.PluginInfo{}
		var configSchema sql.NullString
		var pluginPackage sql.NullString
		if err := rows.Scan(
			&p.PluginID, &p.PluginName, &p.Description, &p.Author,
			&p.Version, &p.PluginType, &pluginPackage, &configSchema, &p.Status,
			&p.IsOfficial, &p.DownloadCount, &p.Rating,
			&p.CreatedAt, &p.UpdatedAt,
		); err != nil {
			return nil, 0, wrapDBErr(err, "ListPluginInfos scan")
		}
		p.PluginPackage = pluginPackage.String
		p.ConfigSchema = configSchema.String
		list = append(list, p)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, wrapDBErr(err, "ListPluginInfos rows")
	}
	return list, total, nil
}

// GetPluginInfo 按插件ID查询插件信息，无结果返回 nil, nil
func (r *PluginRepository) GetPluginInfo(ctx context.Context, pluginID string) (*types.PluginInfo, error) {
	ctx, cancel := context.WithTimeout(ctx, defaultQueryTimeout)
	defer cancel()

	const q = `
SELECT plugin_id, plugin_name, description, author, version, plugin_type, plugin_package,
       config_schema, status, is_official, download_count, rating, created_at, updated_at
FROM workflow_plugin_info WHERE plugin_id = @p1`

	row := r.db.QueryRowContext(ctx, q, sql.Named("p1", pluginID))
	p := &types.PluginInfo{}
	var configSchema sql.NullString
	var pluginPackage sql.NullString
	err := row.Scan(
		&p.PluginID, &p.PluginName, &p.Description, &p.Author,
		&p.Version, &p.PluginType, &pluginPackage, &configSchema, &p.Status,
		&p.IsOfficial, &p.DownloadCount, &p.Rating,
		&p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, wrapDBErr(err, "GetPluginInfo")
	}
	p.PluginPackage = pluginPackage.String
	p.ConfigSchema = configSchema.String
	return p, nil
}

// UpsertPluginInfo 插入或更新插件信息（以 plugin_id 为匹配键）
func (r *PluginRepository) UpsertPluginInfo(ctx context.Context, info *types.PluginInfo) error {
	ctx, cancel := context.WithTimeout(ctx, defaultQueryTimeout)
	defer cancel()

	const q = `
MERGE INTO workflow_plugin_info AS target
USING (SELECT @p1 AS plugin_id) AS source
ON target.plugin_id = source.plugin_id
WHEN MATCHED THEN UPDATE SET
    plugin_name    = @p2,
    description    = @p3,
    author         = @p4,
    version        = @p5,
    plugin_type    = @p6,
    plugin_package = @p7,
    config_schema  = @p8,
    status         = @p9,
    is_official    = @p10,
    download_count = @p11,
    rating         = @p12,
    updated_at     = GETDATE()
WHEN NOT MATCHED THEN INSERT
    (plugin_id, plugin_name, description, author, version, plugin_type, plugin_package,
     config_schema, status, is_official, download_count, rating, created_at, updated_at)
VALUES
    (@p1, @p2, @p3, @p4, @p5, @p6, @p7, @p8, @p9, @p10, @p11, @p12, GETDATE(), GETDATE());`

	_, err := r.db.ExecContext(ctx, q,
		sql.Named("p1", info.PluginID),
		sql.Named("p2", info.PluginName),
		sql.Named("p3", info.Description),
		sql.Named("p4", info.Author),
		sql.Named("p5", info.Version),
		sql.Named("p6", string(info.PluginType)),
		sql.Named("p7", info.PluginPackage),
		sql.Named("p8", info.ConfigSchema),
		sql.Named("p9", info.Status),
		sql.Named("p10", info.IsOfficial),
		sql.Named("p11", info.DownloadCount),
		sql.Named("p12", info.Rating),
	)
	return wrapDBErr(err, "UpsertPluginInfo")
}

// UpdatePluginStatus 更新插件状态
func (r *PluginRepository) UpdatePluginStatus(ctx context.Context, pluginID string, status int) error {
	ctx, cancel := context.WithTimeout(ctx, defaultQueryTimeout)
	defer cancel()

	const q = `UPDATE workflow_plugin_info SET status = @p2, updated_at = GETDATE() WHERE plugin_id = @p1`
	_, err := r.db.ExecContext(ctx, q,
		sql.Named("p1", pluginID),
		sql.Named("p2", status),
	)
	return wrapDBErr(err, "UpdatePluginStatus")
}

// GetPluginConfig 查询插件配置，无结果返回 nil, nil
func (r *PluginRepository) GetPluginConfig(ctx context.Context, pluginID string) (*types.PluginConfig, error) {
	ctx, cancel := context.WithTimeout(ctx, defaultQueryTimeout)
	defer cancel()

	const q = `
SELECT plugin_id, plugin_version, config, status, installed_at, updated_at
FROM workflow_plugin_config WHERE plugin_id = @p1`

	row := r.db.QueryRowContext(ctx, q, sql.Named("p1", pluginID))
	cfg := &types.PluginConfig{}
	var config sql.NullString
	err := row.Scan(&cfg.PluginID, &cfg.PluginVersion, &config, &cfg.Status, &cfg.InstalledAt, &cfg.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, wrapDBErr(err, "GetPluginConfig")
	}
	cfg.Config = config.String
	return cfg, nil
}

// UpsertPluginConfig 插入或更新插件配置（以 plugin_id 为匹配键）
func (r *PluginRepository) UpsertPluginConfig(ctx context.Context, cfg *types.PluginConfig) error {
	ctx, cancel := context.WithTimeout(ctx, defaultQueryTimeout)
	defer cancel()

	const q = `
MERGE INTO workflow_plugin_config AS target
USING (SELECT @p1 AS plugin_id) AS source
ON target.plugin_id = source.plugin_id
WHEN MATCHED THEN UPDATE SET
    plugin_version = @p2,
    config         = @p3,
    status         = @p4,
    updated_at     = GETDATE()
WHEN NOT MATCHED THEN INSERT
    (plugin_id, plugin_version, config, status, installed_at, updated_at)
VALUES
    (@p1, @p2, @p3, @p4, GETDATE(), GETDATE());`

	_, err := r.db.ExecContext(ctx, q,
		sql.Named("p1", cfg.PluginID),
		sql.Named("p2", cfg.PluginVersion),
		sql.Named("p3", cfg.Config),
		sql.Named("p4", cfg.Status),
	)
	return wrapDBErr(err, "UpsertPluginConfig")
}

// DeletePluginConfig 删除插件配置
func (r *PluginRepository) DeletePluginConfig(ctx context.Context, pluginID string) error {
	ctx, cancel := context.WithTimeout(ctx, defaultQueryTimeout)
	defer cancel()

	const q = `DELETE FROM workflow_plugin_config WHERE plugin_id = @p1`
	_, err := r.db.ExecContext(ctx, q, sql.Named("p1", pluginID))
	return wrapDBErr(err, "DeletePluginConfig")
}

// ListInstalledPlugins 查询已安装（状态 1）或已启用（状态 2）的插件配置列表
func (r *PluginRepository) ListInstalledPlugins(ctx context.Context) ([]*types.PluginConfig, error) {
	ctx, cancel := context.WithTimeout(ctx, defaultQueryTimeout)
	defer cancel()

	const q = `
SELECT plugin_id, plugin_version, config, status, installed_at, updated_at
FROM workflow_plugin_config
WHERE status IN (1, 2)
ORDER BY installed_at DESC`

	rows, err := r.db.QueryContext(ctx, q)
	if err != nil {
		return nil, wrapDBErr(err, "ListInstalledPlugins query")
	}
	defer rows.Close()

	var list []*types.PluginConfig
	for rows.Next() {
		cfg := &types.PluginConfig{}
		var config sql.NullString
		if err := rows.Scan(&cfg.PluginID, &cfg.PluginVersion, &config, &cfg.Status, &cfg.InstalledAt, &cfg.UpdatedAt); err != nil {
			return nil, wrapDBErr(err, "ListInstalledPlugins scan")
		}
		cfg.Config = config.String
		list = append(list, cfg)
	}
	if err := rows.Err(); err != nil {
		return nil, wrapDBErr(err, "ListInstalledPlugins rows")
	}
	return list, nil
}


