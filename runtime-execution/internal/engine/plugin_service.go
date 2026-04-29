package engine

import (
	"context"
	"fmt"
	"time"

	api "github.com/zhiyunliu/distributed-workflow/runtime-execution/pkg/api"
	"github.com/zhiyunliu/distributed-workflow/runtime-execution/internal/common/plugin"
	"github.com/zhiyunliu/distributed-workflow/runtime-execution/pkg/types"
)

// 编译时接口断言
var _ api.PluginService = (*pluginService)(nil)

type pluginService struct {
	repo     api.PluginRepository
	registry *plugin.PluginRegistry
}

// NewPluginService 创建插件管理服务
func NewPluginService(repo api.PluginRepository, registry *plugin.PluginRegistry) api.PluginService {
	return &pluginService{repo: repo, registry: registry}
}

// ListPluginMarket 查询插件市场列表
func (s *pluginService) ListPluginMarket(ctx context.Context, page, pageSize int) ([]*types.PluginInfo, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	return s.repo.ListPluginInfos(ctx, page, pageSize)
}

// GetPlugin 获取插件详情
func (s *pluginService) GetPlugin(ctx context.Context, pluginID string) (*types.PluginInfo, error) {
	info, err := s.repo.GetPluginInfo(ctx, pluginID)
	if err != nil {
		return nil, err
	}
	if info == nil {
		return nil, fmt.Errorf("插件不存在: %s", pluginID)
	}
	return info, nil
}

// InstallPlugin 安装插件
func (s *pluginService) InstallPlugin(ctx context.Context, req types.InstallPluginRequest) error {
	if req.PluginID == "" {
		return fmt.Errorf("插件ID不能为空")
	}
	now := time.Now()
	info := &types.PluginInfo{
		PluginID:  req.PluginID,
		Status:    types.PluginStatusInstalled,
		UpdatedAt: now,
	}
	if err := s.repo.UpsertPluginInfo(ctx, info); err != nil {
		return err
	}
	cfg := &types.PluginConfig{
		PluginID:    req.PluginID,
		Config:      req.Config,
		Status:      types.PluginStatusInstalled,
		InstalledAt: now,
		UpdatedAt:   now,
	}
	return s.repo.UpsertPluginConfig(ctx, cfg)
}

// ListInstalledPlugins 查询已安装插件列表
func (s *pluginService) ListInstalledPlugins(ctx context.Context) ([]*types.PluginConfig, error) {
	return s.repo.ListInstalledPlugins(ctx)
}

// EnablePlugin 启用插件
func (s *pluginService) EnablePlugin(ctx context.Context, pluginID string) error {
	return s.repo.UpdatePluginStatus(ctx, pluginID, types.PluginStatusEnabled)
}

// DisablePlugin 停用插件
func (s *pluginService) DisablePlugin(ctx context.Context, pluginID string) error {
	return s.repo.UpdatePluginStatus(ctx, pluginID, types.PluginStatusDisabled)
}

// UninstallPlugin 卸载插件
func (s *pluginService) UninstallPlugin(ctx context.Context, pluginID string) error {
	if err := s.repo.UpdatePluginStatus(ctx, pluginID, types.PluginStatusUninstalled); err != nil {
		return err
	}
	if err := s.repo.DeletePluginConfig(ctx, pluginID); err != nil {
		return err
	}
	s.registry.Unregister(pluginID)
	return nil
}

// UpdatePluginConfig 更新插件配置
func (s *pluginService) UpdatePluginConfig(ctx context.Context, pluginID, config string) error {
	if pluginID == "" {
		return fmt.Errorf("插件ID不能为空")
	}
	cfg := &types.PluginConfig{
		PluginID:  pluginID,
		Config:    config,
		UpdatedAt: time.Now(),
	}
	return s.repo.UpsertPluginConfig(ctx, cfg)
}


