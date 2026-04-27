import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: '/login',
      name: 'Login',
      component: () => import('@/views/Login.vue'),
      meta: { requiresAuth: false },
    },
    {
      path: '/',
      component: () => import('@/layouts/MainLayout.vue'),
      meta: { requiresAuth: true },
      children: [
        {
          path: '',
          redirect: '/workflow/list',
        },
        // ── 工作流管理 ────────────────────────────────────────────────
        {
          path: 'workflow/list',
          name: 'WorkflowList',
          component: () => import('@/views/workflow/WorkflowList.vue'),
          meta: { title: '流程定义', icon: 'List' },
        },
        {
          path: 'workflow/designer/:id?',
          name: 'WorkflowDesigner',
          component: () => import('@/views/workflow/WorkflowDesigner.vue'),
          meta: { title: '流程设计器', icon: 'Edit' },
        },
        // ── 实例监控 ──────────────────────────────────────────────────
        {
          path: 'instance/list',
          name: 'InstanceList',
          component: () => import('@/views/instance/InstanceList.vue'),
          meta: { title: '实例监控', icon: 'Monitor' },
        },
        {
          path: 'instance/:id',
          name: 'InstanceDetail',
          component: () => import('@/views/instance/InstanceDetail.vue'),
          meta: { title: '实例详情', icon: 'Monitor' },
        },
        // ── 审批工作台 ────────────────────────────────────────────────
        {
          path: 'approval',
          name: 'ApprovalWorkbench',
          component: () => import('@/views/approval/ApprovalWorkbench.vue'),
          meta: { title: '审批工作台', icon: 'Check' },
        },
        // ── 端点管理 ──────────────────────────────────────────────────
        {
          path: 'endpoint',
          name: 'EndpointList',
          component: () => import('@/views/endpoint/EndpointList.vue'),
          meta: { title: '端点管理', icon: 'Connection' },
        },
        // ── 审计日志 ──────────────────────────────────────────────────
        {
          path: 'audit',
          name: 'AuditLog',
          component: () => import('@/views/audit/AuditLog.vue'),
          meta: { title: '审计日志', icon: 'Document' },
        },
        // ── 系统管理 ──────────────────────────────────────────────────
        {
          path: 'system/user',
          name: 'UserManage',
          component: () => import('@/views/system/UserManage.vue'),
          meta: { title: '用户管理', icon: 'User' },
        },
        {
          path: 'system/role',
          name: 'RoleManage',
          component: () => import('@/views/system/RoleManage.vue'),
          meta: { title: '角色管理', icon: 'Key' },
        },
        {
          path: 'system/menu',
          name: 'MenuManage',
          component: () => import('@/views/system/MenuManage.vue'),
          meta: { title: '菜单管理', icon: 'Menu' },
        },
        {
          path: 'system/dictionary',
          name: 'DictionaryManage',
          component: () => import('@/views/system/DictionaryManage.vue'),
          meta: { title: '数据字典', icon: 'Collection' },
        },
      ],
    },
    {
      path: '/:pathMatch(.*)*',
      redirect: '/',
    },
  ],
})

// 全局路由守卫
router.beforeEach((to, _from, next) => {
  const authStore = useAuthStore()
  if (to.meta.requiresAuth !== false && !authStore.isLoggedIn) {
    next({ name: 'Login', query: { redirect: to.fullPath } })
  } else {
    next()
  }
})

export default router
