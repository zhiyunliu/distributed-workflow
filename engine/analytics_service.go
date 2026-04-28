package engine

import (
	"context"
	"fmt"
	"regexp"

	"github.com/zhiyunliu/distributed-workflow/interfaces"
	"github.com/zhiyunliu/distributed-workflow/types"
)

// 编译时接口断言
var _ interfaces.AnalyticsService = (*analyticsService)(nil)

// 日期格式校验正则：YYYY-MM-DD
var reDateFormat = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

// analyticsService BI分析业务服务实现
type analyticsService struct {
	repo interfaces.AnalyticsRepository
}

// NewAnalyticsService 创建分析服务
func NewAnalyticsService(repo interfaces.AnalyticsRepository) interfaces.AnalyticsService {
	return &analyticsService{repo: repo}
}

// GetWorkflowOverview 获取流程运营总览数据
func (s *analyticsService) GetWorkflowOverview(ctx context.Context) (*types.WorkflowOverviewStats, error) {
	totalDefs, totalInsts, monthlyNew, activeInsts, err := s.repo.GetWorkflowCounts(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取流程计数失败: %w", err)
	}

	avgMs, passRate, err := s.repo.GetAvgExecutionTime(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取平均执行时间失败: %w", err)
	}

	return &types.WorkflowOverviewStats{
		TotalDefinitions:    totalDefs,
		TotalInstances:      totalInsts,
		MonthlyNewInstances: monthlyNew,
		ActiveInstances:     activeInsts,
		AvgExecutionTimeMs:  avgMs,
		PassRate:            passRate,
	}, nil
}

// GetWorkflowEfficiency 获取流程效率分析数据
func (s *analyticsService) GetWorkflowEfficiency(ctx context.Context, params types.StatsQueryParams) ([]*types.WorkflowDailyStats, error) {
	if params.StartDate == "" || params.EndDate == "" {
		return nil, fmt.Errorf("startDate和endDate不能为空")
	}
	return s.repo.GetWorkflowDailyStats(ctx, params)
}

// GetApprovalPerformance 获取审批绩效分析数据
func (s *analyticsService) GetApprovalPerformance(ctx context.Context, params types.StatsQueryParams) ([]*types.ApprovalPerformanceStats, error) {
	if params.StartDate == "" || params.EndDate == "" {
		return nil, fmt.Errorf("startDate和endDate不能为空")
	}
	return s.repo.GetApprovalDailyStats(ctx, params)
}

// GetInstanceTrend 获取流程实例趋势数据
func (s *analyticsService) GetInstanceTrend(ctx context.Context, params types.StatsQueryParams) (*types.StatsTrendData, error) {
	if params.StartDate == "" || params.EndDate == "" {
		return nil, fmt.Errorf("startDate和endDate不能为空")
	}
	if params.GroupBy == "" {
		params.GroupBy = "day"
	}
	return s.repo.GetInstanceTrendData(ctx, params)
}

// RefreshDailyStats 刷新指定日期的统计数据（供定时任务调用）
func (s *analyticsService) RefreshDailyStats(ctx context.Context, date string) error {
	if !reDateFormat.MatchString(date) {
		return fmt.Errorf("date格式不正确，应为YYYY-MM-DD")
	}

	if err := s.repo.RefreshWorkflowDailyStats(ctx, date); err != nil {
		return fmt.Errorf("刷新流程日统计失败: %w", err)
	}
	if err := s.repo.RefreshApprovalDailyStats(ctx, date); err != nil {
		return fmt.Errorf("刷新审批日统计失败: %w", err)
	}
	return nil
}
