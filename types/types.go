package types

import "time"

// ─────────────────────────────────────────────────────────────────────────────
// 枚举类型
// ─────────────────────────────────────────────────────────────────────────────

// WorkflowStatus 工作流实例状态
type WorkflowStatus string

const (
	// WorkflowStatusPending 待执行：实例已创建，尚未启动调度
	WorkflowStatusPending WorkflowStatus = "pending"
	// WorkflowStatusRunning 执行中：实例正在调度执行
	WorkflowStatusRunning WorkflowStatus = "running"
	// WorkflowStatusCompleted 已完成：所有节点执行成功，流程正常结束
	WorkflowStatusCompleted WorkflowStatus = "completed"
	// WorkflowStatusFailed 执行失败：节点执行失败，无后续可执行节点，流程异常结束
	WorkflowStatusFailed WorkflowStatus = "failed"
	// WorkflowStatusPaused 已暂停：流程暂停执行，等待恢复（D2新增）
	WorkflowStatusPaused WorkflowStatus = "paused"
	// WorkflowStatusCancelled 已取消：手动终止执行（D2新增）
	WorkflowStatusCancelled WorkflowStatus = "cancelled"
	// WorkflowStatusRetrying 重试中：流程正在执行重试逻辑（D2新增）
	WorkflowStatusRetrying WorkflowStatus = "retrying"
)

// WorkflowNodeStatus 工作流节点执行状态
type WorkflowNodeStatus string

const (
	// WorkflowNodeStatusPending 待执行：节点依赖未满足，等待调度
	WorkflowNodeStatusPending WorkflowNodeStatus = "pending"
	// WorkflowNodeStatusAssigned 已分配：已分配给 NodeWorker，等待执行
	WorkflowNodeStatusAssigned WorkflowNodeStatus = "assigned"
	// WorkflowNodeStatusRunning 执行中：NodeWorker 正在执行节点逻辑
	WorkflowNodeStatusRunning WorkflowNodeStatus = "running"
	// WorkflowNodeStatusCompleted 已完成：节点执行成功
	WorkflowNodeStatusCompleted WorkflowNodeStatus = "completed"
	// WorkflowNodeStatusFailed 执行失败：节点执行异常
	WorkflowNodeStatusFailed WorkflowNodeStatus = "failed"
	// WorkflowNodeStatusSkipped 已跳过：流程终止时跳过未执行的节点（D2新增）
	WorkflowNodeStatusSkipped WorkflowNodeStatus = "skipped"
	// WorkflowNodeStatusCancelled 已取消：节点执行被手动终止（D2新增）
	WorkflowNodeStatusCancelled WorkflowNodeStatus = "cancelled"
	// WorkflowNodeStatusQueued 重试队列中：等待重试执行（D2新增）
	WorkflowNodeStatusQueued WorkflowNodeStatus = "queued"
)

// ConnectionType 流程连线类型
type ConnectionType string

const (
	// ConnectionTypeSuccess 成功分支：源节点执行成功后，执行目标节点
	ConnectionTypeSuccess ConnectionType = "success"
	// ConnectionTypeFailure 失败分支：源节点执行失败后，执行目标节点
	ConnectionTypeFailure ConnectionType = "failure"
	// ConnectionTypeAlways 无条件分支：无论源节点执行成败，都执行目标节点
	ConnectionTypeAlways ConnectionType = "always"
)

// DependencyMode 节点前置依赖模式
type DependencyMode string

const (
	// DependencyModeAll 全量依赖：所有前置节点完成后，才执行当前节点
	DependencyModeAll DependencyMode = "all"
	// DependencyModeAny 任意依赖：任意一个前置节点完成后，即可执行当前节点
	DependencyModeAny DependencyMode = "any"
)

// RetryStrategyType 重试策略类型（D2新增）
type RetryStrategyType string

const (
	// RetryStrategyFixed 固定间隔重试
	RetryStrategyFixed RetryStrategyType = "fixed"
	// RetryStrategyExponential 指数退避重试
	RetryStrategyExponential RetryStrategyType = "exponential"
)

