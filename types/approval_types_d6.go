package types

import "time"

const (
	ApprovalModeCountersign ApprovalMode = "countersign" // 会签：需所有审批人全部通过
	ApprovalModeOrsign      ApprovalMode = "orsign"      // 或签：任意一人通过即可
)

// ApprovalStatus D6扩展状态常量
const (
	ApprovalStatusWaitingSign = "waiting_sign" // 待会签
	ApprovalStatusWaitingSeq  = "waiting_seq"  // 待顺序审批
	ApprovalStatusAddSign     = "add_sign"     // 加签中
	ApprovalStatusTransfered  = "transfered"   // 已转签
	ApprovalStatusReturned    = "returned"     // 已退回
	ApprovalStatusWithdrawn   = "withdrawn"    // 已撤回
	ApprovalStatusCC          = "cc"           // 已抄送
)

// AdvancedApprovalConfig 高级审批配置（用于流程节点配置）
type AdvancedApprovalConfig struct {
	Mode          ApprovalMode `json:"mode"`                    // 审批模式
	PassRate      float64      `json:"passRate,omitempty"`      // 会签通过率（0-1），默认1.0
	TimeoutAction string       `json:"timeoutAction,omitempty"` // 超时处理：auto_approve/auto_reject/escalate
	TimeoutHours  int          `json:"timeoutHours,omitempty"`  // 超时小时数
}

// AddSignRequest 加签请求
type AddSignRequest struct {
	WorkflowInstanceID string   `json:"workflowInstanceId"`
	NodeID             string   `json:"nodeId"`
	OperatorID         string   `json:"operatorId"`
	SignType           string   `json:"signType"`  // before=前加签 after=后加签
	SignUsers          []string `json:"signUsers"` // 加签人员ID列表
	Comment            string   `json:"comment,omitempty"`
}

// TransferRequest 转签请求
type TransferRequest struct {
	WorkflowInstanceID string `json:"workflowInstanceId"`
	NodeID             string `json:"nodeId"`
	OperatorID         string `json:"operatorId"`
	TargetUserID       string `json:"targetUserId"`
	Comment            string `json:"comment,omitempty"`
}

// ReturnRequest 退回请求
type ReturnRequest struct {
	WorkflowInstanceID string `json:"workflowInstanceId"`
	NodeID             string `json:"nodeId"`
	OperatorID         string `json:"operatorId"`
	TargetNodeID       string `json:"targetNodeId"` // 退回到哪个节点（空则退回发起人）
	Comment            string `json:"comment,omitempty"`
}

// WithdrawRequest 撤回请求
type WithdrawRequest struct {
	WorkflowInstanceID string `json:"workflowInstanceId"`
	OperatorID         string `json:"operatorId"`
	Comment            string `json:"comment,omitempty"`
}

// DelegateConfig 审批委托配置
type DelegateConfig struct {
	ID          int64     `json:"id,omitempty"`
	DelegatorID string    `json:"delegatorId"`
	AgentID     string    `json:"agentId"`
	StartTime   time.Time `json:"startTime"`
	EndTime     time.Time `json:"endTime"`
	Status      int       `json:"status"` // 1=有效 2=已失效
}

// CCRequest 抄送请求
type CCRequest struct {
	WorkflowInstanceID string   `json:"workflowInstanceId"`
	NodeID             string   `json:"nodeId"`
	CCUserIDs          []string `json:"ccUserIds"`
}

// CCRecord 抄送记录
type CCRecord struct {
	ID                 int64      `json:"id"`
	WorkflowInstanceID string     `json:"workflowInstanceId"`
	NodeID             string     `json:"nodeId"`
	CCUserID           string     `json:"ccUserId"`
	CCTime             time.Time  `json:"ccTime"`
	IsRead             bool       `json:"isRead"`
	ReadTime           *time.Time `json:"readTime,omitempty"`
}

// AddSignRecord 加签记录
type AddSignRecord struct {
	ID                 int64     `json:"id"`
	WorkflowInstanceID string    `json:"workflowInstanceId"`
	NodeID             string    `json:"nodeId"`
	OperatorID         string    `json:"operatorId"`
	SignType           string    `json:"signType"`
	SignUsers          []string  `json:"signUsers"`
	CreatedAt          time.Time `json:"createdAt"`
}
