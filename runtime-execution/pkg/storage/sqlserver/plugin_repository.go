package sqlserver

import (
	"context"
	"database/sql"
	"fmt"

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
	var total int64
	const countQ = `SELECT COUNT(*) FROM workflow_plugin_info`
	if err := r.db.QueryRowContext(ctx, countQ).Scan(&total); err != nil {
		return nil, 0, err
	}
	if total == 0 {
		return []*types.PluginInfo{}, 0, nil
	}

	offset := (page - 1) * pageSize
	query := fmt.Sprintf(`SELECT plugin_id, plugin_name, description, author, version, plugin_type, config_schema, status, is_official, download_count, rating, created_at, updated_at
	FROM workflow_plugin_info
	ORDER BY created_at DESC
	OFFSET %d ROWS
	FETCH NEXT %d ROWS ONLY`, offset, pageSize)
	
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var list []*types.PluginInfo
	for rows.Next() {
		p := &types.PluginInfo{}
		if err := rows.Scan(
			&p.PluginID,
			&p.PluginName,
			&p.Description,
			&p.Author,
			&p.Version,
			&p.PluginType,
			&p.ConfigSchema,
			&p.Status,
			&p.IsOfficial,
			&p.DownloadCount,
			&p.Rating,
			&p.CreatedAt,
			&p.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		list = append(list, p)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

// GetPluginInfo 按插件ID查询插件信息，无结果返回 nil, nil
func (r *PluginRepository) GetPluginInfo(ctx context.Context, pluginID string) (*types.PluginInfo, error) {
	query := `SELECT plugin_id, plugin_name, description, author, version, plugin_type, config_schema, status, is_official, download_count, rating, created_at, updated_at
	FROM workflow_plugin_info
	WHERE plugin_id = @p1`
	
	p := &types.PluginInfo{}
	err := r.db.QueryRowContext(ctx, query, pluginID).Scan(
		&p.PluginID,
		&p.PluginName,
		&p.Description,
		&p.Author,
		&p.Version,
		&p.PluginType,
		&p.ConfigSchema,
		&p.Status,
		&p.IsOfficial,
		&p.DownloadCount,
		&p.Rating,
		&p.CreatedAt,
		&p.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return p, nil
}

// UpsertPluginInfo 插入或更新插件信息（以 plugin_id 为匹配键）
func (r *PluginRepository) UpsertPluginInfo(ctx context.Context, info *types.PluginInfo) error {
	query := `MERGE INTO workflow_plugin_info AS target
	USING (SELECT @p1 AS plugin_id, @p2 AS plugin_name, @p3 AS description, @p4 AS author, @p5 AS version, 
				  @p6 AS plugin_type, @p7 AS config_schema, @p8 AS status, @p9 AS is_official,
				  @p10 AS download_count, @p11 AS rating, @p12 AS created_at, @p13 AS updated_at) AS source
	ON target.plugin_id = source.plugin_id
	WHEN MATCHED THEN
		UPDATE SET 
			plugin_name = source.plugin_name,
			description = source.description,
			author = source.author,
			version = source.version,
			plugin_type = source.plugin_type,
			config_schema = source.config_schema,
			status = source.status,
			is_official = source.is_official,
			download_count = source.download_count,
			rating = source.rating,
			updated_at = source.updated_at
	WHEN NOT MATCHED THEN
		INSERT (plugin_id, plugin_name, description, author, version, plugin_type, config_schema, status, is_official, download_count, rating, created_at, updated_at)
		VALUES (source.plugin_id, source.plugin_name, source.description, source.author, source.version, source.plugin_type, source.config_schema, source.status, source.is_official, source.download_count, source.rating, source.created_at, source.updated_at);`

	_, err := r.db.ExecContext(ctx, query,
		info.PluginID,
		info.PluginName,
		info.Description,
		info.Author,
		info.Version,
		info.PluginType,
		info.ConfigSchema,
		info.Status,
		info.IsOfficial,
		info.DownloadCount,
		info.Rating,
		info.CreatedAt,
		info.UpdatedAt,
	)
	return err
}

// UpdatePluginStatus 更新插件状态
func (r *PluginRepository) UpdatePluginStatus(ctx context.Context, pluginID string, status int) error {
	const q = `UPDATE workflow_plugin_info SET status = @p2, updated_at = GETDATE() WHERE plugin_id = @p1`
	_, err := r.db.ExecContext(ctx, q, pluginID, status)
	return err
}

// GetPluginConfig 查询插件配置，无结果返回 nil, nil
func (r *PluginRepository) GetPluginConfig(ctx context.Context, pluginID string) (*types.PluginConfig, error) {
	const q = `
SELECT plugin_id, plugin_version, config, status, installed_at, updated_at
FROM workflow_plugin_config WHERE plugin_id = @p1`

	row := r.db.QueryRowContext(ctx, q, pluginID)
	cfg := &types.PluginConfig{}
	var config sql.NullString
	err := row.Scan(&cfg.PluginID, &cfg.PluginVersion, &config, &cfg.Status, &cfg.InstalledAt, &cfg.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	cfg.Config = config.String
	return cfg, nil
}

// UpsertPluginConfig 插入或更新插件配置（以 plugin_id 为匹配键）
func (r *PluginRepository) UpsertPluginConfig(ctx context.Context, cfg *types.PluginConfig) error {
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
		cfg.PluginID,
		cfg.PluginVersion,
		cfg.Config,
		cfg.Status,
	)
	return err
}

// DeletePluginConfig 删除插件配置
func (r *PluginRepository) DeletePluginConfig(ctx context.Context, pluginID string) error {
	const q = `DELETE FROM workflow_plugin_config WHERE plugin_id = @p1`
	_, err := r.db.ExecContext(ctx, q, pluginID)
	return err
}

// ListInstalledPlugins 查询已安装（状态 1）或已启用（状态 2）的插件配置列表
func (r *PluginRepository) ListInstalledPlugins(ctx context.Context) ([]*types.PluginConfig, error) {
	const q = `
SELECT plugin_id, plugin_version, config, status, installed_at, updated_at
FROM workflow_plugin_config
WHERE status IN (1, 2)
ORDER BY installed_at DESC`

	rows, err := r.db.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*types.PluginConfig
	for rows.Next() {
		cfg := &types.PluginConfig{}
		var config sql.NullString
		if err := rows.Scan(&cfg.PluginID, &cfg.PluginVersion, &config, &cfg.Status, &cfg.InstalledAt, &cfg.UpdatedAt); err != nil {
			return nil, err
		}
		cfg.Config = config.String
		list = append(list, cfg)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return list, nil
}


