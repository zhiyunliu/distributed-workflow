package api

import (
	"time"

	"github.com/zhiyunliu/distributed-workflow/runtime-execution/pkg/types"
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


