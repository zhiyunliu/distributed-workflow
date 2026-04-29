// Package engine 实现分布式工作流引擎核心调度逻辑。
//
// 主要职责：
//   - 工作流定义 CRUD（WorkflowService）
//   - DAG 解析与节点调度（SchedulerService）
//   - Worker 管理与任务分配（WorkerManagerService）
//   - 实例状态持久化（InstanceStateService）
//   - 对外暴露 gRPC 服务端（EngineService）
package engine

