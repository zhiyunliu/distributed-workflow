package types

import "time"

// PluginType 插件类型
type PluginType string

const (
	PluginTypeNodeExecutor PluginType = "node_executor" // 自定义节点执行器
	PluginTypeConnector    PluginType = "connector"     // 第三方系统连接器
	PluginTypeNotification PluginType = "notification"  // 消息通知渠道
	PluginTypeReport       PluginType = "report"        // 报表组件
)

// PluginStatus 插件状态
const (
	PluginStatusUninstalled = 0 // 未安装
	PluginStatusInstalled   = 1 // 已安装
	PluginStatusEnabled     = 2 // 已启用
	PluginStatusDisabled    = 3 // 已停用
)

// PluginInfo 插件信息（插件市场展示）
type PluginInfo struct {
	PluginID      string     `json:"pluginId"`
	PluginName    string     `json:"pluginName"`
	Description   string     `json:"description"`
	Author        string     `json:"author"`
	Version       string     `json:"version"`
	PluginType    PluginType `json:"pluginType"`
	ConfigSchema  string     `json:"configSchema,omitempty"` // 配置Schema JSON
	Status        int        `json:"status"`
	IsOfficial    bool       `json:"isOfficial"`
	DownloadCount int        `json:"downloadCount"`
	Rating        float64    `json:"rating"`
	CreatedAt     time.Time  `json:"createdAt"`
	UpdatedAt     time.Time  `json:"updatedAt"`
}

// PluginConfig 已安装插件配置
type PluginConfig struct {
	PluginID      string    `json:"pluginId"`
	PluginVersion string    `json:"pluginVersion"`
	Config        string    `json:"config,omitempty"` // JSON配置字符串
	Status        int       `json:"status"`
	InstalledAt   time.Time `json:"installedAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

// InstallPluginRequest 安装插件请求
type InstallPluginRequest struct {
	PluginID string `json:"pluginId"`
	Config   string `json:"config,omitempty"`
}
