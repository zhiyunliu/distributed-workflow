package api

import (
	"context"

	"github.com/zhiyunliu/distributed-workflow/runtime-execution/pkg/types"
)

// AnalyticsService 数据分析服务接口
type AnalyticsService interface {
	// GetWorkflowOverview 获取流程运营总览数据
	GetWorkflowOverview(ctx context.Context) (*types.WorkflowOverviewStats, error)
	// GetWorkflowEfficiency 获取流程效率分析数据
	GetWorkflowEfficiency(ctx context.Context, params types.StatsQueryParams) ([]*types.WorkflowDailyStats, error)
	// GetApprovalPerformance 获取审批绩效分析数据
	GetApprovalPerformance(ctx context.Context, params types.StatsQueryParams) ([]*types.ApprovalPerformanceStats, error)
	// GetInstanceTrend 获取流程实例趋势数据
	GetInstanceTrend(ctx context.Context, params types.StatsQueryParams) (*types.StatsTrendData, error)
	// RefreshDailyStats 刷新指定日期的统计数据（供定时任务调用）
	RefreshDailyStats(ctx context.Context, date string) error
}

// AnalyticsRepository 数据分析仓库接口
type AnalyticsRepository interface {
	GetWorkflowCounts(ctx context.Context) (totalDefs, totalInsts, monthlyNew, activeInsts int64, err error)
	GetAvgExecutionTime(ctx context.Context) (avgMs int64, passRate float64, err error)
	GetWorkflowDailyStats(ctx context.Context, params types.StatsQueryParams) ([]*types.WorkflowDailyStats, error)
	GetApprovalDailyStats(ctx context.Context, params types.StatsQueryParams) ([]*types.ApprovalPerformanceStats, error)
	GetInstanceTrendData(ctx context.Context, params types.StatsQueryParams) (*types.StatsTrendData, error)
	RefreshWorkflowDailyStats(ctx context.Context, date string) error
	RefreshApprovalDailyStats(ctx context.Context, date string) error
}


