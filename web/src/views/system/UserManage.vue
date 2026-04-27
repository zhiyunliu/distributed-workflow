<script setup lang="ts">
import { ref, onMounted, reactive } from 'vue'
import { userApi } from '@/api/user'
import { roleApi } from '@/api/role'
import type { SystemUser, SystemRole } from '@/types'
import { ElMessage, ElMessageBox } from 'element-plus'

const loading = ref(false)
const list = ref<SystemUser[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)

const searchForm = reactive({ username: '', status: undefined as number | undefined })

// 用户对话框
const dialogVisible = ref(false)
const isEdit = ref(false)
const currentUser = ref<Partial<SystemUser> & { password?: string; roleIds?: number[] }>({})

// 角色分配对话框
const roleDialogVisible = ref(false)
const allRoles = ref<SystemRole[]>([])
const assignedRoleIds = ref<number[]>([])
const currentUserId = ref<number>(0)

const formRef = ref()

const rules = {
  username: [{ required: true, message: '请输入用户名', trigger: 'blur' }],
  password: [{ required: !isEdit.value, message: '请输入密码', trigger: 'blur' }],
}

async function loadList() {
  loading.value = true
  try {
    const res = await userApi.list({ ...searchForm, page: page.value, pageSize: pageSize.value })
    const data = res.data.data
    list.value = data?.list || data || []
    total.value = data?.total ?? list.value.length
  } finally {
    loading.value = false
  }
}

function openCreate() {
  isEdit.value = false
  currentUser.value = { status: 1 }
  dialogVisible.value = true
}

function openEdit(row: SystemUser) {
  isEdit.value = true
  currentUser.value = { ...row }
  dialogVisible.value = true
}

async function openRoleAssign(row: SystemUser) {
  currentUserId.value = row.id
  const [rolesRes, assignedRes] = await Promise.all([
    roleApi.list(),
    userApi.getUserRoles(row.id),
  ])
  allRoles.value = rolesRes.data.data?.list || rolesRes.data.data || []
  const assigned = assignedRes.data.data || []
  assignedRoleIds.value = assigned.map((r: SystemRole) => r.id)
  roleDialogVisible.value = true
}

async function handleSave() {
  await formRef.value?.validate()
  try {
    if (isEdit.value) {
      await userApi.update(currentUser.value.id!, currentUser.value)
      ElMessage.success('更新成功')
    } else {
      await userApi.create(currentUser.value as SystemUser & { password: string })
      ElMessage.success('创建成功')
    }
    dialogVisible.value = false
    loadList()
  } catch {
    // handled
  }
}

async function handleDelete(row: SystemUser) {
  await ElMessageBox.confirm(`确定删除用户 "${row.username}" 吗？`, '警告', { type: 'warning' })
  await userApi.delete(row.id)
  ElMessage.success('删除成功')
  loadList()
}

async function handleAssignRoles() {
  await userApi.assignRoles(currentUserId.value, assignedRoleIds.value)
  ElMessage.success('角色分配成功')
  roleDialogVisible.value = false
}

async function handleResetPwd(row: SystemUser) {
  const { value } = await ElMessageBox.prompt('请输入新密码', '重置密码', {
    confirmButtonText: '确定',
    cancelButtonText: '取消',
    inputType: 'password',
    inputPattern: /.{6,}/,
    inputErrorMessage: '密码不能少于6位',
  })
  await userApi.resetPassword(row.id, value)
  ElMessage.success('密码重置成功')
}

onMounted(loadList)
</script>

<template>
  <div class="page-container">
    <el-card class="filter-card">
      <el-form :model="searchForm" inline>
        <el-form-item label="用户名">
          <el-input v-model="searchForm.username" placeholder="用户名" clearable />
        </el-form-item>
        <el-form-item label="状态">
          <el-select v-model="searchForm.status" style="width: 120px" clearable>
            <el-option label="启用" :value="1" />
            <el-option label="禁用" :value="0" />
          </el-select>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="loadList">搜索</el-button>
          <el-button @click="Object.assign(searchForm, { username: '', status: undefined }); loadList()">重置</el-button>
        </el-form-item>
      </el-form>
    </el-card>

    <el-card>
      <template #header>
        <div class="card-header">
          <span>用户管理</span>
          <el-button type="primary" @click="openCreate">
            <el-icon><Plus /></el-icon>新建用户
          </el-button>
        </div>
      </template>

      <el-table v-loading="loading" :data="list" stripe>
        <el-table-column prop="id" label="ID" width="80" />
        <el-table-column prop="username" label="用户名" width="140" />
        <el-table-column prop="realName" label="姓名" width="120" />
        <el-table-column prop="email" label="邮箱" />
        <el-table-column prop="phone" label="电话" width="140" />
        <el-table-column label="状态" width="80">
          <template #default="{ row }">
            <el-tag :type="row.status === 1 ? 'success' : 'danger'">
              {{ row.status === 1 ? '启用' : '禁用' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="createTime" label="创建时间" width="180" />
        <el-table-column label="操作" width="240" fixed="right">
          <template #default="{ row }">
            <el-button type="primary" link @click="openEdit(row)">编辑</el-button>
            <el-button type="primary" link @click="openRoleAssign(row)">分配角色</el-button>
            <el-button type="warning" link @click="handleResetPwd(row)">重置密码</el-button>
            <el-button type="danger" link @click="handleDelete(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>

      <el-pagination
        v-model:current-page="page"
        v-model:page-size="pageSize"
        :total="total"
        layout="total, sizes, prev, pager, next, jumper"
        class="pagination"
        @change="loadList"
      />
    </el-card>

    <!-- 用户表单弹框 -->
    <el-dialog v-model="dialogVisible" :title="isEdit ? '编辑用户' : '新建用户'" width="520px">
      <el-form ref="formRef" :model="currentUser" :rules="rules" label-width="80px">
        <el-form-item label="用户名" prop="username">
          <el-input v-model="currentUser.username" :disabled="isEdit" />
        </el-form-item>
        <el-form-item v-if="!isEdit" label="密码" prop="password">
          <el-input v-model="currentUser.password" type="password" show-password />
        </el-form-item>
        <el-form-item label="姓名">
          <el-input v-model="currentUser.realName" />
        </el-form-item>
        <el-form-item label="邮箱">
          <el-input v-model="currentUser.email" />
        </el-form-item>
        <el-form-item label="电话">
          <el-input v-model="currentUser.phone" />
        </el-form-item>
        <el-form-item label="状态">
          <el-switch v-model="currentUser.status" :active-value="1" :inactive-value="0" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="handleSave">保存</el-button>
      </template>
    </el-dialog>

    <!-- 角色分配弹框 -->
    <el-dialog v-model="roleDialogVisible" title="分配角色" width="500px">
      <el-checkbox-group v-model="assignedRoleIds">
        <el-checkbox
          v-for="role in allRoles"
          :key="role.id"
          :value="role.id"
          :label="role.roleName + '(' + role.roleCode + ')'"
        />
      </el-checkbox-group>
      <template #footer>
        <el-button @click="roleDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="handleAssignRoles">确定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.page-container {
  padding: 16px;
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.pagination {
  margin-top: 16px;
  display: flex;
  justify-content: flex-end;
}
</style>
