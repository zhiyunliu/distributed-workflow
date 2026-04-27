<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { endpointApi } from '@/api/endpoint'
import { ElMessage } from 'element-plus'
import type { WorkflowEndpoint } from '@/types'

const loading = ref(false)
const list = ref<WorkflowEndpoint[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const cronExpr = ref('')
const cronNextTimes = ref<string[]>([])
const validatingCron = ref(false)

async function loadList() {
  loading.value = true
  try {
    const res = await endpointApi.list({ page: page.value, pageSize: pageSize.value })
    const data = res.data.data
    list.value = data?.list || data || []
    total.value = data?.total ?? list.value.length
  } catch {
    // handled
  } finally {
    loading.value = false
  }
}

async function toggleEndpoint(row: WorkflowEndpoint) {
  try {
    if (row.disabled) {
      await endpointApi.enable(row.id)
      ElMessage.success('已启用')
    } else {
      await endpointApi.disable(row.id)
      ElMessage.success('已禁用')
    }
    await loadList()
  } catch {
    // handled
  }
}

async function testEndpoint(row: WorkflowEndpoint) {
  try {
    const res = await endpointApi.test(row.id)
    ElMessage.success(`触发成功，实例ID: ${res.data.data?.instanceId}`)
  } catch {
    // handled
  }
}

async function validateCron() {
  if (!cronExpr.value) {
    ElMessage.warning('请输入 Cron 表达式')
    return
  }
  validatingCron.value = true
  try {
    const res = await endpointApi.validateCron(cronExpr.value)
    const data = res.data.data
    if (data.valid) {
      cronNextTimes.value = data.nextTimes || []
      ElMessage.success('Cron 表达式有效')
    } else {
      cronNextTimes.value = []
      ElMessage.error('无效的 Cron 表达式: ' + data.message)
    }
  } catch {
    // handled
  } finally {
    validatingCron.value = false
  }
}

onMounted(loadList)
</script>

<template>
  <div class="page-container">
    <!-- Cron 验证工具 -->
    <el-card class="cron-card">
      <template #header>Cron 表达式验证</template>
      <el-form inline>
        <el-form-item label="Cron 表达式">
          <el-input
            v-model="cronExpr"
            placeholder="如：0 */5 * * *"
            style="width: 260px"
          />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" :loading="validatingCron" @click="validateCron">
            验证
          </el-button>
        </el-form-item>
      </el-form>
      <div v-if="cronNextTimes.length" class="next-times">
        <span class="next-title">未来5次执行时间：</span>
        <el-tag
          v-for="(t, idx) in cronNextTimes"
          :key="idx"
          style="margin: 0 4px 4px 0"
        >
          {{ t }}
        </el-tag>
      </div>
    </el-card>

    <!-- 端点列表 -->
    <el-card>
      <template #header>
        <div class="card-header">
          <span>端点管理</span>
          <el-button @click="loadList"><el-icon><Refresh /></el-icon>刷新</el-button>
        </div>
      </template>

      <el-table v-loading="loading" :data="list" stripe>
        <el-table-column prop="id" label="端点ID" width="200" show-overflow-tooltip />
        <el-table-column prop="name" label="名称" />
        <el-table-column prop="type" label="类型" width="120" />
        <el-table-column prop="workflowId" label="流程ID" width="180" show-overflow-tooltip />
        <el-table-column prop="triggerCount" label="触发次数" width="100" />
        <el-table-column label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="row.disabled ? 'danger' : 'success'">
              {{ row.disabled ? '已禁用' : '启用中' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="createTime" label="创建时间" width="180" />
        <el-table-column label="操作" width="200" fixed="right">
          <template #default="{ row }">
            <el-button type="primary" link @click="testEndpoint(row)">测试</el-button>
            <el-button
              :type="row.disabled ? 'success' : 'warning'"
              link
              @click="toggleEndpoint(row)"
            >
              {{ row.disabled ? '启用' : '禁用' }}
            </el-button>
          </template>
        </el-table-column>
      </el-table>

      <el-pagination
        v-model:current-page="page"
        v-model:page-size="pageSize"
        :total="total"
        layout="total, prev, pager, next"
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
.cron-card .next-times {
  margin-top: 8px;
  display: flex;
  align-items: center;
  flex-wrap: wrap;
}
.next-title {
  font-size: 13px;
  color: #606266;
  margin-right: 8px;
}
.pagination {
  margin-top: 16px;
  display: flex;
  justify-content: flex-end;
}
</style>
