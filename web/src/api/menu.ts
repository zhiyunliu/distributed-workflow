import request from '@/utils/request'
import type { SystemMenu } from '@/types'

export const menuApi = {
  list: () => request.get('/system/menu'),
  tree: () => request.get('/system/menu/tree'),
  create: (data: Partial<SystemMenu>) => request.post('/system/menu', data),
  update: (id: number, data: Partial<SystemMenu>) =>
    request.put(`/system/menu/${id}`, data),
  delete: (id: number) => request.delete(`/system/menu/${id}`),
}
