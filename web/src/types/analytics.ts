// 分析统计相关 TypeScript 类型定义

export interface WorkflowOverviewStats {
  totalDefinitions: number
  activeDefinitions: number
  totalInstances: number
  runningInstances: number
  completedInstances: number
  failedInstances: number
  avgExecutionTimeMs: number
  passRate: number
}

export interface WorkflowDailyStats {
  statDate: string
  workflowDefId: string
  totalInstances: number
  completedInstances: number
  failedInstances: number
  avgExecutionTimeMs: number
}

export interface ApprovalPerformanceStats {
  statDate: string
  approverUserId: string
  totalApprovals: number
  approvedCount: number
  rejectedCount: number
  avgApprovalTimeMs: number
}

export interface StatsSeries {
  name: string
  data: number[]
}

export interface StatsTrendData {
  dates: string[]
  series: StatsSeries[]
}

export interface StatsQueryParams {
  startDate?: string
  endDate?: string
  workflowDefId?: string
  groupBy?: 'day' | 'week' | 'month'
}
