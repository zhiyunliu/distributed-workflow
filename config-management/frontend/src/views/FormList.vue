<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { formApi } from '@/api/form'
import { FormStatus } from '@/types/form'
import type { FormDefinition, FormListParams } from '@/types/form'
import { ElMessage, ElMessageBox } from 'element-plus'

const router = useRouter()
const loading = ref(false)
const list = ref<FormDefinition[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const keyword = ref('')

const dialogVisible = ref(false)
const dialogTitle = ref('新建表单')
const editForm = ref<Partial<FormDefinition>>({ formName: '', description: '' })
const editingId = ref('')

const versionsVisible = ref(false)
const versionList = ref<{ version: number; publishedBy: string; publishedAt: string }[]>([])

async function loadList() {
  loading.value = true
  try {
    const params: FormListParams = { page: page.value, pageSize: pageSize.value, keyword: keyword.value || undefined }
    const res = await formApi.list(params)
    const data = res.data.data
    list.value = data?.list || []
    total.value = data?.total ?? 0
  } catch {
    ElMessage.error('加载表单列表失败')
  } finally {
    loading.value = false
  }
}

function openCreate() {
  editingId.value = ''
  dialogTitle.value = '新建表单'
  editForm.value = { formName: '', description: '' }
  dialogVisible.value = true
}

function openEdit(row: FormDefinition) {
  editingId.value = row.formId
  dialogTitle.value = '编辑表单'
  editForm.value = { formName: row.formName, description: row.description }
  dialogVisible.value = true
}

async function submitDialog() {
  try {
    if (editingId.value) {
      await formApi.update(editingId.value, editForm.value)
      ElMessage.success('保存成功')
    } else {
      await formApi.create(editForm.value)
      ElMessage.success('创建成功')
    }
    dialogVisible.value = false
    loadList()
  } catch {
    ElMessage.error('操作失败')
  }
}

async function handlePublish(row: FormDefinition) {
  await ElMessageBox.confirm(`确定发布表单 "${row.formName}" 吗？`, '确认发布', {
    confirmButtonText: '发布',
    cancelButtonText: '取消',
    type: 'warning',
  })
  try {
    await formApi.publish(row.formId)
    ElMessage.success('发布成功')
    loadList()
  } catch {
    ElMessage.error('发布失败')
  }
}

async function handleDisable(row: FormDefinition) {
  await ElMessageBox.confirm(`确定停用表单 "${row.formName}" 吗？`, '确认停用', {
    confirmButtonText: '停用',
    cancelButtonText: '取消',
    type: 'warning',
  })
  try {
    await formApi.update(row.formId, { status: FormStatus.Disabled })
    ElMessage.success('停用成功')
    loadList()
  } catch {
    ElMessage.error('停用失败')
  }
}

async function viewVersions(row: FormDefinition) {
  try {
    const res = await formApi.getVersions(row.formId)
    versionList.value = res.data.data || []
    versionsVisible.value = true
  } catch {
    ElMessage.error('获取版本历史失败')
  }
}

function goDesigner(row: FormDefinition) {
  router.push(`/form/designer/${row.formId}`)
}

function statusTag(status: number) {
  const map: Record<number, string> = { 0: 'info', 1: 'success', 2: 'danger' }
  return map[status] || 'info'
}
function statusText(status: number) {
  const map: Record<number, string> = { 0: '草稿', 1: '已发布', 2: '已停用' }
  return map[status] || '未知'
}

onMounted(loadList)
</script>

<template>
  <div class="page-container">
    <el-card class="filter-card">
      <el-form inline>
        <el-form-item label="关键词">
          <el-input v-model="keyword" placeholder="请输入表单名称" clearable @keyup.enter="loadList" />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="loadList">搜索</el-button>
          <el-button @click="keyword = ''; loadList()">重置</el-button>
        </el-form-item>
      </el-form>
    </el-card>

    <el-card>
      <template #header>
        <div class="card-header">
          <span>表单管理</span>
          <el-button type="primary" @click="openCreate">新建表单</el-button>
        </div>
      </template>

      <el-table v-loading="loading" :data="list" stripe>
        <el-table-column prop="formId" label="表单ID" width="180" show-overflow-tooltip />
        <el-table-column prop="formName" label="表单名称" />
        <el-table-column prop="version" label="版本" width="80" />
        <el-table-column label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="statusTag(row.status)">{{ statusText(row.status) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="createdBy" label="创建人" width="120" />
        <el-table-column prop="createdAt" label="创建时间" width="180" />
        <el-table-column label="操作" width="240" fixed="right">
          <template #default="{ row }">
            <el-button type="primary" link @click="goDesigner(row)">设计</el-button>
            <el-button v-if="row.status === FormStatus.Draft" type="primary" link @click="openEdit(row)">编辑</el-button>
            <el-button v-if="row.status === FormStatus.Draft" type="success" link @click="handlePublish(row)">发布</el-button>
            <el-button type="info" link @click="viewVersions(row)">版本历史</el-button>
            <el-button v-if="row.status === FormStatus.Published" type="danger" link @click="handleDisable(row)">停用</el-button>
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

    <el-dialog v-model="dialogVisible" :title="dialogTitle" width="480px">
      <el-form :model="editForm" label-width="90px">
        <el-form-item label="表单名称" required>
          <el-input v-model="editForm.formName" placeholder="请输入表单名称" />
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="editForm.description" type="textarea" :rows="3" placeholder="请输入描述" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="submitDialog">确定</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="versionsVisible" title="版本历史" width="600px">
      <el-table :data="versionList">
        <el-table-column prop="version" label="版本号" width="100" />
        <el-table-column prop="publishedBy" label="发布人" width="120" />
        <el-table-column prop="publishedAt" label="发布时间" />
      </el-table>
    </el-dialog>
  </div>
</template>

<style scoped>
.page-container { padding: 16px; display: flex; flex-direction: column; gap: 16px; }
.card-header { display: flex; justify-content: space-between; align-items: center; }
.pagination { margin-top: 16px; display: flex; justify-content: flex-end; }
</style>
