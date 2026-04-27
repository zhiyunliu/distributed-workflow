// Package http 实现工作流的 HTTP 触发端点。
//
// 提供 REST API：
//
//	POST /api/v1/workflows                       创建工作流定义
//	GET  /api/v1/workflows/:workflowId           获取工作流定义
//	PUT  /api/v1/workflows/:workflowId           更新工作流定义
//	DELETE /api/v1/workflows/:workflowId         删除工作流定义
//	POST /api/v1/workflows/:workflowId/start     启动工作流实例
//	GET  /api/v1/instances/:instanceId           获取实例详情
package http
