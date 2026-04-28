package plugin

import (
	"fmt"
	"sync"
)

// PluginRegistry 管理运行时插件实例
type PluginRegistry struct {
	mu      sync.RWMutex
	plugins map[string]Plugin // key: pluginID
}

// NewPluginRegistry 创建插件注册表
func NewPluginRegistry() *PluginRegistry {
	return &PluginRegistry{
		plugins: make(map[string]Plugin),
	}
}

// Register 注册一个插件实例
func (r *PluginRegistry) Register(pluginID string, p Plugin) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.plugins[pluginID]; exists {
		return fmt.Errorf("插件 %s 已注册", pluginID)
	}
	r.plugins[pluginID] = p
	return nil
}

// Unregister 从注册表中移除插件
func (r *PluginRegistry) Unregister(pluginID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.plugins, pluginID)
}

// Get 获取插件实例
func (r *PluginRegistry) Get(pluginID string) (Plugin, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.plugins[pluginID]
	return p, ok
}

// List 列出所有已注册的插件ID
func (r *PluginRegistry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	ids := make([]string, 0, len(r.plugins))
	for id := range r.plugins {
		ids = append(ids, id)
	}
	return ids
}
