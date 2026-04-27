package interfaces

import (
	"time"

	"github.com/zhiyunliu/distributed-workflow/types"
)

// NodeExecutor 节点执行器接口，所有自定义节点必须实现该接口
// D1 阶段内置执行器：log（日志打印）、http（HTTP 请求）、sleep（延时执行）
type NodeExecutor interface {
	// Type 返回节点类型，全局唯一
	Type() string
	// Execute 执行节点逻辑
	// config: 节点配置（来自 WorkflowNode.Config）
	// input: 节点输入数据（来自流程上下文）
	// ctx: 流程执行上下文
	// 返回 output（输出数据）和 err（执行错误）
	Execute(config map[string]interface{}, input map[string]interface{}, ctx *types.WorkflowContext) (output map[string]interface{}, err error)
}

// NodeExecutorContainer 节点执行器容器接口，管理执行器的注册与获取
type NodeExecutorContainer interface {
	// Register 注册节点执行器，同类型重复注册返回错误
	Register(executor NodeExecutor) error
	// Get 根据节点类型获取执行器
	Get(nodeType string) (NodeExecutor, error)
	// List 获取所有已注册的节点类型列表
	List() []string
}

// TaskExecutionManager 任务执行管理器接口
type TaskExecutionManager interface {
	// Submit 提交任务到执行队列，按节点类型并发控制
	Submit(task *types.Task) error
	// Cancel 取消正在执行的任务
	Cancel(instanceID string, nodeID string) error
	// GetCurrentLoad 获取当前各节点类型的负载（执行数）
	GetCurrentLoad() map[string]int
}

// DistributedLock 分布式锁接口
type DistributedLock interface {
	// Lock 获取锁，返回是否获取成功
	Lock() (bool, error)
	// Unlock 释放锁，返回是否释放成功
	Unlock() (bool, error)
	// TryLockWithTimeout 带超时时间的锁获取
	TryLockWithTimeout(timeout time.Duration) (bool, error)
}

// TaskQueueManager 任务队列管理器接口
type TaskQueueManager interface {
	// Enqueue 任务入队
	Enqueue(task *types.Task) error
	// Dequeue 任务出队，阻塞等待直到有任务或超时
	Dequeue() (*types.Task, error)
	// Remove 移除指定任务
	Remove(instanceID string, nodeID string) error
	// Len 获取队列长度
	Len() int
}

// ContextManager 流程上下文管理器接口
type ContextManager interface {
	// CreateContext 创建流程上下文
	CreateContext(instanceID string, inputData map[string]interface{}) (*types.WorkflowContext, error)
	// GetContext 获取流程上下文
	GetContext(instanceID string) (*types.WorkflowContext, error)
	// UpdateContext 更新流程上下文，合并传入的数据（不覆盖原有 key）
	UpdateContext(instanceID string, data map[string]interface{}) error
	// DeleteContext 删除流程上下文
	DeleteContext(instanceID string) error
	// SaveSnapshot 保存上下文快照（用于暂停/恢复）（D2新增）
	SaveSnapshot(instanceID string) (snapshotID string, err error)
	// RestoreSnapshot 恢复上下文快照（D2新增）
	RestoreSnapshot(instanceID string, snapshotID string) error
}

// WorkflowLoader 工作流加载器接口，支持从不同数据源加载工作流定义
type WorkflowLoader interface {
	// Load 加载工作流定义
	Load(workflowID string) (*types.WorkflowDef, error)
	// List 加载所有工作流定义
	List() ([]*types.WorkflowDef, error)
}

// ErrorHandler 统一错误处理器接口
type ErrorHandler interface {
	// HandleError 判断是否需要重试，超限则入死信队列
	HandleError(instanceID, nodeID, workerID, workerIP string, errMsg, errCode string, retryCount int, policy *types.RetryPolicy, taskData map[string]interface{}) error
	// MoveToDeadLetter 直接将任务移入死信队列
	MoveToDeadLetter(instanceID, nodeID, workerID, workerIP, errMsg, errCode string, retryCount int, taskData map[string]interface{}) error
	// ResendDeadLetterTask 重发死信任务
	ResendDeadLetterTask(taskID string) error
	// ListDeadLetterTasks 列举实例的死信任务
	ListDeadLetterTasks(instanceID string) ([]*types.DeadLetterTask, error)
}

// WorkflowVersionManager 流程版本管理器接口（D2新增）
type WorkflowVersionManager interface {
	// CreateVersion 创建新版本
	CreateVersion(workflowID string, def *types.WorkflowDef, changeLog string, createdBy string) (*types.WorkflowVersion, error)
	// GetVersion 获取指定版本
	GetVersion(workflowID string, version int) (*types.WorkflowVersion, error)
	// GetCurrentVersion 获取当前生效版本
	GetCurrentVersion(workflowID string) (*types.WorkflowVersion, error)
	// ListVersions 列表所有版本
	ListVersions(workflowID string) ([]*types.WorkflowVersion, error)
	// SetCurrentVersion 设置当前生效版本
	SetCurrentVersion(workflowID string, version int) error
	// SelectVersionForInstance 为新实例选择版本（考虑灰度配置）
	SelectVersionForInstance(workflowID string, tenantID string) (int, error)
}

// LifecycleManager 流程生命周期管理器接口（D2新增）
// 负责暂停、恢复、取消、重试等生命周期操作的状态机验证和协调
type LifecycleManager interface {
	// Pause 暂停实例
	Pause(instanceID string, operator string) error
	// Resume 恢复实例
	Resume(instanceID string, operator string) error
	// Cancel 取消实例
	Cancel(instanceID string, operator string) error
	// Retry 重试实例
	Retry(instanceID string, operator string) error
	// RetryNode 重试单节点
	RetryNode(instanceID string, nodeID string, operator string) error
}

// SubflowManager 子流程管理器接口（D2新增）
type SubflowManager interface {
	// TriggerSubflow 触发子流程
	TriggerSubflow(parentInstanceID string, parentNodeID string, subflowID string, inputData map[string]interface{}, mode types.SubflowCallMode) (string, error)
	// OnSubflowComplete 子流程完成回调
	OnSubflowComplete(subflowInstanceID string, success bool, outputData map[string]interface{}) error
	// GetSubflowDepth 获取子流程嵌套深度
	GetSubflowDepth(instanceID string) (int, error)
}

// FailoverManager 故障转移管理器接口（D2新增）
// 负责 Worker 健康检湋和任务故障转移
type FailoverManager interface {
	// StartHealthCheckLoop 启动健康检查循环（每 10s一次）
	StartHealthCheckLoop()
	// StopHealthCheckLoop 停止健康检查循环
	StopHealthCheckLoop()
	// HealthCheckWorker 检查单个 Worker 健康状态
	HealthCheckWorker(workerID string) (bool, error)
	// FailoverWorker 执行 Worker 故障转移
	FailoverWorker(workerID string) error
}
