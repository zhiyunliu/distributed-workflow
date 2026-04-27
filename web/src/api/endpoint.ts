import request from '@/utils/request'

export const endpointApi = {
  list: (params?: { page?: number; pageSize?: number }) =>
    request.get('/endpoint', { params }),
  get: (id: string) => request.get(`/endpoint/${id}`),
  create: (data: Record<string, unknown>) => request.post('/endpoint', data),
  update: (id: string, data: Record<string, unknown>) =>
    request.put(`/endpoint/${id}`, data),
  delete: (id: string) => request.delete(`/endpoint/${id}`),
  enable: (id: string) => request.put(`/endpoint/${id}/enable`),
  disable: (id: string) => request.put(`/endpoint/${id}/disable`),
  triggerHistory: (id: string, params?: { page?: number; pageSize?: number }) =>
    request.get(`/endpoint/${id}/trigger-history`, { params }),
  test: (id: string) => request.post(`/endpoint/${id}/test`),
  validateCron: (expr: string) =>
    request.post('/endpoint/cron/validate', { expr }),
}
