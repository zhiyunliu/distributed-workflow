<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { workflowApi } from '@/api/workflow'
import { instanceApi } from '@/api/instance'
import type { WorkflowInstance, AuditLog } from '@/types'

const route = useRoute()
const router = useRouter()
const instanceId = route.params.id as string

const loading = ref(false)
const instance = ref<WorkflowInstance | null>(null)
const events = ref<AuditLog[]>([])
const activeTab = ref('basic')

async function loadData() {
  loading.value = true
  try {
    const [instRes, eventRes] = await Promise.all([
      instanceApi.get(instanceId),
      workflowApi.getInstanceEvents(instanceId),
    ])
    instance.value = instRes.data.data
    events.value = eventRes.data.data || []
  } catch {
    // handled
  } finally {
    loading.value = false
  }
}

async function handleCancel() {
  if (!instance.value) return
  await instanceApi.cancel(instanceId, '手动取消')
  await loadData()
}

function statusType(status: string) {
  const map: Record<string, string> = {
    running: 'primary',
    completed: 'success',
    failed: 'danger',
    cancelled: 'info',
    paused: 'warning',
  }
  return map[status] || 'info'
}

onMounted(loadData)
</script>

<template>
  <div class="page-container" v-loading="loading">
    <div class="page-header">
      <el-button @click="router.back()">
        <el-icon><ArrowLeft /></el-icon>返回
      </el-button>
      <span class="title">实例详情</span>
      <div class="actions">
        <el-button
          v-if="instance?.status === 'running'"
          type="danger"
          @click="handleCancel"
        >
          取消实例
        </el-button>
      </div>
    </div>

    <el-tabs v-model="activeTab">
      <el-tab-pane label="基本信息" name="basic">
        <el-card v-if="instance">
          <el-descriptions :column="2" border>
            <el-descriptions-item label="实例ID">{{ instance.id }}</el-descriptions-item>
            <el-descriptions-item label="流程ID">{{ instance.workflowId }}</el-descriptions-item>
            <el-descriptions-item label="版本">{{ instance.workflowVersion }}</el-descriptions-item>
            <el-descriptions-item label="状态">
              <el-tag :type="statusType(instance.status)">{{ instance.status }}</el-tag>
            </el-descriptions-item>
            <el-descriptions-item label="创建人">{{ instance.createdBy }}</el-descriptions-item>
            <el-descriptions-item label="开始时间">{{ instance.startTime }}</el-descriptions-item>
            <el-descriptions-item label="结束时间">{{ instance.endTime || '-' }}</el-descriptions-item>
            <el-descriptions-item label="错误信息" :span="2">
              {{ instance.errorMessage || '-' }}
            </el-descriptions-item>
            <el-descriptions-item label="输入数据" :span="2">
              <pre class="json-pre">{{ JSON.stringify(instance.inputData, null, 2) }}</pre>
            </el-descriptions-item>
            <el-descriptions-item label="输出数据" :span="2">
              <pre class="json-pre">{{ JSON.stringify(instance.outputData, null, 2) }}</pre>
            </el-descriptions-item>
          </el-descriptions>
        </el-card>
      </el-tab-pane>

      <el-tab-pane label="执行事件" name="events">
        <el-card>
          <el-timeline>
            <el-timeline-item
              v-for="event in events"
              :key="event.id"
              :timestamp="event.operateTime"
              placement="top"
            >
              <el-card class="event-card">
                <p><strong>操作类型：</strong>{{ event.operationType }}</p>
                <p><strong>操作人：</strong>{{ event.operator }}</p>
                <p v-if="event.detail"><strong>详情：</strong>{{ event.detail }}</p>
              </el-card>
            </el-timeline-item>
            <el-empty v-if="events.length === 0" description="暂无事件记录" />
          </el-timeline>
        </el-card>
      </el-tab-pane>
    </el-tabs>
  </div>
</template>

<style scoped>
.page-container {
  padding: 16px;
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.page-header {
  display: flex;
  align-items: center;
  gap: 12px;
}
.title {
  font-size: 18px;
  font-weight: 600;
}
.actions {
  margin-left: auto;
}
.json-pre {
  font-size: 12px;
  background: #f5f7fa;
  padding: 8px;
  border-radius: 4px;
  overflow: auto;
  max-height: 200px;
  white-space: pre-wrap;
}
.event-card {
  padding: 8px;
}
.event-card p {
  margin: 4px 0;
  font-size: 13px;
}
</style>
