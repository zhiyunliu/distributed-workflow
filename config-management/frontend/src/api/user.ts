import request from '@/utils/request'
import type { SystemUser } from '@/types'

interface UserFilter {
  username?: string
  realName?: string
  status?: number
  page?: number
  pageSize?: number
}

export const userApi = {
  list: (params?: UserFilter) => request.get('/system/user', { params }),
  create: (data: Partial<SystemUser> & { password: string }) =>
    request.post('/system/user', data),
  update: (id: number, data: Partial<SystemUser>) =>
    request.put(`/system/user/${id}`, data),
  delete: (id: number) => request.delete(`/system/user/${id}`),
  getUserRoles: (id: number) => request.get(`/system/user/${id}/roles`),
  assignRoles: (id: number, roleIds: number[]) =>
    request.put(`/system/user/${id}/roles`, { roleIds }),
  resetPassword: (id: number, newPassword: string) =>
    request.put(`/system/user/${id}/reset-pwd`, { newPassword }),
}
