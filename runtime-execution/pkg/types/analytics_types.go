package types

// WorkflowOverviewStats 流程运营总览指标
type WorkflowOverviewStats struct {
	TotalDefinitions    int64   `json:"totalDefinitions"`    // 流程定义总数
	TotalInstances      int64   `json:"totalInstances"`      // 实例总数
	MonthlyNewInstances int64   `json:"monthlyNewInstances"` // 本月新增实例数
	ActiveInstances     int64   `json:"activeInstances"`     // 活跃实例数（运行中）
	AvgExecutionTimeMs  int64   `json:"avgExecutionTimeMs"`  // 平均执行时长（毫秒）
	PassRate            float64 `json:"passRate"`            // 流程通过率
}

// WorkflowDailyStats 流程日统计数据
type WorkflowDailyStats struct {
	WorkflowID         string `json:"workflowId"`
	StatDate           string `json:"statDate"`
	StartCount         int    `json:"startCount"`
	CompleteCount      int    `json:"completeCount"`
	FailCount          int    `json:"failCount"`
	TimeoutCount       int    `json:"timeoutCount"`
	RejectCount        int    `json:"rejectCount"`
	AvgExecutionTimeMs int64  `json:"avgExecutionTimeMs"`
	MaxExecutionTimeMs int64  `json:"maxExecutionTimeMs"`
}

// ApprovalPerformanceStats 审批绩效统计
type ApprovalPerformanceStats struct {
	UserID            string `json:"userId"`
	StatDate          string `json:"statDate"`
	ApproveCount      int    `json:"approveCount"`
	RejectCount       int    `json:"rejectCount"`
	TimeoutCount      int    `json:"timeoutCount"`
	AvgApprovalTimeMs int64  `json:"avgApprovalTimeMs"`
}

// StatsTrendData 趋势数据
type StatsTrendData struct {
	Dates  []string      `json:"dates"`
	Series []StatsSeries `json:"series"`
}

// StatsSeries 趋势数据系列
type StatsSeries struct {
	Name string  `json:"name"`
	Data []int64 `json:"data"`
}

// StatsQueryParams 统计查询参数
type StatsQueryParams struct {
	StartDate  string `json:"startDate"`
	EndDate    string `json:"endDate"`
	WorkflowID string `json:"workflowId,omitempty"`
	UserID     string `json:"userId,omitempty"`
	GroupBy    string `json:"groupBy,omitempty"` // day/week/month
}

