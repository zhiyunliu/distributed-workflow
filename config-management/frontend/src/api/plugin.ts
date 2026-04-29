import request from '@/utils/request'
import type { ApiResponse } from '@/types'
import type { PluginInfo, PluginConfig } from '@/types/plugin'

export const pluginApi = {
  // 插件市场列表
  listMarket: (params: { page: number; pageSize: number }) =>
    request.get<ApiResponse<{ list: PluginInfo[]; total: number }>>('/api/plugins/market', { params }),

  // 已安装插件列表
  listInstalled: () =>
    request.get<ApiResponse<PluginConfig[]>>('/api/plugins/installed'),

  // 插件详情
  get: (pluginId: string) =>
    request.get<ApiResponse<PluginInfo>>(`/api/plugins/${pluginId}`),

  // 安装插件
  install: (pluginId: string, data: { version?: string; configJson?: string }) =>
    request.post(`/api/plugins/${pluginId}/install`, data),

  // 卸载插件
  uninstall: (pluginId: string) =>
    request.post(`/api/plugins/${pluginId}/uninstall`),

  // 启用插件
  enable: (pluginId: string) =>
    request.post(`/api/plugins/${pluginId}/enable`),

  // 禁用插件
  disable: (pluginId: string) =>
    request.post(`/api/plugins/${pluginId}/disable`),

  // 获取插件配置
  getConfig: (pluginId: string) =>
    request.get<ApiResponse<PluginConfig>>(`/api/plugins/${pluginId}/config`),

  // 保存插件配置
  saveConfig: (pluginId: string, data: Partial<PluginConfig>) =>
    request.put(`/api/plugins/${pluginId}/config`, data),
}
