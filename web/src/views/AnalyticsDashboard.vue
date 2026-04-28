<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { analyticsApi } from '@/api/analytics'
import type { WorkflowOverviewStats, WorkflowDailyStats, StatsTrendData, StatsQueryParams } from '@/types/analytics'
import { ElMessage } from 'element-plus'

const overview = ref<WorkflowOverviewStats | null>(null)
const dailyStats = ref<WorkflowDailyStats[]>([])
const trendData = ref<StatsTrendData | null>(null)

const dateRange = ref<[string, string]>(['', ''])
const loadingOverview = ref(false)
const loadingTable = ref(false)

async function loadOverview() {
  loadingOverview.value = true
  try {
    const res = await analyticsApi.getWorkflowOverview()
    overview.value = res.data.data
  } catch {
    ElMessage.error('加载总览数据失败')
  } finally {
    loadingOverview.value = false
  }
}

async function loadStats() {
  loadingTable.value = true
  try {
    const params: StatsQueryParams = {
      startDate: dateRange.value[0] || undefined,
      endDate: dateRange.value[1] || undefined,
    }
    const [effRes, trendRes] = await Promise.all([
      analyticsApi.getWorkflowEfficiency(params),
      analyticsApi.getInstanceTrend(params),
    ])
    dailyStats.value = effRes.data.data || []
    trendData.value = trendRes.data.data
  } catch {
    ElMessage.error('加载统计数据失败')
  } finally {
    loadingTable.value = false
  }
}

function msToMin(ms: number) {
  if (!ms) return '0'
  return (ms / 60000).toFixed(1)
}

function handleDateChange() {
  loadStats()
}

onMounted(() => {
  loadOverview()
  loadStats()
})
</script>

<template>
  <div class="page-container">
    <div class="stat-cards" v-loading="loadingOverview">
      <el-card class="stat-card">
        <el-statistic title="总实例数" :value="overview?.totalInstances ?? 0" />
      </el-card>
      <el-card class="stat-card">
        <el-statistic title="运行中" :value="overview?.runningInstances ?? 0" />
      </el-card>
      <el-card class="stat-card">
        <el-statistic title="已完成" :value="overview?.completedInstances ?? 0" />
      </el-card>
      <el-card class="stat-card">
        <el-statistic
          title="通过率"
          :value="overview ? +(overview.passRate * 100).toFixed(1) : 0"
          suffix="%"
        />
      </el-card>
    </div>

    <el-card>
      <template #header>
        <div class="card-header">
          <span>工作流效率分析</span>
          <el-date-picker
            v-model="dateRange"
            type="daterange"
            range-separator="至"
            start-placeholder="开始日期"
            end-placeholder="结束日期"
            value-format="YYYY-MM-DD"
            @change="handleDateChange"
          />
        </div>
      </template>

      <!-- 趋势数据表格 -->
      <template v-if="trendData && trendData.dates.length > 0">
        <div class="section-title">实例趋势</div>
        <el-table :data="trendData.dates.map((d, i) => ({ date: d, ...Object.fromEntries(trendData!.series.map(s => [s.name, s.data[i]])) }))" stripe size="small" style="margin-bottom:16px">
          <el-table-column prop="date" label="日期" width="120" />
          <el-table-column
            v-for="s in trendData.series"
            :key="s.name"
            :prop="s.name"
            :label="s.name"
          />
        </el-table>
      </template>

      <div class="section-title">日效率统计</div>
      <el-table v-loading="loadingTable" :data="dailyStats" stripe>
        <el-table-column prop="statDate" label="日期" width="120" />
        <el-table-column prop="totalInstances" label="总实例" width="90" />
        <el-table-column prop="completedInstances" label="已完成" width="90" />
        <el-table-column prop="failedInstances" label="失败" width="80" />
        <el-table-column label="平均耗时(分)" width="130">
          <template #default="{ row }">
            {{ msToMin(row.avgExecutionTimeMs) }}
          </template>
        </el-table-column>
        <el-table-column label="完成率" width="100">
          <template #default="{ row }">
            {{ row.totalInstances ? ((row.completedInstances / row.totalInstances) * 100).toFixed(1) : '0' }}%
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
.card-header { display: flex; justify-content: space-between; align-items: center; }
.section-title { font-weight: 600; margin-bottom: 8px; color: #303133; }
</style>
