// Package worker 实现分布式工作流执行节点（NodeWorker）逻辑。
//
// 主要职责：
//   - 节点执行器管理（NodeExecutorContainer）
//   - 任务执行并发控制（TaskExecutionManager）
//   - 对外暴露 gRPC 服务端（NodeWorkerService）
//   - 心跳上报与 Engine 注册（NodeWorker）
//   - 内置执行器：log、http、sleep
package worker

