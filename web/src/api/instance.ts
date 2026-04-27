import request from '@/utils/request'

interface InstanceFilter {
  workflowId?: string
  status?: string
  page?: number
  pageSize?: number
}

export const instanceApi = {
  list: (params?: InstanceFilter) =>
    request.get('/workflow/instance', { params }),
  get: (id: string) => request.get(`/workflow/instance/${id}`),
  start: (workflowId: string, inputData: Record<string, unknown>) =>
    request.post('/workflow/instance', { workflowId, inputData }),
  cancel: (id: string, reason: string) =>
    request.post(`/workflow/instance/${id}/cancel`, { reason }),
  pause: (id: string) =>
    request.post(`/workflow/instance/${id}/pause`),
  resume: (id: string) =>
    request.post(`/workflow/instance/${id}/resume`),
}
