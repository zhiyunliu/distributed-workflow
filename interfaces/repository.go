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

	// ─── 节点状态 ─────────────────────────────────────────────────────
	// CreateWorkflowNodeState 创建节点执行状态
	CreateWorkflowNodeState(state *types.WorkflowNodeState) error
	// UpdateWorkflowNodeState 更新节点执行状态
	UpdateWorkflowNodeState(state *types.WorkflowNodeState) error
	// GetWorkflowNodeState 获取节点执行状态
	GetWorkflowNodeState(instanceID string, nodeID string) (*types.WorkflowNodeState, error)
	// ListWorkflowNodeStates 获取实例的所有节点状态
	ListWorkflowNodeStates(instanceID string) ([]*types.WorkflowNodeState, error)

	// ─── 上下文 ───────────────────────────────────────────────────────
	// CreateWorkflowContext 创建流程上下文
	CreateWorkflowContext(ctx *types.WorkflowContext) error
	// UpdateWorkflowContext 更新流程上下文
	UpdateWorkflowContext(ctx *types.WorkflowContext) error
	// GetWorkflowContext 获取流程上下文
	GetWorkflowContext(instanceID string) (*types.WorkflowContext, error)
	// DeleteWorkflowContext 删除流程上下文
	DeleteWorkflowContext(instanceID string) error
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
