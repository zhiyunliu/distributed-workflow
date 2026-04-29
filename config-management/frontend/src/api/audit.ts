import request from '@/utils/request'

interface AuditFilter {
  instanceId?: string
  operator?: string
  operationType?: string
  startTime?: string
  endTime?: string
  page?: number
  pageSize?: number
}

export const auditApi = {
  list: (params?: AuditFilter) => request.get('/audit', { params }),
}