// GrayReleaseType 灰度发布类型（D2新增）
type GrayReleaseType string

const (
	// GrayReleaseByRatio 按比例灰度
	GrayReleaseByRatio GrayReleaseType = "ratio"
	// GrayReleaseByWhitelist 按白名单灰度
	GrayReleaseByWhitelist GrayReleaseType = "whitelist"
)

// SubflowCallMode 子流程调用模式（D2新增）
type SubflowCallMode string

const (
	// SubflowCallModeSync 同步调用：父流程等待子流程执行完成后继续
	SubflowCallModeSync SubflowCallMode = "sync"
	// SubflowCallModeAsync 异步调用：父流程触发子流程后立即继续执行
	SubflowCallModeAsync SubflowCallMode = "async"
)

// ─────────────────────────────────────────────────────────────────────────────
// 流程定义类型
// ─────────────────────────────────────────────────────────────────────────────

// WorkflowDef 工作流定义，流程执行的规则蓝本
type WorkflowDef struct {
	// ID 工作流唯一标识，全局唯一
	ID string `json:"id" validate:"required,max=64"`
	// Name 工作流名称
	Name string `json:"name" validate:"required,max=255"`
	// Description 工作流描述
	Description string `json:"description"`
	// Nodes 流程节点集合，key 为节点 ID
	Nodes map[string]*WorkflowNode `json:"nodes" validate:"required,min=1"`
	// Connections 节点连线集合，定义节点间的执行顺序
	Connections []*Connection `json:"connections" validate:"required,min=1"`
	// StartNodeID 起始节点 ID，必须存在于 Nodes 集合中
	StartNodeID string `json:"startNodeId" validate:"required"`
	// EndNodeID 结束节点 ID，必须存在于 Nodes 集合中
	EndNodeID string `json:"endNodeId" validate:"required"`
	// CreatedAt 创建时间
	CreatedAt time.Time `json:"createdAt"`
	// UpdatedAt 更新时间
	UpdatedAt time.Time `json:"updatedAt"`
	// CreatedBy 创建人
	CreatedBy string `json:"createdBy"`
	// UpdatedBy 更新人
	UpdatedBy string `json:"updatedBy"`
	// Disabled 是否禁用
	Disabled bool `json:"disabled"`
	// CurrentVersion 当前生效版本号（D2新增）
	CurrentVersion int `json:"currentVersion"`
	// GlobalRetryPolicy 全局重试策略，节点未配置时使用全局配置（D2新增）
	GlobalRetryPolicy *RetryPolicy `json:"globalRetryPolicy"`
	// Timeout 流程全局超时时间，单位秒，0 表示不限制（D2新增）
	Timeout int `json:"timeout"`
}

// WorkflowNode 工作流执行节点，流程执行的最小单元
type WorkflowNode struct {
	// ID 节点唯一标识，同一工作流内唯一
	ID string `json:"id" validate:"required,max=64"`
	// Type 节点类型，对应 NodeWorker 注册的执行器类型
	Type string `json:"type" validate:"required,max=64"`
	// Name 节点名称
	Name string `json:"name" validate:"required,max=255"`
	// Description 节点描述
	Description string `json:"description"`
	// Config 节点执行配置，不同类型节点有不同的配置结构
	Config map[string]interface{} `json:"config"`
	// DependencyMode 前置依赖模式，默认 all
	DependencyMode DependencyMode `json:"dependencyMode"`
	// Timeout 节点执行超时时间，单位秒，0 表示不限制
	Timeout int `json:"timeout"`
	// RetryPolicy 节点级重试策略，优先级高于全局策略（D2新增）
	RetryPolicy *RetryPolicy `json:"retryPolicy"`
	// SkipOnError 执行失败时是否跳过该节点，继续执行后续流程（D2新增）
	SkipOnError bool `json:"skipOnError"`
	// IsSubflowNode 是否为子流程节点（D2新增）
	IsSubflowNode bool `json:"isSubflowNode"`
	// SubflowConfig 子流程配置，仅子流程节点生效（D2新增）
	SubflowConfig *SubflowNodeConfig `json:"subflowConfig"`
}

