<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { menuApi } from '@/api/menu'
import type { SystemMenu } from '@/types'
import { ElMessage, ElMessageBox } from 'element-plus'

const loading = ref(false)
const menuTree = ref<SystemMenu[]>([])
const dialogVisible = ref(false)
const isEdit = ref(false)
const currentMenu = ref<Partial<SystemMenu>>({})
const formRef = ref()

async function loadTree() {
  loading.value = true
  try {
    const res = await menuApi.tree()
    menuTree.value = res.data.data || []
  } finally {
    loading.value = false
  }
}

function openCreate(parentId?: number) {
  isEdit.value = false
  currentMenu.value = {
    parentId: parentId ?? 0,
    menuType: 'C',
    sort: 0,
    visible: 1,
    isFrame: 0,
  }
  dialogVisible.value = true
}

function openEdit(row: SystemMenu) {
  isEdit.value = true
  currentMenu.value = { ...row }
  dialogVisible.value = true
}

async function handleSave() {
  await formRef.value?.validate()
  try {
    if (isEdit.value) {
      await menuApi.update(currentMenu.value.id!, currentMenu.value)
    } else {
      await menuApi.create(currentMenu.value)
    }
    ElMessage.success(isEdit.value ? '更新成功' : '创建成功')
    dialogVisible.value = false
    loadTree()
  } catch {
    // handled
  }
}

async function handleDelete(row: SystemMenu) {
  await ElMessageBox.confirm(`确定删除菜单 "${row.menuName}" 吗？`, '警告', { type: 'warning' })
  await menuApi.delete(row.id)
  ElMessage.success('删除成功')
  loadTree()
}

onMounted(loadTree)
</script>

<template>
  <div class="page-container">
    <el-card>
      <template #header>
        <div class="card-header">
          <span>菜单管理</span>
          <el-button type="primary" @click="openCreate()">
            <el-icon><Plus /></el-icon>新建菜单
          </el-button>
        </div>
      </template>

      <el-table
        v-loading="loading"
        :data="menuTree"
        row-key="id"
        default-expand-all
        stripe
      >
        <el-table-column prop="menuName" label="菜单名称" min-width="200" />
        <el-table-column prop="menuType" label="类型" width="80">
          <template #default="{ row }">
            <el-tag size="small" :type="row.menuType === 'M' ? undefined : row.menuType === 'C' ? 'success' : 'info'">
              {{ row.menuType === 'M' ? '目录' : row.menuType === 'C' ? '菜单' : '按钮' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="icon" label="图标" width="80" />
        <el-table-column prop="path" label="路由路径" width="200" />
        <el-table-column prop="perms" label="权限标识" width="200" show-overflow-tooltip />
        <el-table-column prop="sort" label="排序" width="80" />
        <el-table-column label="显示" width="80">
          <template #default="{ row }">
            <el-tag :type="row.visible === 1 ? 'success' : 'info'" size="small">
              {{ row.visible === 1 ? '显示' : '隐藏' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="200" fixed="right">
          <template #default="{ row }">
            <el-button type="primary" link @click="openCreate(row.id)">新增子菜单</el-button>
            <el-button type="primary" link @click="openEdit(row)">编辑</el-button>
            <el-button type="danger" link @click="handleDelete(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <!-- 菜单表单弹框 -->
    <el-dialog v-model="dialogVisible" :title="isEdit ? '编辑菜单' : '新建菜单'" width="520px">
      <el-form ref="formRef" :model="currentMenu" label-width="90px">
        <el-form-item label="菜单名称" :rules="[{ required: true, message: '必填', trigger: 'blur' }]" prop="menuName">
          <el-input v-model="currentMenu.menuName" />
        </el-form-item>
        <el-form-item label="菜单类型" prop="menuType">
          <el-radio-group v-model="currentMenu.menuType">
            <el-radio value="M">目录</el-radio>
            <el-radio value="C">菜单</el-radio>
            <el-radio value="F">按钮</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="路由路径" prop="path">
          <el-input v-model="currentMenu.path" />
        </el-form-item>
        <el-form-item label="组件路径" prop="component">
          <el-input v-model="currentMenu.component" />
        </el-form-item>
        <el-form-item label="权限标识" prop="perms">
          <el-input v-model="currentMenu.perms" />
        </el-form-item>
        <el-form-item label="菜单图标" prop="icon">
          <el-input v-model="currentMenu.icon" />
        </el-form-item>
        <el-form-item label="显示排序" prop="sort">
          <el-input-number v-model="currentMenu.sort" :min="0" :max="999" />
        </el-form-item>
        <el-form-item label="显示状态" prop="visible">
          <el-switch v-model="currentMenu.visible" :active-value="1" :inactive-value="0" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="handleSave">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.page-container {
  padding: 16px;
}
.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
</style>
