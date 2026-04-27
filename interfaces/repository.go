package interfaces

import (
	"time"

	"github.com/zhiyunliu/distributed-workflow/types"
)

// WorkflowRepository SQL Server 持久化接口
// 所有写操作需通过数据库事务保证原子性
type WorkflowRepository interface {
	// ─── 工作流定义 ───────────────────────────────────────────────────
	// CreateWorkflowDef 创建工作流定义
	CreateWorkflowDef(def *types.WorkflowDef) error
	// GetWorkflowDef 根据 ID 获取工作流定义
	GetWorkflowDef(workflowID string) (*types.WorkflowDef, error)
	// UpdateWorkflowDef 更新工作流定义
	UpdateWorkflowDef(def *types.WorkflowDef) error
	// DeleteWorkflowDef 删除工作流定义
	DeleteWorkflowDef(workflowID string) error
	// ListWorkflowDefs 查询所有工作流定义
	ListWorkflowDefs() ([]*types.WorkflowDef, error)

	// ─── 流程实例 ─────────────────────────────────────────────────────
	// CreateWorkflowInstance 创建流程实例
	CreateWorkflowInstance(instance *types.WorkflowInstance) error
	// UpdateWorkflowInstance 更新流程实例
	UpdateWorkflowInstance(instance *types.WorkflowInstance) error
	// GetWorkflowInstance 获取流程实例详情
	GetWorkflowInstance(instanceID string) (*types.WorkflowInstance, error)
	// ListUnfinishedInstances 查询所有未完成的实例（用于引擎重启恢复）
	ListUnfinishedInstances() ([]*types.WorkflowInstance, error)
	// ListWorkflowInstances 分页查询实例列表（D2新增）
	ListWorkflowInstances(workflowID string, status types.WorkflowStatus, pageSize, pageNum int) ([]*types.WorkflowInstance, error)

	// ─── 节点状态 ─────────────────────────────────────────────────────
	// CreateWorkflowNodeState 创建节点执行状态
	CreateWorkflowNodeState(state *types.WorkflowNodeState) error
	// UpdateWorkflowNodeState 更新节点执行状态
	UpdateWorkflowNodeState(state *types.WorkflowNodeState) error
	// GetWorkflowNodeState 获取节点执行状态
	GetWorkflowNodeState(instanceID string, nodeID string) (*types.WorkflowNodeState, error)
	// ListWorkflowNodeStates 获取实例的所有节点状态
	ListWorkflowNodeStates(instanceID string) ([]*types.WorkflowNodeState, error)
	// BatchUpdateNodeStates 批量更新节点状态（D2新增，用于取消/跳过场景）
	BatchUpdateNodeStates(instanceID string, nodeIDs []string, status types.WorkflowNodeStatus, reason string) error
	// GetAssignedNodesByWorker 获取分配给指定 Worker 的节点（D2新增，用于故障转移）
	GetAssignedNodesByWorker(workerID string) ([]*types.WorkflowNodeState, error)
	// GetFailedNodes 获取实例中所有失败的节点（D2新增）
	GetFailedNodes(instanceID string) ([]*types.WorkflowNodeState, error)

	// ─── 上下文 ───────────────────────────────────────────────────────
	// CreateWorkflowContext 创建流程上下文
	CreateWorkflowContext(ctx *types.WorkflowContext) error
	// UpdateWorkflowContext 更新流程上下文
	UpdateWorkflowContext(ctx *types.WorkflowContext) error
	// GetWorkflowContext 获取流程上下文
	GetWorkflowContext(instanceID string) (*types.WorkflowContext, error)
	// DeleteWorkflowContext 删除流程上下文
	DeleteWorkflowContext(instanceID string) error

	// ─── 版本管理（D2新增） ──────────────────────────────────────────
	// CreateWorkflowVersion 创建新版本
	CreateWorkflowVersion(ver *types.WorkflowVersion) error
	// GetWorkflowVersion 获取指定版本
	GetWorkflowVersion(workflowID string, version int) (*types.WorkflowVersion, error)
	// GetCurrentWorkflowVersion 获取当前生效版本
	GetCurrentWorkflowVersion(workflowID string) (*types.WorkflowVersion, error)
	// ListWorkflowVersions 列举所有版本
	ListWorkflowVersions(workflowID string) ([]*types.WorkflowVersion, error)
	// SetCurrentVersion 设置当前生效版本
	SetCurrentVersion(workflowID string, version int) error
	// UpdateVersionGrayConfig 更新版本灰度配置
	UpdateVersionGrayConfig(workflowID string, version int, cfg *types.GrayReleaseConfig) error

	// ─── 死信队列（D2新增） ──────────────────────────────────────────
	// CreateDeadLetterTask 将任务加入死信队列
	CreateDeadLetterTask(task *types.DeadLetterTask) error
	// GetDeadLetterTask 获取死信任务
	GetDeadLetterTask(id string) (*types.DeadLetterTask, error)
	// ListDeadLetterTasks 列举死信任务
	ListDeadLetterTasks(instanceID string) ([]*types.DeadLetterTask, error)
	// UpdateDeadLetterTask 更新死信任务（重发计数等）
	UpdateDeadLetterTask(task *types.DeadLetterTask) error
}

// RedisRepository Redis 存储接口
type RedisRepository interface {
	// ─── 分布式锁 ─────────────────────────────────────────────────────
	// Lock 原子性加锁，成功返回 true
	Lock(key string, value string, ttl time.Duration) (bool, error)
	// Unlock 释放锁，校验持有者后删除，成功返回 true
	Unlock(key string, value string) (bool, error)
	// RenewLock 续期锁，成功返回 true
	RenewLock(key string, value string, ttl time.Duration) (bool, error)

	// ─── 任务队列 ─────────────────────────────────────────────────────
	// EnqueueTask 任务入队（LPUSH）
	EnqueueTask(key string, task string) error
	// DequeueTask 任务出队，阻塞等待（BRPOP）
	DequeueTask(key string, timeout time.Duration) (string, error)
	// GetQueueLength 获取队列长度
	GetQueueLength(key string) (int64, error)

	// ─── 缓存 ─────────────────────────────────────────────────────────
	// SetCache 设置缓存
	SetCache(key string, value string, ttl time.Duration) error
	// GetCache 获取缓存
	GetCache(key string) (string, error)
	// DeleteCache 删除缓存
	DeleteCache(key string) error
}
