<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { workflowApi } from '@/api/workflow'
import type { WorkflowDef } from '@/types'
import { ElMessage, ElMessageBox } from 'element-plus'

type TagType = 'primary' | 'success' | 'warning' | 'info' | 'danger'

const router = useRouter()
const loading = ref(false)
const list = ref<WorkflowDef[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)

const searchForm = ref({ name: '' })

async function loadList() {
  loading.value = true
  try {
    const res = await workflowApi.list()
    const data = res.data.data
    list.value = Array.isArray(data) ? data : data?.list || []
    total.value = data?.total ?? list.value.length
  } catch {
    // error handled by interceptor
  } finally {
    loading.value = false
  }
}

function goDesigner(id?: string) {
  router.push(`/workflow/designer/${id || ''}`)
}

async function handleDelete(row: WorkflowDef) {
  await ElMessageBox.confirm(`确定删除流程 "${row.name}" 吗？`, '警告', {
    confirmButtonText: '删除',
    cancelButtonText: '取消',
    type: 'warning',
  })
  // TODO: 调用删除接口后刷新
  ElMessage.warning('删除功能待后端接口对接')
}

function statusTag(status: string): TagType {
  const map: Record<string, TagType> = {
    draft: 'info',
    published: 'success',
    disabled: 'danger',
  }
  return map[status] || 'info'
}

function statusText(status: string) {
  const map: Record<string, string> = {
    draft: '草稿',
    published: '已发布',
    disabled: '已禁用',
  }
  return map[status] || status
}

onMounted(loadList)
</script>

<template>
  <div class="page-container">
    <el-card class="filter-card">
      <el-form :model="searchForm" inline>
        <el-form-item label="流程名称">
          <el-input v-model="searchForm.name" placeholder="请输入流程名称" clearable />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="loadList">搜索</el-button>
          <el-button @click="searchForm.name = ''; loadList()">重置</el-button>
        </el-form-item>
      </el-form>
    </el-card>

    <el-card>
      <template #header>
        <div class="card-header">
          <span>流程定义列表</span>
          <el-button type="primary" @click="goDesigner()">
            <el-icon><Plus /></el-icon>新建流程
          </el-button>
        </div>
      </template>

      <el-table v-loading="loading" :data="list" stripe>
        <el-table-column prop="id" label="流程ID" width="200" />
        <el-table-column prop="name" label="流程名称" />
        <el-table-column prop="description" label="描述" show-overflow-tooltip />
        <el-table-column prop="version" label="版本" width="80" />
        <el-table-column label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="statusTag(row.status)">{{ statusText(row.status) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="createdBy" label="创建人" width="120" />
        <el-table-column prop="createdAt" label="创建时间" width="180" />
        <el-table-column label="操作" width="200" fixed="right">
          <template #default="{ row }">
            <el-button type="primary" link @click="goDesigner(row.id)">设计</el-button>
            <el-button type="primary" link>查看版本</el-button>
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
  </div>
</template>

<style scoped>
.page-container {
  padding: 16px;
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.filter-card {
  margin-bottom: 0;
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