// Connection 流程连线，定义节点间的执行顺序与分支规则
type Connection struct {
	// ID 连线唯一标识，同一工作流内唯一
	ID string `json:"id" validate:"required,max=64"`
	// SourceID 源节点 ID
	SourceID string `json:"sourceId" validate:"required"`
	// TargetID 目标节点 ID
	TargetID string `json:"targetId" validate:"required"`
	// Type 连线类型，默认 success
	Type ConnectionType `json:"type"`
}

// ─────────────────────────────────────────────────────────────────────────────
// 运行时类型
// ─────────────────────────────────────────────────────────────────────────────

// WorkflowInstance 工作流实例，对应一次工作流的运行时执行
type WorkflowInstance struct {
	// ID 实例唯一标识，全局唯一
	ID string `json:"id"`
	// WorkflowID 关联的工作流定义 ID
	WorkflowID string `json:"workflowId"`
	// WorkflowVersion 工作流定义版本
	WorkflowVersion int `json:"workflowVersion"`
	// Status 实例执行状态
	Status WorkflowStatus `json:"status"`
	// StartTime 实例启动时间
	StartTime time.Time `json:"startTime"`
	// EndTime 实例结束时间
	EndTime *time.Time `json:"endTime,omitempty"`
	// InputData 流程启动时的输入参数
	InputData map[string]interface{} `json:"inputData"`
	// OutputData 流程执行完成后的输出结果
	OutputData map[string]interface{} `json:"outputData"`
	// CreatedBy 流程创建人
	CreatedBy string `json:"createdBy"`
	// ErrorMessage 流程执行失败时的错误信息
	ErrorMessage string `json:"errorMessage"`
	// PauseTime 暂停时间（D2新增）
	PauseTime *time.Time `json:"pauseTime,omitempty"`
	// CancelTime 取消时间（D2新增）
	CancelTime *time.Time `json:"cancelTime,omitempty"`
	// ParentInstanceID 父流程实例ID，子流程实例生效（D2新增）
	ParentInstanceID string `json:"parentInstanceId"`
	// ParentNodeID 父流程节点ID，子流程实例生效（D2新增）
	ParentNodeID string `json:"parentNodeId"`
	// IsSubflowInstance 是否为子流程实例（D2新增）
	IsSubflowInstance bool `json:"isSubflowInstance"`
	// PauseNodeIDs 暂停时已完成的节点ID列表（D2新增）
	PauseNodeIDs []string `json:"pauseNodeIds"`
	// Operator 最后操作人（D2新增）
	Operator string `json:"operator"`
}

// WorkflowNodeState 节点执行状态，记录单个节点的执行详情
type WorkflowNodeState struct {
	// ID 自增主键
	ID int64 `json:"id"`
	// InstanceID 关联的流程实例 ID
	InstanceID string `json:"instanceId"`
	// NodeID 关联的节点 ID
	NodeID string `json:"nodeId"`
	// Status 节点执行状态
	Status WorkflowNodeStatus `json:"status"`
	// AssignedTo 分配的 NodeWorker ID
	AssignedTo string `json:"assignedTo"`
	// WorkerIP 分配的 NodeWorker 服务器 IP 地址
	WorkerIP string `json:"workerIp"`
	// StartTime 节点开始执行时间
	StartTime *time.Time `json:"startTime,omitempty"`
	// EndTime 节点执行结束时间
	EndTime *time.Time `json:"endTime,omitempty"`
	// InputData 节点执行的输入数据
	InputData map[string]interface{} `json:"inputData"`
	// OutputData 节点执行的输出数据
	OutputData map[string]interface{} `json:"outputData"`
	// ErrorMessage 节点执行失败时的错误信息
	ErrorMessage string `json:"errorMessage"`
	// RetryCount 已重试次数
	RetryCount int `json:"retryCount"`
	// NextRetryTime 下一次重试时间（D2新增）
	NextRetryTime *time.Time `json:"nextRetryTime,omitempty"`
	// LastRetryTime 上一次重试时间（D2新增）
	LastRetryTime *time.Time `json:"lastRetryTime,omitempty"`
	// ErrorCode 错误码（D2新增）
	ErrorCode string `json:"errorCode"`
	// SkippedReason 跳过原因（D2新增）
	SkippedReason string `json:"skippedReason"`
}

