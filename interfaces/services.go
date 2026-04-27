package interfaces

import "github.com/zhiyunliu/distributed-workflow/types"

// WorkflowService 工作流定义管理服务接口
// 负责工作流定义的 CRUD、流程实例的创建与启动、对外提供 HTTP 触发接口
type WorkflowService interface {
	// CreateWorkflow 创建工作流定义，返回工作流 ID
	CreateWorkflow(def *types.WorkflowDef) (string, error)
	// GetWorkflow 根据 ID 获取工作流定义
	GetWorkflow(workflowID string) (*types.WorkflowDef, error)
	// UpdateWorkflow 更新工作流定义
	UpdateWorkflow(def *types.WorkflowDef) error
	// DeleteWorkflow 删除工作流定义
	DeleteWorkflow(workflowID string) error
	// StartWorkflow 启动工作流实例，返回实例 ID
	StartWorkflow(workflowID string, inputData map[string]interface{}, createdBy string) (string, error)
	// GetWorkflowInstance 获取流程实例详情
	GetWorkflowInstance(instanceID string) (*types.WorkflowInstance, error)
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
