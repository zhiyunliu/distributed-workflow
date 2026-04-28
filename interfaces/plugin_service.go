package interfaces

import (
	"context"

	"github.com/zhiyunliu/distributed-workflow/types"
)

// PluginService 插件管理服务接口
type PluginService interface {
	// ListPluginMarket 查询插件市场列表
	ListPluginMarket(ctx context.Context, page, pageSize int) ([]*types.PluginInfo, int64, error)
	// GetPlugin 获取插件详情
	GetPlugin(ctx context.Context, pluginID string) (*types.PluginInfo, error)
	// InstallPlugin 安装插件
	InstallPlugin(ctx context.Context, req types.InstallPluginRequest) error
	// ListInstalledPlugins 查询已安装插件列表
	ListInstalledPlugins(ctx context.Context) ([]*types.PluginConfig, error)
	// EnablePlugin 启用插件
	EnablePlugin(ctx context.Context, pluginID string) error
	// DisablePlugin 停用插件
	DisablePlugin(ctx context.Context, pluginID string) error
	// UninstallPlugin 卸载插件
	UninstallPlugin(ctx context.Context, pluginID string) error
	// UpdatePluginConfig 更新插件配置
	UpdatePluginConfig(ctx context.Context, pluginID, config string) error
}

// PluginRepository 插件数据仓库接口
type PluginRepository interface {
	ListPluginInfos(ctx context.Context, page, pageSize int) ([]*types.PluginInfo, int64, error)
	GetPluginInfo(ctx context.Context, pluginID string) (*types.PluginInfo, error)
	UpsertPluginInfo(ctx context.Context, info *types.PluginInfo) error
	UpdatePluginStatus(ctx context.Context, pluginID string, status int) error
	GetPluginConfig(ctx context.Context, pluginID string) (*types.PluginConfig, error)
	UpsertPluginConfig(ctx context.Context, cfg *types.PluginConfig) error
	DeletePluginConfig(ctx context.Context, pluginID string) error
	ListInstalledPlugins(ctx context.Context) ([]*types.PluginConfig, error)
}
