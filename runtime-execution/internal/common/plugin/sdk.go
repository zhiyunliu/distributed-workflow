package plugin

import (
	"context"

	"github.com/zhiyunliu/distributed-workflow/runtime-execution/internal/common/types"
)

// Plugin 插件基础接口
type Plugin interface {
	// ID 返回插件唯一标识
	ID() string
	// Name 返回插件名称
	Name() string
	// Version 返回插件版本
	Version() string
	// Type 返回插件类型
	Type() types.PluginType
	// Init 初始化插件（传入JSON配置字符串）
	Init(config string) error
	// Destroy 销毁插件，释放资源
	Destroy() error
}

// NodeExecutorPlugin 自定义节点执行器插件
type NodeExecutorPlugin interface {
	Plugin
	// Execute 执行自定义节点逻辑
	Execute(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error)
}

// ConnectorPlugin 第三方系统连接器插件
type ConnectorPlugin interface {
	Plugin
	// Connect 建立连接
	Connect(config map[string]interface{}) error
	// Call 调用第三方系统接口
	Call(method string, params map[string]interface{}) (interface{}, error)
}

// NotificationPlugin 消息通知渠道插件
type NotificationPlugin interface {
	Plugin
	// Send 发送消息通知
	Send(ctx context.Context, recipient string, subject, content string) error
}

// BasePlugin 插件基础实现（可嵌入到自定义插件中）
type BasePlugin struct {
	id      string
	name    string
	version string
	ptype   types.PluginType
}

// NewBasePlugin 创建 BasePlugin 实例
func NewBasePlugin(id, name, version string, ptype types.PluginType) *BasePlugin {
	return &BasePlugin{id: id, name: name, version: version, ptype: ptype}
}

func (b *BasePlugin) ID() string             { return b.id }
func (b *BasePlugin) Name() string           { return b.name }
func (b *BasePlugin) Version() string        { return b.version }
func (b *BasePlugin) Type() types.PluginType { return b.ptype }
func (b *BasePlugin) Init(_ string) error    { return nil }
func (b *BasePlugin) Destroy() error         { return nil }
