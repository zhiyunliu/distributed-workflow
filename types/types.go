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
	// WorkflowVersion 工作流定义版本，D1 阶段固定为 1
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
	// RetryCount 已重试次数，D1 阶段固定为 0
	RetryCount int `json:"retryCount"`
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
}

// WorkerStatus Worker 状态常量
const (
	WorkerStatusOnline  = "online"
	WorkerStatusOffline = "offline"
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
	// StartTime 执行开始时间
	StartTime time.Time `json:"startTime"`
	// EndTime 执行结束时间
	EndTime time.Time `json:"endTime"`
}
