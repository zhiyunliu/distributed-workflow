<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { auditApi } from '@/api/audit'
import type { AuditLog } from '@/types'

const loading = ref(false)
const list = ref<AuditLog[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)

const searchForm = ref({
  instanceId: '',
  operator: '',
  startTime: '',
  endTime: '',
})

const dateRange = ref<[string, string] | null>(null)

async function loadList() {
  loading.value = true
  try {
    const [startTime, endTime] = dateRange.value || ['', '']
    const res = await auditApi.list({
      ...searchForm.value,
      startTime: startTime || undefined,
      endTime: endTime || undefined,
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

onMounted(loadList)
</script>

<template>
  <div class="page-container">
    <el-card class="filter-card">
      <el-form :model="searchForm" inline>
        <el-form-item label="实例ID">
          <el-input v-model="searchForm.instanceId" placeholder="实例ID" clearable />
        </el-form-item>
        <el-form-item label="操作人">
          <el-input v-model="searchForm.operator" placeholder="操作人" clearable />
        </el-form-item>
        <el-form-item label="时间范围">
          <el-date-picker
            v-model="dateRange"
            type="daterange"
            start-placeholder="开始时间"
            end-placeholder="结束时间"
            value-format="YYYY-MM-DDTHH:mm:ssZ"
          />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="loadList">搜索</el-button>
          <el-button @click="searchForm = { instanceId: '', operator: '', startTime: '', endTime: '' }; dateRange = null; loadList()">
            重置
          </el-button>
        </el-form-item>
      </el-form>
    </el-card>

    <el-card>
      <template #header>审计日志</template>
      <el-table v-loading="loading" :data="list" stripe>
        <el-table-column prop="id" label="日志ID" width="200" show-overflow-tooltip />
        <el-table-column prop="instanceId" label="实例ID" width="200" show-overflow-tooltip />
        <el-table-column prop="operationType" label="操作类型" width="140" />
        <el-table-column prop="operator" label="操作人" width="120" />
        <el-table-column prop="operateIP" label="操作IP" width="140" />
        <el-table-column prop="detail" label="详情" show-overflow-tooltip />
        <el-table-column prop="operateTime" label="操作时间" width="180" />
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
.pagination {
  margin-top: 16px;
  display: flex;
  justify-content: flex-end;
}
</style>
