import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { SystemUser, SystemRole, LoginRequest } from '@/types'
import { authApi } from '@/api/auth'

export const useAuthStore = defineStore('auth', () => {
  const token = ref<string>(localStorage.getItem('token') || '')
  const user = ref<SystemUser | null>(null)
  const roles = ref<SystemRole[]>([])
  const perms = ref<string[]>([])
  const menus = ref<unknown[]>([])

  const isLoggedIn = computed(() => !!token.value)

  async function login(req: LoginRequest) {
    const res = await authApi.login(req)
    const data = res.data.data
    token.value = data.token
    user.value = data.user
    roles.value = data.roles
    perms.value = data.perms
    localStorage.setItem('token', data.token)
    return data
  }

  async function fetchUserInfo() {
    const res = await authApi.getUserInfo()
    const data = res.data.data
    user.value = data.user
    roles.value = data.roles
    perms.value = data.perms
  }

  async function fetchMenus() {
    const res = await authApi.getMenuTree()
    menus.value = res.data.data
  }

  async function logout() {
    try {
      await authApi.logout()
    } finally {
      token.value = ''
      user.value = null
      roles.value = []
      perms.value = []
      localStorage.removeItem('token')
    }
  }

  function hasPermission(perm: string): boolean {
    return perms.value.includes('*:*:*') || perms.value.includes(perm)
  }

  function hasRole(roleCode: string): boolean {
    return roles.value.some((r) => r.roleCode === roleCode)
  }

  return {
    token,
    user,
    roles,
    perms,
    menus,
    isLoggedIn,
    login,
    logout,
    fetchUserInfo,
    fetchMenus,
    hasPermission,
    hasRole,
  }
})
