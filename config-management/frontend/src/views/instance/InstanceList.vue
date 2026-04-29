<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { instanceApi } from '@/api/instance'
import type { WorkflowInstance } from '@/types'

type TagType = 'primary' | 'success' | 'warning' | 'info' | 'danger'

const router = useRouter()
const loading = ref(false)
const list = ref<WorkflowInstance[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)

const searchForm = ref({
  workflowId: '',
  status: '',
})

const statusOptions = [
  { label: '全部', value: '' },
  { label: '运行中', value: 'running' },
  { label: '已完成', value: 'completed' },
  { label: '已失败', value: 'failed' },
  { label: '已取消', value: 'cancelled' },
  { label: '已暂停', value: 'paused' },
]

async function loadList() {
  loading.value = true
  try {
    const res = await instanceApi.list({
      workflowId: searchForm.value.workflowId || undefined,
      status: searchForm.value.status || undefined,
      page: page.value,
      pageSize: pageSize.value,
    })
    const data = res.data.data
    list.value = data?.list || data || []
    total.value = data?.total ?? list.value.length
  } catch {
    // handled
  } finally {
    loading.value = false
  }
}

function statusType(status: string): TagType {
  const map: Record<string, TagType> = {
    running: 'primary',
    completed: 'success',
    failed: 'danger',
    cancelled: 'info',
    paused: 'warning',
  }
  return map[status] || 'info'
}

function statusText(status: string) {
  const map: Record<string, string> = {
    running: '运行中',
    completed: '已完成',
    failed: '已失败',
    cancelled: '已取消',
    paused: '已暂停',
  }
  return map[status] || status
}

onMounted(loadList)
</script>

<template>
  <div class="page-container">
    <el-card class="filter-card">
      <el-form :model="searchForm" inline>
        <el-form-item label="流程ID">
          <el-input v-model="searchForm.workflowId" placeholder="流程ID" clearable />
        </el-form-item>
        <el-form-item label="状态">
          <el-select v-model="searchForm.status" style="width: 140px">
            <el-option
              v-for="opt in statusOptions"
              :key="opt.value"
              :label="opt.label"
              :value="opt.value"
            />
          </el-select>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="loadList">搜索</el-button>
          <el-button @click="searchForm = { workflowId: '', status: '' }; loadList()">重置</el-button>
        </el-form-item>
      </el-form>
    </el-card>

    <el-card>
      <template #header>
        <div class="card-header">
          <span>实例列表</span>
          <el-button @click="loadList">
            <el-icon><Refresh /></el-icon>刷新
          </el-button>
        </div>
      </template>

      <el-table v-loading="loading" :data="list" stripe>
        <el-table-column prop="id" label="实例ID" width="200" show-overflow-tooltip />
        <el-table-column prop="workflowId" label="流程ID" width="180" show-overflow-tooltip />
        <el-table-column prop="workflowVersion" label="版本" width="70" />
        <el-table-column label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="statusType(row.status)">{{ statusText(row.status) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="createdBy" label="创建人" width="100" />
        <el-table-column prop="startTime" label="开始时间" width="180" />
        <el-table-column prop="endTime" label="结束时间" width="180" />
        <el-table-column prop="errorMessage" label="错误信息" show-overflow-tooltip />
        <el-table-column label="操作" width="160" fixed="right">
          <template #default="{ row }">
            <el-button type="primary" link @click="router.push(`/instance/${row.id}`)">
              详情
            </el-button>
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
