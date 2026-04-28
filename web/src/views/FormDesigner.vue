<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { formApi } from '@/api/form'
import { ElMessage } from 'element-plus'

interface FieldOption { label: string; value: string }
interface FormField {
  name: string
  label: string
  type: 'text' | 'textarea' | 'number' | 'select' | 'date' | 'checkbox' | 'radio'
  required: boolean
  options: FieldOption[]
  placeholder: string
}

const FIELD_TYPES: { type: FormField['type']; label: string }[] = [
  { type: 'text', label: '单行文本' },
  { type: 'textarea', label: '多行文本' },
  { type: 'number', label: '数字' },
  { type: 'select', label: '下拉选择' },
  { type: 'date', label: '日期' },
  { type: 'checkbox', label: '多选框' },
  { type: 'radio', label: '单选框' },
]

const route = useRoute()
const router = useRouter()
const formId = computed(() => route.params.id as string | undefined)
const formName = ref('')
const fields = ref<FormField[]>([])
const selectedIndex = ref<number | null>(null)

const selectedField = computed(() =>
  selectedIndex.value !== null ? fields.value[selectedIndex.value] : null
)

const schemaPreview = computed(() => JSON.stringify({ fields: fields.value }, null, 2))

function addField(type: FormField['type']) {
  const f: FormField = {
    name: `field_${Date.now()}`,
    label: '新字段',
    type,
    required: false,
    options: [],
    placeholder: '',
  }
  fields.value.push(f)
  selectedIndex.value = fields.value.length - 1
}

function removeField(idx: number) {
  fields.value.splice(idx, 1)
  selectedIndex.value = null
}

function addOption() {
  if (selectedField.value) {
    selectedField.value.options.push({ label: '', value: '' })
  }
}

function removeOption(idx: number) {
  selectedField.value?.options.splice(idx, 1)
}

async function loadForm() {
  if (!formId.value) return
  try {
    const res = await formApi.get(formId.value)
    const def = res.data.data
    formName.value = def.formName
    try {
      const schema = JSON.parse(def.formSchema || '{}')
      fields.value = schema.fields || []
    } catch {
      fields.value = []
    }
  } catch {
    ElMessage.error('加载表单失败')
  }
}

async function saveDraft() {
  if (!formId.value) { ElMessage.warning('请先创建表单'); return }
  try {
    await formApi.update(formId.value, { formSchema: JSON.stringify({ fields: fields.value }) })
    ElMessage.success('草稿已保存')
  } catch {
    ElMessage.error('保存失败')
  }
}

async function publish() {
  if (!formId.value) { ElMessage.warning('请先创建表单'); return }
  try {
    await formApi.update(formId.value, { formSchema: JSON.stringify({ fields: fields.value }) })
    await formApi.publish(formId.value)
    ElMessage.success('发布成功')
    router.push('/form/list')
  } catch {
    ElMessage.error('发布失败')
  }
}

onMounted(loadForm)
</script>

<template>
  <div class="designer-wrap">
    <div class="toolbar">
      <span class="title">表单设计器 - {{ formName || '未命名' }}</span>
      <div>
        <el-button @click="saveDraft">保存草稿</el-button>
        <el-button type="primary" @click="publish">发布</el-button>
      </div>
    </div>

    <div class="main">
      <!-- 左侧字段类型 -->
      <div class="panel left-panel">
        <div class="panel-title">字段类型</div>
        <div
          v-for="ft in FIELD_TYPES"
          :key="ft.type"
          class="field-type-item"
          @click="addField(ft.type)"
        >
          + {{ ft.label }}
        </div>
      </div>

      <!-- 中间预览 -->
      <div class="panel center-panel">
        <div class="panel-title">表单预览</div>
        <div v-if="fields.length === 0" class="empty-tip">点击左侧字段类型添加字段</div>
        <div
          v-for="(field, idx) in fields"
          :key="field.name"
          class="preview-field"
          :class="{ active: selectedIndex === idx }"
          @click="selectedIndex = idx"
        >
          <span class="field-label">{{ field.label }}</span>
          <span class="field-type-tag">{{ field.type }}</span>
          <el-button type="danger" link size="small" @click.stop="removeField(idx)">删除</el-button>
        </div>
      </div>

      <!-- 右侧属性 -->
      <div class="panel right-panel">
        <div class="panel-title">字段属性</div>
        <template v-if="selectedField">
          <el-form label-width="80px" size="small">
            <el-form-item label="字段名">
              <el-input v-model="selectedField.name" />
            </el-form-item>
            <el-form-item label="标签">
              <el-input v-model="selectedField.label" />
            </el-form-item>
            <el-form-item label="占位符">
              <el-input v-model="selectedField.placeholder" />
            </el-form-item>
            <el-form-item label="必填">
              <el-switch v-model="selectedField.required" />
            </el-form-item>
            <template v-if="['select','checkbox','radio'].includes(selectedField.type)">
              <el-form-item label="选项">
                <div v-for="(opt, oi) in selectedField.options" :key="oi" class="option-row">
                  <el-input v-model="opt.label" placeholder="标签" style="width:80px" />
                  <el-input v-model="opt.value" placeholder="值" style="width:80px;margin:0 4px" />
                  <el-button type="danger" link @click="removeOption(oi)">删</el-button>
                </div>
                <el-button size="small" @click="addOption">+ 添加选项</el-button>
              </el-form-item>
            </template>
          </el-form>
        </template>
        <div v-else class="empty-tip">请选择字段</div>
      </div>
    </div>

    <!-- JSON 预览 -->
    <el-card class="schema-preview">
      <template #header><span>JSON Schema 预览</span></template>
      <pre>{{ schemaPreview }}</pre>
    </el-card>
  </div>
</template>

<style scoped>
.designer-wrap { display: flex; flex-direction: column; height: 100%; padding: 12px; gap: 12px; }
.toolbar { display: flex; justify-content: space-between; align-items: center; padding: 8px 0; }
.title { font-size: 16px; font-weight: 600; }
.main { display: flex; gap: 12px; flex: 1; min-height: 400px; }
.panel { border: 1px solid #e4e7ed; border-radius: 4px; padding: 12px; overflow-y: auto; }
.panel-title { font-weight: 600; margin-bottom: 10px; color: #303133; }
.left-panel { width: 140px; flex-shrink: 0; }
.center-panel { flex: 1; }
.right-panel { width: 240px; flex-shrink: 0; }
.field-type-item { padding: 6px 10px; cursor: pointer; border-radius: 4px; margin-bottom: 4px; background: #f5f7fa; }
.field-type-item:hover { background: #ecf5ff; color: #409eff; }
.preview-field { display: flex; align-items: center; gap: 8px; padding: 8px; border: 1px solid #e4e7ed; border-radius: 4px; margin-bottom: 6px; cursor: pointer; }
.preview-field.active { border-color: #409eff; background: #ecf5ff; }
.field-label { flex: 1; }
.field-type-tag { font-size: 12px; color: #909399; }
.option-row { display: flex; align-items: center; margin-bottom: 4px; }
.empty-tip { color: #c0c4cc; text-align: center; padding: 20px; }
.schema-preview pre { font-size: 12px; max-height: 160px; overflow-y: auto; margin: 0; }
</style>
