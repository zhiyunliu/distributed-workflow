<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { pluginApi } from '@/api/plugin'
import type { PluginConfig } from '@/types/plugin'
import { ElMessage, ElMessageBox } from 'element-plus'

const loading = ref(false)
const list = ref<PluginConfig[]>([])

const configVisible = ref(false)
const configPluginId = ref('')
const configJson = ref('')

async function loadList() {
  loading.value = true
  try {
    const res = await pluginApi.listInstalled()
    list.value = res.data.data || []
  } catch {
    ElMessage.error('加载已安装插件失败')
  } finally {
    loading.value = false
  }
}

async function handleEnable(row: PluginConfig) {
  try {
    await pluginApi.enable(row.pluginId)
    ElMessage.success('启用成功')
    loadList()
  } catch {
    ElMessage.error('启用失败')
  }
}

async function handleDisable(row: PluginConfig) {
  try {
    await pluginApi.disable(row.pluginId)
    ElMessage.success('停用成功')
    loadList()
  } catch {
    ElMessage.error('停用失败')
  }
}

async function handleUninstall(row: PluginConfig) {
  await ElMessageBox.confirm(`确定卸载插件 "${row.pluginId}" 吗？`, '确认卸载', {
    confirmButtonText: '卸载',
    cancelButtonText: '取消',
    type: 'warning',
  })
  try {
    await pluginApi.uninstall(row.pluginId)
    ElMessage.success('卸载成功')
    loadList()
  } catch {
    ElMessage.error('卸载失败')
  }
}

async function openConfig(row: PluginConfig) {
  configPluginId.value = row.pluginId
  try {
    const res = await pluginApi.getConfig(row.pluginId)
    configJson.value = res.data.data?.config || '{}'
  } catch {
    configJson.value = row.config || '{}'
  }
  configVisible.value = true
}

async function saveConfig() {
  try {
    JSON.parse(configJson.value)
  } catch {
    ElMessage.error('配置格式无效，请输入合法的JSON')
    return
  }
  try {
    await pluginApi.saveConfig(configPluginId.value, { config: configJson.value })
    ElMessage.success('配置保存成功')
    configVisible.value = false
  } catch {
    ElMessage.error('保存配置失败')
  }
}

function statusTag(status: number) {
  return status === 1 ? 'success' : 'info'
}
function statusText(status: number) {
  return status === 1 ? '运行中' : '已停用'
}

onMounted(loadList)
</script>

<template>
  <div class="page-container">
    <el-card>
      <template #header>
        <div class="card-header">
          <span>已安装插件</span>
          <el-button @click="loadList">刷新</el-button>
        </div>
      </template>

      <el-table v-loading="loading" :data="list" stripe>
        <el-table-column prop="pluginId" label="插件名" show-overflow-tooltip />
        <el-table-column prop="pluginVersion" label="版本" width="100" />
        <el-table-column label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="statusTag(row.status)">{{ statusText(row.status) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="installedAt" label="安装时间" width="180" />
        <el-table-column label="操作" width="240" fixed="right">
          <template #default="{ row }">
            <el-button
              v-if="row.status !== 1"
              type="success"
              link
              @click="handleEnable(row)"
            >启用</el-button>
            <el-button
              v-else
              type="warning"
              link
              @click="handleDisable(row)"
            >停用</el-button>
            <el-button type="primary" link @click="openConfig(row)">配置</el-button>
            <el-button type="danger" link @click="handleUninstall(row)">卸载</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-dialog v-model="configVisible" title="插件配置" width="520px">
      <div class="config-hint">请输入 JSON 格式的配置</div>
      <el-input
        v-model="configJson"
        type="textarea"
        :rows="10"
        placeholder="{}"
        style="font-family: monospace"
      />
      <template #footer>
        <el-button @click="configVisible = false">取消</el-button>
        <el-button type="primary" @click="saveConfig">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.page-container { padding: 16px; }
.card-header { display: flex; justify-content: space-between; align-items: center; }
.config-hint { font-size: 12px; color: #909399; margin-bottom: 8px; }
</style>
