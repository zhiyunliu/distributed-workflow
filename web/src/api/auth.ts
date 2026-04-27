import request from '@/utils/request'
import type { LoginRequest } from '@/types'

export const authApi = {
  login: (data: LoginRequest) => request.post('/auth/login', data),
  logout: () => request.post('/auth/logout'),
  getUserInfo: () => request.get('/auth/user-info'),
  getMenuTree: () => request.get('/auth/menu'),
  changePassword: (data: { oldPassword: string; newPassword: string }) =>
    request.post('/auth/change-pwd', data),
}
