// workflow_d2.go - D2 新增 proto 消息类型定义
// 由于无法运行 protoc，此处使用类型别名复用结构兼容的现有 proto 消息类型。
// 各别名的字段映射关系：
//
//   PauseTaskRequest     ≡ CancelTaskRequest   (field1=workflowInstanceId, field2=workflowNodeId)
//   PauseTaskResponse    ≡ CancelTaskResponse   (field1=code, field2=message, field3=success)
//   HealthCheckRequest   ≡ UnregisterWorkerRequest (field1=workerId)
//   HealthCheckResponse  ≡ HeartbeatResponse    (field1=code, field2=message, field3=success/healthy)
//
// wire 编码完全兼容，可直接用于 gRPC 传输。

package proto

// PauseTaskRequest 暂停任务请求（D2 新增）
// 字段与 CancelTaskRequest 完全兼容：
//   WorkflowInstanceId → WorkflowInstanceId (field 1)
//   WorkflowNodeId     → WorkflowNodeId     (field 2)
type PauseTaskRequest = CancelTaskRequest

// PauseTaskResponse 暂停任务响应（D2 新增）
type PauseTaskResponse = CancelTaskResponse

// HealthCheckRequest Worker 健康检查请求（D2 新增）
// 字段与 UnregisterWorkerRequest 兼容：
//   WorkerId → WorkerId (field 1)
type HealthCheckRequest = UnregisterWorkerRequest

// HealthCheckResponse Worker 健康检查响应（D2 新增）
// 字段与 HeartbeatResponse 兼容：
//   Code    → Code    (field 1)
//   Message → Message (field 2)
//   Success → Healthy (field 3, bool) — Success=true 代表 healthy
type HealthCheckResponse = HeartbeatResponse