// WorkflowContext 流程执行上下文，用于节点间数据传递
type WorkflowContext struct {
	// InstanceID 流程实例 ID
	InstanceID string `json:"instanceId"`
	// Data 上下文数据集合
	Data map[string]interface{} `json:"data"`
	// CreatedAt 创建时间
	CreatedAt time.Time `json:"createdAt"`
	// UpdatedAt 更新时间
	UpdatedAt time.Time `json:"updatedAt"`
}

// NodeWorkerInfo NodeWorker 节点信息
type NodeWorkerInfo struct {
	// ID Worker 唯一标识，全局唯一
	ID string `json:"id"`
	// Address Worker gRPC 服务地址，格式：ip:port
	Address string `json:"address"`
	// IP Worker 服务器 IP 地址
	IP string `json:"ip"`
	// Capabilities 节点执行能力，key 为节点类型，value 为该类型的最大并发数
	Capabilities map[string]int `json:"capabilities"`
	// CurrentLoad 当前负载，key 为节点类型，value 为当前正在执行的任务数
	CurrentLoad map[string]int `json:"currentLoad"`
	// LastHeartbeat 最后一次心跳时间
	LastHeartbeat time.Time `json:"lastHeartbeat"`
	// Status Worker 状态：online/offline
	Status string `json:"status"`
	// HealthStatus 健康状态：healthy/unhealthy/offline（D2新增）
	HealthStatus string `json:"healthStatus"`
	// FailedHeartbeatCount 连续心跳失败次数（D2新增）
	FailedHeartbeatCount int `json:"failedHeartbeatCount"`
	// RegisterTime 注册时间（D2新增）
	RegisterTime time.Time `json:"registerTime"`
}

// WorkerStatus Worker 状态常量
const (
	WorkerStatusOnline  = "online"
	WorkerStatusOffline = "offline"
)

// WorkerHealthStatus Worker 健康状态常量（D2新增）
const (
	WorkerHealthStatusHealthy   = "healthy"
	WorkerHealthStatusUnhealthy = "unhealthy"
)

// Task 待执行任务
type Task struct {
	// TaskID 任务唯一 ID，格式：instanceId-nodeId
	TaskID string `json:"taskId"`
	// InstanceID 流程实例 ID
	InstanceID string `json:"instanceId"`
	// NodeID 节点 ID
	NodeID string `json:"nodeId"`
	// NodeType 节点类型
	NodeType string `json:"nodeType"`
	// Config 节点执行配置
	Config map[string]interface{} `json:"config"`
	// InputData 节点执行输入数据
	InputData map[string]interface{} `json:"inputData"`
	// Timeout 执行超时时间，单位秒，0 表示不限制
	Timeout int `json:"timeout"`
}

// TaskResult 任务执行结果
type TaskResult struct {
	// TaskID 任务 ID
	TaskID string `json:"taskId"`
	// InstanceID 流程实例 ID
	InstanceID string `json:"instanceId"`
	// NodeID 节点 ID
	NodeID string `json:"nodeId"`
	// Success 是否执行成功
	Success bool `json:"success"`
	// OutputData 执行输出数据
	OutputData map[string]interface{} `json:"outputData"`
	// ErrorMessage 错误信息
	ErrorMessage string `json:"errorMessage"`
	// ErrorCode 错误码（D2新增）
	ErrorCode string `json:"errorCode"`
	// StartTime 执行开始时间
	StartTime time.Time `json:"startTime"`
	// EndTime 执行结束时间
	EndTime time.Time `json:"endTime"`
}

// ─────────────────────────────────────────────────────────────────────────────
// D2 新增类型
// ─────────────────────────────────────────────────────────────────────────────

