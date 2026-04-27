<script setup lang="ts">
import { ref, onMounted, reactive } from 'vue'
import { roleApi } from '@/api/role'
import { menuApi } from '@/api/menu'
import type { SystemRole, SystemMenu } from '@/types'
import { ElMessage, ElMessageBox } from 'element-plus'

const loading = ref(false)
const list = ref<SystemRole[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const searchForm = reactive({ roleName: '', status: undefined as number | undefined })

const dialogVisible = ref(false)
const isEdit = ref(false)
const currentRole = ref<Partial<SystemRole>>({})
const formRef = ref()

const menuDialogVisible = ref(false)
const menuTree = ref<SystemMenu[]>([])
const selectedMenuIds = ref<number[]>([])
const currentRoleId = ref<number>(0)
const menuTreeRef = ref()

async function loadList() {
  loading.value = true
  try {
    const res = await roleApi.list({ ...searchForm, page: page.value, pageSize: pageSize.value })
    const data = res.data.data
    list.value = data?.list || data || []
    total.value = data?.total ?? list.value.length
  } finally {
    loading.value = false
  }
}

function openCreate() {
  isEdit.value = false
  currentRole.value = { status: 1, dataScope: 1 }
  dialogVisible.value = true
}

function openEdit(row: SystemRole) {
  isEdit.value = true
  currentRole.value = { ...row }
  dialogVisible.value = true
}

async function openMenuAssign(row: SystemRole) {
  currentRoleId.value = row.id
  const [menuRes, assignedRes] = await Promise.all([
    menuApi.tree(),
    roleApi.getRoleMenus(row.id),
  ])
  menuTree.value = menuRes.data.data || []
  const assigned = assignedRes.data.data || []
  selectedMenuIds.value = assigned.map((m: SystemMenu) => m.id)
  menuDialogVisible.value = true
}

async function handleSave() {
  await formRef.value?.validate()
  try {
    if (isEdit.value) {
      await roleApi.update(currentRole.value.id!, currentRole.value)
    } else {
      await roleApi.create(currentRole.value)
    }
    ElMessage.success(isEdit.value ? '更新成功' : '创建成功')
    dialogVisible.value = false
    loadList()
  } catch {
    // handled
  }
}

async function handleDelete(row: SystemRole) {
  await ElMessageBox.confirm(`确定删除角色 "${row.roleName}" 吗？`, '警告', { type: 'warning' })
  await roleApi.delete(row.id)
  ElMessage.success('删除成功')
  loadList()
}

async function handleAssignMenus() {
  const keys = menuTreeRef.value?.getCheckedKeys(false) as number[]
  await roleApi.assignMenus(currentRoleId.value, keys)
  ElMessage.success('菜单分配成功')
  menuDialogVisible.value = false
}

onMounted(loadList)
</script>

<template>
  <div class="page-container">
    <el-card class="filter-card">
      <el-form :model="searchForm" inline>
        <el-form-item label="角色名">
          <el-input v-model="searchForm.roleName" placeholder="角色名" clearable />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="loadList">搜索</el-button>
          <el-button @click="Object.assign(searchForm, { roleName: '', status: undefined }); loadList()">重置</el-button>
        </el-form-item>
      </el-form>
    </el-card>

    <el-card>
      <template #header>
        <div class="card-header">
          <span>角色管理</span>
          <el-button type="primary" @click="openCreate">
            <el-icon><Plus /></el-icon>新建角色
          </el-button>
        </div>
      </template>

      <el-table v-loading="loading" :data="list" stripe>
        <el-table-column prop="id" label="ID" width="80" />
        <el-table-column prop="roleName" label="角色名称" width="160" />
        <el-table-column prop="roleCode" label="角色标识" width="160" />
        <el-table-column prop="description" label="描述" show-overflow-tooltip />
        <el-table-column label="数据范围" width="120">
          <template #default="{ row }">
            {{ ['', '全量', '本部门', '个人'][row.dataScope] || row.dataScope }}
          </template>
        </el-table-column>
        <el-table-column label="状态" width="80">
          <template #default="{ row }">
            <el-tag :type="row.status === 1 ? 'success' : 'danger'">
              {{ row.status === 1 ? '启用' : '禁用' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="200" fixed="right">
          <template #default="{ row }">
            <el-button type="primary" link @click="openEdit(row)">编辑</el-button>
            <el-button type="primary" link @click="openMenuAssign(row)">分配菜单</el-button>
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

    <!-- 角色表单弹框 -->
    <el-dialog v-model="dialogVisible" :title="isEdit ? '编辑角色' : '新建角色'" width="480px">
      <el-form ref="formRef" :model="currentRole" label-width="90px">
        <el-form-item label="角色名称" :rules="[{ required: true, message: '必填', trigger: 'blur' }]" prop="roleName">
          <el-input v-model="currentRole.roleName" />
        </el-form-item>
        <el-form-item label="角色标识" :rules="[{ required: true, message: '必填', trigger: 'blur' }]" prop="roleCode">
          <el-input v-model="currentRole.roleCode" :disabled="isEdit" />
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="currentRole.description" type="textarea" :rows="2" />
        </el-form-item>
        <el-form-item label="数据范围">
          <el-select v-model="currentRole.dataScope">
            <el-option label="全量" :value="1" />
            <el-option label="本部门" :value="2" />
            <el-option label="个人" :value="3" />
          </el-select>
        </el-form-item>
        <el-form-item label="状态">
          <el-switch v-model="currentRole.status" :active-value="1" :inactive-value="0" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="handleSave">保存</el-button>
      </template>
    </el-dialog>

    <!-- 菜单分配弹框 -->
    <el-dialog v-model="menuDialogVisible" title="分配菜单" width="420px">
      <el-tree
        ref="menuTreeRef"
        :data="menuTree"
        :props="{ label: 'menuName', children: 'children' }"
        node-key="id"
        :default-checked-keys="selectedMenuIds"
        show-checkbox
        check-strictly
      />
      <template #footer>
        <el-button @click="menuDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="handleAssignMenus">确定</el-button>
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
