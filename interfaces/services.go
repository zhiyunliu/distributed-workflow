package interfaces

import "github.com/zhiyunliu/distributed-workflow/types"

// WorkflowService 工作流定义管理服务接口
// 负责工作流定义的 CRUD、流程实例的创建与启动、对外提供 HTTP 触发接口
type WorkflowService interface {
	// CreateWorkflow 创建工作流定义，返回工作流 ID
	CreateWorkflow(def *types.WorkflowDef) (string, error)
	// GetWorkflow 根据 ID 获取工作流定义
	GetWorkflow(workflowID string) (*types.WorkflowDef, error)
	// UpdateWorkflow 更新工作流定义，返回新版本号（D2修改返回类型）
	UpdateWorkflow(def *types.WorkflowDef) (int, error)
	// DeleteWorkflow 删除工作流定义
	DeleteWorkflow(workflowID string) error
	// StartWorkflow 启动工作流实例，返回实例 ID
	StartWorkflow(workflowID string, inputData map[string]interface{}, createdBy string) (string, error)
	// GetWorkflowInstance 获取流程实例详情
	GetWorkflowInstance(instanceID string) (*types.WorkflowInstance, error)

	// ─── D2 新增：版本管理 ───
	// GetWorkflowVersion 获取指定版本定义
	GetWorkflowVersion(workflowID string, version int) (*types.WorkflowVersion, error)
	// ListWorkflowVersions 列表工作流的所有版本
	ListWorkflowVersions(workflowID string) ([]*types.WorkflowVersion, error)
	// RollbackWorkflow 回滚到指定版本，实际创建新版本
	RollbackWorkflow(workflowID string, targetVersion int, operator string) (int, error)
	// UpdateGrayConfig 更新指定版本的灰度发布配置
	UpdateGrayConfig(workflowID string, version int, cfg *types.GrayReleaseConfig) error

	// ─── D2 新增：生命周期管理 ───
	// PauseInstance 暂停实例执行
	PauseInstance(instanceID string, operator string) error
	// ResumeInstance 恢复实例执行
	ResumeInstance(instanceID string, operator string) error
	// CancelInstance 取消实例执行
	CancelInstance(instanceID string, operator string) error
	// RetryInstance 重试整个实例（从失败节点开始）
	RetryInstance(instanceID string, operator string) error
	// RetryNode 重试单个节点
	RetryNode(instanceID string, nodeID string, operator string) error
	// ListWorkflowInstances 查询实例列表，workflowID 为空则查询全部
	ListWorkflowInstances(workflowID string, status types.WorkflowStatus, pageSize, pageNum int) ([]*types.WorkflowInstance, error)
	// ListDeadLetterTasks 查询实例的死信任务列表
	ListDeadLetterTasks(instanceID string) ([]*types.DeadLetterTask, error)
}

// SchedulerService 调度服务接口
// DAG 解析、依赖检查、任务调度、Worker 分配，是引擎的核心大脑
type SchedulerService interface {
	// StartExecution 启动流程实例的执行调度
	StartExecution(instanceID string) error
	// ScheduleNextNodes 调度实例的下一批可执行节点
	ScheduleNextNodes(instanceID string) error
	// HandleNodeCompleted 处理节点执行完成事件
	HandleNodeCompleted(instanceID string, nodeID string, success bool, output map[string]interface{}, errMsg string) error
	// RecoverUnfinishedInstances 引擎重启时，恢复未完成的实例调度
	RecoverUnfinishedInstances() error

	// ─── D2 新增 ───
	// PauseExecution 暂停实例调度，中断待调度节点的派发
	PauseExecution(instanceID string) error
	// ResumeExecution 恢复实例调度
	ResumeExecution(instanceID string) error
	// CancelExecution 取消实例执行，向所有进行中节点发送取消指令
	CancelExecution(instanceID string) error
	// RetryExecution 重试整个实例执行
	RetryExecution(instanceID string) error
	// RetrySingleNode 重试单个节点
	RetrySingleNode(instanceID string, nodeID string) error
	// HandleWorkerFailure 处理 Worker 故障时的实例调度
	HandleWorkerFailure(workerID string) error
	// ProcessRetryQueue 处理重试队列，换算定时调用
	ProcessRetryQueue() error
}