// RetryPolicy 节点重试策略（D2新增）
type RetryPolicy struct {
	// Strategy 重试策略类型
	Strategy RetryStrategyType `json:"strategy"`
	// MaxRetryCount 最大重试次数，0表示不重试
	MaxRetryCount int `json:"maxRetryCount"`
	// RetryInterval 重试间隔，单位秒，固定间隔策略生效
	RetryInterval int `json:"retryInterval"`
	// MaxInterval 最大重试间隔，单位秒，指数退避策略生效
	MaxInterval int `json:"maxInterval"`
	// Multiplier 指数退避乘数，默认2
	Multiplier float64 `json:"multiplier"`
	// RetryOnErrorCodes 可重试的错误码，为空则所有错误都重试
	RetryOnErrorCodes []string `json:"retryOnErrorCodes"`
}

// GrayReleaseConfig 灰度发布配置（D2新增）
type GrayReleaseConfig struct {
	// Type 灰度类型
	Type GrayReleaseType `json:"type"`
	// Ratio 灰度比例，0-100，仅按比例灰度时生效
	Ratio int `json:"ratio"`
	// Whitelist 白名单列表，仅按白名单灰度时生效
	Whitelist []string `json:"whitelist"`
	// Enabled 是否开启灰度
	Enabled bool `json:"enabled"`
}

// WorkflowVersion 流程版本定义（D2新增）
type WorkflowVersion struct {
	// ID 自增主键
	ID int64 `json:"id"`
	// WorkflowID 关联的流程定义ID
	WorkflowID string `json:"workflowId"`
	// Version 版本号，从1开始自增
	Version int `json:"version"`
	// Definition 该版本的流程定义完整JSON
	Definition *WorkflowDef `json:"definition"`
	// ChangeLog 版本变更说明
	ChangeLog string `json:"changeLog"`
	// CreatedBy 创建人
	CreatedBy string `json:"createdBy"`
	// CreatedAt 创建时间
	CreatedAt time.Time `json:"createdAt"`
	// IsCurrent 是否为当前生效版本
	IsCurrent bool `json:"isCurrent"`
	// GrayConfig 灰度发布配置
	GrayConfig *GrayReleaseConfig `json:"grayConfig"`
}

// SubflowNodeConfig 子流程节点配置（D2新增）
type SubflowNodeConfig struct {
	// SubflowID 被调用的子流程定义ID
	SubflowID string `json:"subflowId"`
	// CallMode 调用模式
	CallMode SubflowCallMode `json:"callMode"`
	// InputMapping 输入参数映射，父流程上下文 key → 子流程输入 key
	InputMapping map[string]string `json:"inputMapping"`
	// OutputMapping 输出参数映射，子流程输出 key → 父流程上下文 key
	OutputMapping map[string]string `json:"outputMapping"`
	// WaitForSubflow 异步模式下，是否等待子流程完成后标记节点成功
	WaitForSubflow bool `json:"waitForSubflow"`
}

// DeadLetterTask 死信任务（D2新增）
type DeadLetterTask struct {
	// ID 死信任务唯一ID
	ID string `json:"id"`
	// InstanceID 流程实例ID
	InstanceID string `json:"instanceId"`
	// NodeID 节点ID
	NodeID string `json:"nodeId"`
	// WorkerID 执行Worker ID
	WorkerID string `json:"workerId"`
	// WorkerIP 执行Worker IP
	WorkerIP string `json:"workerIp"`
	// Error 错误信息
	Error string `json:"error"`
	// ErrorCode 错误码
	ErrorCode string `json:"errorCode"`
	// RetryCount 已重试次数
	RetryCount int `json:"retryCount"`
	// TaskData 任务数据快照
	TaskData map[string]interface{} `json:"taskData"`
	// CreatedAt 进入死信时间
	CreatedAt time.Time `json:"createdAt"`
	// ResendCount 已重发次数
	ResendCount int `json:"resendCount"`
	// LastResendAt 最后一次重发时间
	LastResendAt *time.Time `json:"lastResendAt,omitempty"`
}
