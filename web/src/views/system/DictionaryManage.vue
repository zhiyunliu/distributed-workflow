<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { dictApi } from '@/api/dict'
import type { DictionaryItem } from '@/types'
import { ElMessage, ElMessageBox } from 'element-plus'

const loading = ref(false)
const dictTypes = ref<string[]>([])
const selectedType = ref('')
const list = ref<DictionaryItem[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)

const dialogVisible = ref(false)
const isEdit = ref(false)
const currentItem = ref<Partial<DictionaryItem>>({})
const formRef = ref()

async function loadTypes() {
  const res = await dictApi.listTypes()
  dictTypes.value = res.data.data || []
  if (dictTypes.value.length && !selectedType.value) {
    selectedType.value = dictTypes.value[0]
    loadList()
  }
}

async function loadList() {
  if (!selectedType.value) return
  loading.value = true
  try {
    const res = await dictApi.list({ dictType: selectedType.value, page: page.value, pageSize: pageSize.value })
    const data = res.data.data
    list.value = data?.list || data || []
    total.value = data?.total ?? list.value.length
  } finally {
    loading.value = false
  }
}

function openCreate() {
  isEdit.value = false
  currentItem.value = { dictType: selectedType.value, sort: 0, status: 1 }
  dialogVisible.value = true
}

function openEdit(row: DictionaryItem) {
  isEdit.value = true
  currentItem.value = { ...row }
  dialogVisible.value = true
}

async function handleSave() {
  await formRef.value?.validate()
  try {
    if (isEdit.value) {
      await dictApi.update(currentItem.value.dicId!, currentItem.value)
    } else {
      await dictApi.create(currentItem.value)
    }
    ElMessage.success(isEdit.value ? '更新成功' : '创建成功')
    dialogVisible.value = false
    loadList()
  } catch {
    // handled
  }
}

async function handleDelete(row: DictionaryItem) {
  await ElMessageBox.confirm(`确定删除字典项 "${row.dictName}" 吗？`, '警告', { type: 'warning' })
  await dictApi.delete(row.dicId)
  ElMessage.success('删除成功')
  loadList()
}

onMounted(loadTypes)
</script>

<template>
  <div class="page-container">
    <el-card>
      <template #header>
        <div class="card-header">
          <span>数据字典</span>
          <el-button type="primary" @click="openCreate">
            <el-icon><Plus /></el-icon>新建字典项
          </el-button>
        </div>
      </template>

      <!-- 字典类型 tab -->
      <el-tabs v-model="selectedType" @tab-click="loadList">
        <el-tab-pane
          v-for="t in dictTypes"
          :key="t"
          :label="t"
          :name="t"
        />
      </el-tabs>

      <el-table v-loading="loading" :data="list" stripe>
        <el-table-column prop="dicId" label="ID" width="80" />
        <el-table-column prop="dictName" label="字典名称" width="180" />
        <el-table-column prop="dictValue" label="字典值" width="160" />
        <el-table-column prop="dictGroup" label="分组" width="120" />
        <el-table-column prop="sort" label="排序" width="80" />
        <el-table-column label="状态" width="80">
          <template #default="{ row }">
            <el-tag :type="row.status === 1 ? 'success' : 'danger'">
              {{ row.status === 1 ? '启用' : '禁用' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="remark" label="备注" show-overflow-tooltip />
        <el-table-column label="操作" width="140" fixed="right">
          <template #default="{ row }">
            <el-button type="primary" link @click="openEdit(row)">编辑</el-button>
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

    <!-- 字典项表单弹框 -->
    <el-dialog v-model="dialogVisible" :title="isEdit ? '编辑字典项' : '新建字典项'" width="480px">
      <el-form ref="formRef" :model="currentItem" label-width="90px">
        <el-form-item label="字典类型" prop="dictType">
          <el-input v-model="currentItem.dictType" :disabled="isEdit" />
        </el-form-item>
        <el-form-item label="字典名称" :rules="[{ required: true, message: '必填', trigger: 'blur' }]" prop="dictName">
          <el-input v-model="currentItem.dictName" />
        </el-form-item>
        <el-form-item label="字典值" :rules="[{ required: true, message: '必填', trigger: 'blur' }]" prop="dictValue">
          <el-input v-model="currentItem.dictValue" />
        </el-form-item>
        <el-form-item label="分组" prop="dictGroup">
          <el-input v-model="currentItem.dictGroup" />
        </el-form-item>
        <el-form-item label="排序" prop="sort">
          <el-input-number v-model="currentItem.sort" :min="0" :max="999" />
        </el-form-item>
        <el-form-item label="备注" prop="remark">
          <el-input v-model="currentItem.remark" type="textarea" :rows="2" />
        </el-form-item>
        <el-form-item label="状态" prop="status">
          <el-switch v-model="currentItem.status" :active-value="1" :inactive-value="0" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="handleSave">保存</el-button>
      </template>
    </el-dialog>
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
.pagination {
  margin-top: 16px;
  display: flex;
  justify-content: flex-end;
}
</style>
