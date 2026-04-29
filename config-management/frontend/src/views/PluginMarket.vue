<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { pluginApi } from '@/api/plugin'
import type { PluginInfo } from '@/types/plugin'
import { ElMessage, ElMessageBox } from 'element-plus'

const loading = ref(false)
const list = ref<PluginInfo[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(12)
const keyword = ref('')

async function loadList() {
  loading.value = true
  try {
    const res = await pluginApi.listMarket({ page: page.value, pageSize: pageSize.value })
    const data = res.data.data
    list.value = data?.list || []
    total.value = data?.total ?? 0
  } catch {
    ElMessage.error('加载插件市场失败')
  } finally {
    loading.value = false
  }
}

const filteredList = ref<PluginInfo[]>([])

async function handleSearch() {
  await loadList()
  if (keyword.value) {
    filteredList.value = list.value.filter(p =>
      p.pluginName.includes(keyword.value) || (p.description || '').includes(keyword.value)
    )
  } else {
    filteredList.value = list.value
  }
}

async function handleInstall(plugin: PluginInfo) {
  await ElMessageBox.confirm(
    `确定安装插件 "${plugin.pluginName}" v${plugin.version} 吗？`,
    '确认安装',
    { confirmButtonText: '安装', cancelButtonText: '取消', type: 'info' }
  )
  try {
    await pluginApi.install(plugin.pluginId, { version: plugin.version })
    ElMessage.success(`插件 ${plugin.pluginName} 安装成功`)
  } catch {
    ElMessage.error('安装失败')
  }
}

function renderStars(rating: number) {
  const full = Math.floor(rating)
  return '★'.repeat(full) + '☆'.repeat(5 - full)
}

function formatDownload(n: number) {
  if (n >= 1000) return `${(n / 1000).toFixed(1)}k`
  return String(n)
}

onMounted(async () => {
  await loadList()
  filteredList.value = list.value
})
</script>

<template>
  <div class="page-container">
    <el-card class="filter-card">
      <div class="filter-row">
        <span class="page-title">插件市场</span>
        <el-input
          v-model="keyword"
          placeholder="搜索插件"
          clearable
          style="width:260px"
          @keyup.enter="handleSearch"
          @clear="handleSearch"
        >
          <template #append>
            <el-button @click="handleSearch">搜索</el-button>
          </template>
        </el-input>
      </div>
    </el-card>

    <div v-loading="loading" class="plugin-grid">
      <el-card
        v-for="plugin in filteredList"
        :key="plugin.pluginId"
        class="plugin-card"
        shadow="hover"
      >
        <div class="plugin-header">
          <span class="plugin-name">{{ plugin.pluginName }}</span>
          <el-tag v-if="plugin.isOfficial" type="warning" size="small">官方</el-tag>
        </div>
        <div class="plugin-desc">{{ plugin.description || '暂无描述' }}</div>
        <div class="plugin-meta">
          <span>作者：{{ plugin.author || '-' }}</span>
          <span>v{{ plugin.version }}</span>
        </div>
        <div class="plugin-stats">
          <span class="stars">{{ renderStars(plugin.rating) }}</span>
          <span class="downloads">⬇ {{ formatDownload(plugin.downloadCount) }}</span>
        </div>
        <el-button type="primary" size="small" style="width:100%;margin-top:10px" @click="handleInstall(plugin)">
          安装
        </el-button>
      </el-card>
    </div>

    <el-empty v-if="!loading && filteredList.length === 0" description="暂无插件" />

    <el-pagination
      v-model:current-page="page"
      v-model:page-size="pageSize"
      :total="total"
      layout="total, prev, pager, next"
      class="pagination"
      @change="loadList"
    />
  </div>
</template>

<style scoped>
.page-container { padding: 16px; display: flex; flex-direction: column; gap: 16px; }
.filter-row { display: flex; justify-content: space-between; align-items: center; }
.page-title { font-size: 18px; font-weight: 600; }
.plugin-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(220px, 1fr)); gap: 16px; }
.plugin-card { cursor: default; }
.plugin-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 6px; }
.plugin-name { font-weight: 600; font-size: 15px; }
.plugin-desc { color: #606266; font-size: 13px; min-height: 40px; margin-bottom: 8px; }
.plugin-meta { display: flex; justify-content: space-between; font-size: 12px; color: #909399; margin-bottom: 4px; }
.plugin-stats { display: flex; justify-content: space-between; font-size: 13px; }
.stars { color: #f0a020; }
.downloads { color: #909399; }
.pagination { display: flex; justify-content: flex-end; }
</style>
