import request from '@/utils/request'
import type { ApiResponse } from '@/types'
import type {
  WorkflowOverviewStats,
  WorkflowDailyStats,
  ApprovalPerformanceStats,
  StatsTrendData,
  StatsQueryParams,
} from '@/types/analytics'

export const analyticsApi = {
  // 工作流总览统计
  getWorkflowOverview: () =>
    request.get<ApiResponse<WorkflowOverviewStats>>('/api/stats/workflow/overview'),

  // 工作流效率分析
  getWorkflowEfficiency: (params: StatsQueryParams) =>
    request.get<ApiResponse<WorkflowDailyStats[]>>('/api/stats/workflow/efficiency', { params }),

  // 审批绩效统计
  getApprovalPerformance: (params: StatsQueryParams) =>
    request.get<ApiResponse<ApprovalPerformanceStats[]>>('/api/stats/approval/performance', { params }),

  // 实例趋势数据
  getInstanceTrend: (params: StatsQueryParams) =>
    request.get<ApiResponse<StatsTrendData>>('/api/stats/instance/trend', { params }),

  // 刷新每日统计
  refreshDailyStats: (date: string) =>
    request.post('/api/stats/refresh', { date }),
}
