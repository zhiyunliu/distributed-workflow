<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { templateApi } from '@/api/template'
import type {
  WorkflowTemplate,
  WorkflowTemplateCategory,
  WorkflowTemplateInstallResult,
} from '@/types'

const loading = ref(false)
const detailLoading = ref(false)
const installLoadingId = ref('')
const list = ref<WorkflowTemplate[]>([])
const categories = ref<WorkflowTemplateCategory[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(12)
const detailVisible = ref(false)
const currentTemplate = ref<WorkflowTemplate | null>(null)

const queryForm = ref({
  keyword: '',
  categoryId: undefined as number | undefined,
})

const workflowDefPreview = computed(() => {
  if (!currentTemplate.value?.workflowDef) {
    return ''
  }
  return JSON.stringify(currentTemplate.value.workflowDef, null, 2)
})

async function loadCategories() {
  const res = await templateApi.categories()
  categories.value = res.data.data || []
}

async function loadList() {
  loading.value = true
  try {
    const res = await templateApi.market({
      keyword: queryForm.value.keyword || undefined,
      categoryId: queryForm.value.categoryId,
      page: page.value,
      pageSize: pageSize.value,
    })
    const data = res.data.data
    list.value = data?.list || []
    total.value = data?.total || 0
  } finally {
    loading.value = false
  }
}

async function handleSearch() {
  page.value = 1
  await loadList()
}

async function handleReset() {
  queryForm.value.keyword = ''
  queryForm.value.categoryId = undefined
  page.value = 1
  await loadList()
}

async function showDetail(templateId: string) {
  detailLoading.value = true
  detailVisible.value = true
  try {
    const res = await templateApi.detail(templateId)
    currentTemplate.value = res.data.data
  } finally {
    detailLoading.value = false
  }
}

async function installTemplate(templateId: string) {
  installLoadingId.value = templateId
  try {
    const res = await templateApi.install(templateId)
    const data = res.data.data as WorkflowTemplateInstallResult
    ElMessage.success(`安装成功，流程ID：${data.workflowId}`)
    await loadList()
  } finally {
    installLoadingId.value = ''
  }
}

function statusText(status: number): string {
  if (status === 1) {
    return '已发布'
  }
  if (status === 2) {
    return '已下架'
  }
  return '草稿'
}

function statusType(status: number): 'success' | 'warning' | 'info' {
  if (status === 1) {
    return 'success'
  }
  if (status === 2) {
    return 'info'
  }
  return 'warning'
}

onMounted(async () => {
  await Promise.all([loadCategories(), loadList()])
})
</script>

<template>
  <div class="page-container">
    <el-card>
      <template #header>
        <div class="card-header">
          <span>模板市场</span>
          <el-button @click="loadList"><el-icon><Refresh /></el-icon>刷新</el-button>
        </div>
      </template>

      <el-form inline class="filter-form">
        <el-form-item label="关键词">
          <el-input
            v-model="queryForm.keyword"
            placeholder="模板名称/描述"
            clearable
            @keyup.enter="handleSearch"
          />
        </el-form-item>
        <el-form-item label="分类">
          <el-select
            v-model="queryForm.categoryId"
            placeholder="全部分类"
            clearable
            style="width: 220px"
          >
            <el-option
              v-for="item in categories"
              :key="item.categoryId"
              :label="item.categoryName"
              :value="item.categoryId"
            />
          </el-select>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="handleSearch">查询</el-button>
          <el-button @click="handleReset">重置</el-button>
        </el-form-item>
      </el-form>

      <el-table v-loading="loading" :data="list" stripe>
        <el-table-column prop="templateName" label="模板名称" min-width="220" />
        <el-table-column prop="categoryName" label="分类" width="140" />
        <el-table-column prop="author" label="作者" width="140" />
        <el-table-column prop="version" label="版本" width="120" />
        <el-table-column label="状态" width="120">
          <template #default="{ row }">
            <el-tag :type="statusType(row.status)">
              {{ statusText(row.status) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="installCount" label="安装次数" width="100" />
        <el-table-column prop="updatedAt" label="更新时间" width="180" />
        <el-table-column label="操作" width="180" fixed="right">
          <template #default="{ row }">
            <el-button type="primary" link @click="showDetail(row.templateId)">详情</el-button>
            <el-button
              type="success"
              link
              :loading="installLoadingId === row.templateId"
              @click="installTemplate(row.templateId)"
            >
              安装
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

    <el-drawer v-model="detailVisible" title="模板详情" size="50%">
      <div v-loading="detailLoading" class="detail-container">
        <template v-if="currentTemplate">
          <el-descriptions :column="2" border>
            <el-descriptions-item label="模板名称">
              {{ currentTemplate.templateName }}
            </el-descriptions-item>
            <el-descriptions-item label="分类">
              {{ currentTemplate.categoryName || '-' }}
            </el-descriptions-item>
            <el-descriptions-item label="作者">
              {{ currentTemplate.author }}
            </el-descriptions-item>
            <el-descriptions-item label="版本">
              {{ currentTemplate.version }}
            </el-descriptions-item>
            <el-descriptions-item label="状态">
              <el-tag :type="statusType(currentTemplate.status)">
                {{ statusText(currentTemplate.status) }}
              </el-tag>
            </el-descriptions-item>
            <el-descriptions-item label="安装次数">
              {{ currentTemplate.installCount }}
            </el-descriptions-item>
            <el-descriptions-item label="描述" :span="2">
              {{ currentTemplate.description || '-' }}
            </el-descriptions-item>
            <el-descriptions-item label="标签" :span="2">
              <el-tag
                v-for="tag in currentTemplate.tags || []"
                :key="tag"
                class="tag-item"
              >
                {{ tag }}
              </el-tag>
              <span v-if="!currentTemplate.tags?.length">-</span>
            </el-descriptions-item>
          </el-descriptions>

          <el-card class="workflow-card">
            <template #header>流程定义预览</template>
            <pre class="workflow-json">{{ workflowDefPreview }}</pre>
          </el-card>
        </template>
      </div>
    </el-drawer>
  </div>
</template>

<style scoped>
.page-container {
  padding: 16px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.filter-form {
  margin-bottom: 16px;
}

.pagination {
  margin-top: 16px;
  display: flex;
  justify-content: flex-end;
}

.detail-container {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.tag-item {
  margin-right: 8px;
}

.workflow-card {
  margin-top: 16px;
}

.workflow-json {
  margin: 0;
  max-height: 480px;
  overflow: auto;
  background: #0f172a;
  color: #e2e8f0;
  padding: 16px;
  border-radius: 8px;
  font-size: 12px;
  line-height: 1.6;
}
</style>
