import request from '@/utils/request'

export const approvalApi = {
  todoList: (params?: { page?: number; pageSize?: number }) =>
    request.get('/approval/todo', { params }),
  historyList: (params?: { page?: number; pageSize?: number }) =>
    request.get('/approval/history', { params }),
  approve: (instanceId: string, nodeId: string, data: { action: string; comment: string }) =>
    request.post(`/approval/${instanceId}/${nodeId}/approve`, data),
  getPendingTask: (instanceId: string, nodeId: string) =>
    request.get(`/approval/${instanceId}/${nodeId}`),
}
