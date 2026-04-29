// 插件管理相关 TypeScript 类型定义

export const PluginType = {
  NodeExecutor: 'node_executor',
  Connector: 'connector',
  Notification: 'notification',
  Report: 'report',
} as const

export type PluginTypeValue = typeof PluginType[keyof typeof PluginType]

export interface PluginInfo {
  pluginId: string
  pluginName: string
  description?: string
  author?: string
  version: string
  pluginType: PluginTypeValue
  configSchema?: string
  status: number
  isOfficial: boolean
  downloadCount: number
  rating: number
  createdAt: string
  updatedAt: string
}

export interface PluginConfig {
  pluginId: string
  pluginVersion: string
  config: string
  status: number
  installedAt: string
  updatedAt: string
}
