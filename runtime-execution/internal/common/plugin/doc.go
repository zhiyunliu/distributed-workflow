// Package plugin 提供Golang插件化框架的核心SDK。
//
// 使用方式：
// 第三方开发者通过实现 Plugin 接口及其子接口（如 NodeExecutorPlugin）来开发自定义插件。
// 可嵌入 BasePlugin 以获得默认实现，只需覆盖业务相关方法。
//
// 插件类型：
//   - NodeExecutorPlugin：自定义流程节点执行器
//   - ConnectorPlugin：第三方系统连接器
//   - NotificationPlugin：消息通知渠道
//
// 示例：
//
//	type MyPlugin struct {
//	    plugin.BasePlugin
//	}
//
//	func (p *MyPlugin) Execute(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
//	    // 自定义执行逻辑
//	    return map[string]interface{}{"result": "ok"}, nil
//	}
package plugin
