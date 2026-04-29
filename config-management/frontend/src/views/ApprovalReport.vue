<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { analyticsApi } from '@/api/analytics'
import type { ApprovalPerformanceStats, StatsQueryParams } from '@/types/analytics'
import { ElMessage } from 'element-plus'

const loading = ref(false)
const list = ref<ApprovalPerformanceStats[]>([])
const dateRange = ref<[string, string]>(['', ''])

const totalApprovals = ref(0)
const totalApproved = ref(0)
const totalRejected = ref(0)
const avgTimeMin = ref(0)

async function loadData() {
  loading.value = true
  try {
    const params: StatsQueryParams = {
      startDate: dateRange.value[0] || undefined,
      endDate: dateRange.value[1] || undefined,
    }
    const res = await analyticsApi.getApprovalPerformance(params)
    list.value = res.data.data || []
    calcSummary()
  } catch {
    ElMessage.error('加载审批绩效数据失败')
  } finally {
    loading.value = false
  }
}

function calcSummary() {
  totalApprovals.value = list.value.reduce((s, r) => s + r.totalApprovals, 0)
  totalApproved.value = list.value.reduce((s, r) => s + r.approvedCount, 0)
  totalRejected.value = list.value.reduce((s, r) => s + r.rejectedCount, 0)
  const totalMs = list.value.reduce((s, r) => s + r.avgApprovalTimeMs, 0)
  avgTimeMin.value = list.value.length > 0 ? +(totalMs / list.value.length / 60000).toFixed(1) : 0
}

function msToMin(ms: number) {
  return (ms / 60000).toFixed(1)
}

onMounted(loadData)
</script>

<template>
  <div class="page-container">
    <el-card class="filter-card">
      <el-form inline>
        <el-form-item label="日期范围">
          <el-date-picker
            v-model="dateRange"
            type="daterange"
            range-separator="至"
            start-placeholder="开始日期"
            end-placeholder="结束日期"
            value-format="YYYY-MM-DD"
          />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="loadData">查询</el-button>
          <el-button @click="dateRange = ['', '']; loadData()">重置</el-button>
        </el-form-item>
      </el-form>
    </el-card>

    <div class="stat-cards">
      <el-card class="stat-card">
        <el-statistic title="总审批数" :value="totalApprovals" />
      </el-card>
      <el-card class="stat-card">
        <el-statistic title="已审批" :value="totalApproved" />
      </el-card>
      <el-card class="stat-card">
        <el-statistic title="已拒绝" :value="totalRejected" />
      </el-card>
      <el-card class="stat-card">
        <el-statistic title="平均审批时长(分)" :value="avgTimeMin" />
      </el-card>
    </div>

    <el-card>
      <template #header><span>审批绩效明细</span></template>
      <el-table v-loading="loading" :data="list" stripe>
        <el-table-column prop="statDate" label="日期" width="120" />
        <el-table-column prop="approverUserId" label="审批人" width="160" />
        <el-table-column prop="totalApprovals" label="总审批数" width="100" />
        <el-table-column prop="approvedCount" label="通过" width="80" />
        <el-table-column prop="rejectedCount" label="拒绝" width="80" />
        <el-table-column label="平均时长(分)" width="130">
          <template #default="{ row }">{{ msToMin(row.avgApprovalTimeMs) }}</template>
        </el-table-column>
        <el-table-column label="通过率" width="100">
          <template #default="{ row }">
            {{ row.totalApprovals ? ((row.approvedCount / row.totalApprovals) * 100).toFixed(1) : '0' }}%
          </template>
        </el-table-column>
      </el-table>
    </el-card>
  </div>
</template>

<style scoped>
.page-container { padding: 16px; display: flex; flex-direction: column; gap: 16px; }
.stat-cards { display: grid; grid-template-columns: repeat(4, 1fr); gap: 16px; }
.stat-card { text-align: center; }
</style>
