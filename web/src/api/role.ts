import request from '@/utils/request'
import type { SystemRole } from '@/types'

interface RoleFilter {
  roleName?: string
  status?: number
  page?: number
  pageSize?: number
}

export const roleApi = {
  list: (params?: RoleFilter) => request.get('/system/role', { params }),
  create: (data: Partial<SystemRole>) => request.post('/system/role', data),
  update: (id: number, data: Partial<SystemRole>) =>
    request.put(`/system/role/${id}`, data),
  delete: (id: number) => request.delete(`/system/role/${id}`),
  getRoleMenus: (id: number) => request.get(`/system/role/${id}/menus`),
  assignMenus: (id: number, menuIds: number[]) =>
    request.put(`/system/role/${id}/menus`, { menuIds }),
}
