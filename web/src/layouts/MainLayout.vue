<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { ElMessage, ElMessageBox } from 'element-plus'

const router = useRouter()
const authStore = useAuthStore()
const isCollapse = ref(false)

const menuItems = [
  {
    title: '工作流管理',
    icon: 'Setting',
    children: [
      { title: '流程定义', icon: 'List', path: '/workflow/list' },
      { title: '流程设计器', icon: 'Edit', path: '/workflow/designer' },
    ],
  },
  {
    title: '实例监控',
    icon: 'Monitor',
    path: '/instance/list',
  },
  {
    title: '审批工作台',
    icon: 'Check',
    path: '/approval',
  },
  {
    title: '端点管理',
    icon: 'Connection',
    path: '/endpoint',
  },
  {
    title: '审计日志',
    icon: 'Document',
    path: '/audit',
  },
  {
    title: '系统管理',
    icon: 'Tools',
    children: [
      { title: '用户管理', icon: 'User', path: '/system/user' },
      { title: '角色管理', icon: 'Key', path: '/system/role' },
      { title: '菜单管理', icon: 'Menu', path: '/system/menu' },
      { title: '数据字典', icon: 'Collection', path: '/system/dictionary' },
    ],
  },
]

onMounted(async () => {
  if (!authStore.user) {
    await authStore.fetchUserInfo()
  }
})

async function handleLogout() {
  await ElMessageBox.confirm('确定要退出登录吗？', '提示', {
    confirmButtonText: '确定',
    cancelButtonText: '取消',
    type: 'warning',
  })
  await authStore.logout()
  ElMessage.success('退出成功')
  router.push('/login')
}
</script>

<template>
  <el-container class="layout-container">
    <!-- 侧边栏 -->
    <el-aside :width="isCollapse ? '64px' : '220px'" class="aside">
      <div class="logo">
        <span v-if="!isCollapse" class="logo-text">分布式工作流</span>
        <el-icon v-else><Setting /></el-icon>
      </div>
      <el-menu
        :collapse="isCollapse"
        :collapse-transition="false"
        router
        class="side-menu"
      >
        <template v-for="item in menuItems" :key="item.path ?? item.title">
          <el-sub-menu v-if="item.children" :index="item.title">
            <template #title>
              <el-icon><component :is="item.icon" /></el-icon>
              <span>{{ item.title }}</span>
            </template>
            <el-menu-item
              v-for="child in item.children"
              :key="child.path"
              :index="child.path"
            >
              <el-icon><component :is="child.icon" /></el-icon>
              <span>{{ child.title }}</span>
            </el-menu-item>
          </el-sub-menu>
          <el-menu-item v-else :index="item.path!">
            <el-icon><component :is="item.icon" /></el-icon>
            <span>{{ item.title }}</span>
          </el-menu-item>
        </template>
      </el-menu>
    </el-aside>

    <!-- 主体 -->
    <el-container>
      <!-- 顶部栏 -->
      <el-header class="header">
        <div class="header-left">
          <el-icon class="collapse-btn" @click="isCollapse = !isCollapse">
            <Fold v-if="!isCollapse" />
            <Expand v-else />
          </el-icon>
        </div>
        <div class="header-right">
          <el-dropdown @command="handleLogout">
            <span class="user-info">
              <el-avatar :size="28" src="https://cube.elemecdn.com/3/7c/3ea6beec64369c2642b92c6726f1epng.png" />
              <span class="username">{{ authStore.user?.realName || authStore.user?.username }}</span>
              <el-icon><ArrowDown /></el-icon>
            </span>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item command="logout">退出登录</el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </div>
      </el-header>

      <!-- 内容区 -->
      <el-main class="main-content">
        <router-view />
      </el-main>
    </el-container>
  </el-container>
</template>

<style scoped>
.layout-container {
  height: 100vh;
}

.aside {
  background-color: #304156;
  transition: width 0.3s;
  overflow: hidden;
}

.logo {
  height: 60px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  font-size: 18px;
  font-weight: bold;
  background-color: #263445;
}

.logo-text {
  white-space: nowrap;
  overflow: hidden;
}

.side-menu {
  border-right: none;
  background-color: #304156;
  flex: 1;
  overflow-y: auto;
}

:deep(.el-menu-item),
:deep(.el-sub-menu__title) {
  color: #bfcbd9 !important;
}

:deep(.el-menu-item:hover),
:deep(.el-sub-menu__title:hover) {
  background-color: #263445 !important;
}

:deep(.el-menu-item.is-active) {
  color: #409eff !important;
  background-color: #263445 !important;
}

.header {
  background-color: #fff;
  border-bottom: 1px solid #e6e6e6;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 16px;
  height: 60px;
}

.collapse-btn {
  font-size: 20px;
  cursor: pointer;
  color: #606266;
}

.header-right {
  display: flex;
  align-items: center;
  gap: 12px;
}

.user-info {
  display: flex;
  align-items: center;
  gap: 8px;
  cursor: pointer;
  color: #606266;
}

.username {
  font-size: 14px;
}

.main-content {
  background-color: #f0f2f5;
  overflow-y: auto;
}
</style>