// InstanceStateService 实例状态管理服务接口
// 流程实例与节点状态的持久化、查询、更新，接收 Worker 的任务结果上报
type InstanceStateService interface {
	// CreateInstance 创建流程实例
	CreateInstance(instance *types.WorkflowInstance) error
	// UpdateInstanceStatus 更新流程实例状态
	UpdateInstanceStatus(instanceID string, status types.WorkflowStatus, errMsg string) error
	// CreateNodeState 创建节点执行状态
	CreateNodeState(state *types.WorkflowNodeState) error
	// UpdateNodeState 更新节点执行状态
	UpdateNodeState(state *types.WorkflowNodeState) error
	// GetNodeState 获取节点执行状态
	GetNodeState(instanceID string, nodeID string) (*types.WorkflowNodeState, error)
	// GetInstanceNodeStates 获取实例的所有节点状态
	GetInstanceNodeStates(instanceID string) ([]*types.WorkflowNodeState, error)
	// ReportTaskResult 接收 Worker 上报的任务执行结果
	ReportTaskResult(result *types.TaskResult) error

	// ─── D2 新增 ───
	// ValidateStateTransition 验证状态转换是否合法
	ValidateStateTransition(fromStatus, toStatus types.WorkflowStatus) error
	// BatchUpdateNodeStatus 批量更新节点状态（取消/跳过时使用）
	BatchUpdateNodeStatus(instanceID string, nodeIDs []string, status types.WorkflowNodeStatus, reason string) error
	// UpdateRetryState 更新节点重试状态
	UpdateRetryState(instanceID string, nodeID string, retryCount int, nextRetryTime *int64) error
	// GetFailedNodes 获取实例中所有失败的节点
	GetFailedNodes(instanceID string) ([]*types.WorkflowNodeState, error)
	// GetAssignedNodesByWorker 获取分配给指定 Worker 的节点列表
	GetAssignedNodesByWorker(workerID string) ([]*types.WorkflowNodeState, error)
}

// WorkerManagerService NodeWorker 管理服务接口
// Worker 服务发现、健康状态管理、负载信息维护、Worker 选择与任务分配
type WorkerManagerService interface {
	// GetOnlineWorkers 获取所有在线的 Worker
	GetOnlineWorkers() []*types.NodeWorkerInfo
	// GetWorkersByNodeType 获取支持指定节点类型的在线 Worker
	GetWorkersByNodeType(nodeType string) []*types.NodeWorkerInfo
	// SelectBestWorker 选择最优的 Worker 执行任务（加权最小负载算法）
	SelectBestWorker(nodeType string) (*types.NodeWorkerInfo, error)
	// AssignTask 向指定 Worker 分配任务
	AssignTask(workerID string, task *types.Task) error
	// HandleWorkerOffline 处理 Worker 下线事件，重新分配未完成的任务
	HandleWorkerOffline(workerID string) error
	// UpdateWorkerLoad 更新 Worker 负载信息
	UpdateWorkerLoad(workerID string, nodeType string, delta int) error
	// RegisterWorker 注册 Worker
	RegisterWorker(info *types.NodeWorkerInfo) error
	// UpdateHeartbeat 更新 Worker 心跳与负载信息
	UpdateHeartbeat(workerID string, currentLoad map[string]int) error

	// ─── D2 新增 ───
	// CheckWorkerHealth 主动健康检查指定 Worker
	CheckWorkerHealth(workerID string) error
	// GetUnhealthyWorkers 获取所有不健康的 Worker
	GetUnhealthyWorkers() []*types.NodeWorkerInfo
	// FailoverWorker 故障转移 Worker 上的所有任务
	FailoverWorker(workerID string) error
	// CleanupOrphanTasks 清理孤儿任务（分配了 Worker 但 Worker 已离线）
	CleanupOrphanTasks() error
}

// HTTPEndpointService HTTP 触发端点服务接口
type HTTPEndpointService interface {
	// Start 启动 HTTP 服务
	Start(addr string) error
	// Stop 停止 HTTP 服务
	Stop() error
	// RegisterWorkflowEndpoint 注册工作流的 HTTP 触发端点
	RegisterWorkflowEndpoint(workflowID string, path string) error
}
