<script setup lang="ts">
import { ref, reactive, computed, watch, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { formApi } from '@/api/form'
import { ElMessage } from 'element-plus'
import request from '@/utils/request'
import type { FormInstance as ElFormInstance, FormRules } from 'element-plus'
import type { DynamicFormField, DynamicFormSchema, SchemaObject } from '@/types/form'
import { normalizeToDynamicSchema, stringifyPersistedSchema } from '@/utils/schema-adapter'

type OptionItem = { label: string; value: string }

const FIELD_TYPES: { type: DynamicFormField['type']; label: string }[] = [
  { type: 'input', label: '单行文本' },
  { type: 'textarea', label: '多行文本' },
  { type: 'select', label: '下拉选择' },
  { type: 'selectMultiple', label: '多选下拉' },
  { type: 'date', label: '日期' },
  { type: 'switch', label: '开关' },
  { type: 'table', label: '表格' },
]

const route = useRoute()
const router = useRouter()
const formId = computed(() => {
  const current = route.params.formId
  const legacy = route.params.id
  if (typeof current === 'string') return current
  if (Array.isArray(current) && current.length > 0) return current[0]
  if (typeof legacy === 'string') return legacy
  if (Array.isArray(legacy) && legacy.length > 0) return legacy[0]
  return undefined
})
const formName = ref('')
const commonApi = ref('')
const fields = ref<DynamicFormField[]>([])
const editingData = reactive<Record<string, unknown>>({})
const optionMap = ref<Record<string, OptionItem[]>>({})
const validationRules = reactive<FormRules>({})
const formRef = ref<ElFormInstance>()
const selectedIndex = ref<number | null>(null)

const selectedField = computed(() =>
  selectedIndex.value !== null ? fields.value[selectedIndex.value] : null
)

const dynamicSchema = computed(() =>
  ({
    formName: formName.value,
    commonApi: commonApi.value,
    fields: fields.value,
  } as DynamicFormSchema),
)

const dynamicSchemaKey = computed(() => JSON.stringify(dynamicSchema.value))

function initFieldValue(field: DynamicFormField): unknown {
  if (field.defaultValue !== undefined) {
    return field.defaultValue
  }
  if (field.type === 'selectMultiple') {
    return []
  }
  if (field.type === 'switch') {
    return false
  }
  if (field.type === 'table') {
    return []
  }
  return ''
}

function initEditingData() {
  const nextData: Record<string, unknown> = {}
  for (const field of fields.value) {
    nextData[field.key] = editingData[field.key] ?? initFieldValue(field)
  }
  for (const key of Object.keys(editingData)) {
    if (!(key in nextData)) {
      delete editingData[key]
    }
  }
  Object.assign(editingData, nextData)
}

function buildValidationRules() {
  const nextRules: FormRules = {}
  for (const field of fields.value) {
    nextRules[field.key] = getRules(field)
  }
  for (const key of Object.keys(validationRules)) {
    delete validationRules[key]
  }
  Object.assign(validationRules, nextRules)
}

function addField(type: DynamicFormField['type']) {
  const f: DynamicFormField = {
    key: `field_${Date.now()}`,
    label: '新字段',
    type,
    required: false,
    placeholder: '',
    options: type === 'select' || type === 'selectMultiple' ? [] : undefined,
    columns: type === 'table'
      ? [
        { key: 'col_1', label: '列1', type: 'input', placeholder: '请输入' },
      ]
      : undefined,
  }
  fields.value.push(f)
  selectedIndex.value = fields.value.length - 1
  editingData[f.key] = initFieldValue(f)
  validationRules[f.key] = getRules(f)
}

function addOption() {
  if (selectedField.value) {
    if (!selectedField.value.options) {
      selectedField.value.options = []
    }
    selectedField.value.options.push({ label: '', value: '' })
  }
}

function removeOption(idx: number) {
  selectedField.value?.options?.splice(idx, 1)
}

function addTableColumn() {
  if (!selectedField.value || selectedField.value.type !== 'table') {
    return
  }
  if (!selectedField.value.columns) {
    selectedField.value.columns = []
  }
  const nextIndex = selectedField.value.columns.length + 1
  selectedField.value.columns.push({
    key: `col_${Date.now()}`,
    label: `列${nextIndex}`,
    type: 'input',
    placeholder: '请输入',
  })
}

function removeTableColumn(index: number) {
  if (!selectedField.value || selectedField.value.type !== 'table' || !selectedField.value.columns) {
    return
  }
  selectedField.value.columns.splice(index, 1)
}

function getRules(field: DynamicFormField) {
  const rules: Array<Record<string, unknown>> = []
  if (field.required) {
    rules.push({
      required: true,
      message: `请输入${field.label}`,
      trigger: ['blur', 'change'],
    })
  }
  if (field.regex) {
    try {
      rules.push({
        pattern: new RegExp(field.regex),
        message: field.regexMsg || `${field.label}格式不正确`,
        trigger: ['blur', 'change'],
      })
    } catch {
      ElMessage.warning(`字段 ${field.label} 的正则表达式无效，已忽略`)
    }
  }
  return rules
}

function normalizeOptions(source: unknown): OptionItem[] {
  if (!Array.isArray(source)) {
    return []
  }
  return source
    .map((item) => {
      if (typeof item === 'string' || typeof item === 'number' || typeof item === 'boolean') {
        const value = String(item)
        return { label: value, value }
      }
      if (typeof item === 'object' && item !== null) {
        const obj = item as Record<string, unknown>
        const valueCandidate = obj.value ?? obj.id ?? obj.code ?? obj.key
        const labelCandidate = obj.label ?? obj.name ?? obj.title ?? valueCandidate
        if (valueCandidate != null && labelCandidate != null) {
          return { label: String(labelCandidate), value: String(valueCandidate) }
        }
      }
      return null
    })
    .filter((item): item is OptionItem => item !== null)
}

async function loadAllOptions(schema: DynamicFormSchema) {
  optionMap.value = {}
  const apiFields = schema.fields.filter((f) => f.type === 'select' || f.type === 'selectMultiple')

  for (const field of apiFields) {
    if (Array.isArray(field.options) && field.options.length > 0) {
      optionMap.value[field.key] = field.options
      continue
    }
    if (!schema.commonApi) {
      optionMap.value[field.key] = []
      continue
    }

    try {
      const res = await request.get(schema.commonApi, {
        params: field.apiParams,
      })
      const rawData = (res.data as { data?: unknown })?.data ?? res.data
      const options = normalizeOptions(rawData)
      optionMap.value[field.key] = options
      field.options = options
    } catch {
      optionMap.value[field.key] = []
      ElMessage.error(`加载字段 ${field.label} 选项失败`)
    }
  }
}

function addTableRow(fieldKey: string) {
  const targetField = fields.value.find((item) => item.key === fieldKey)
  const template = (targetField?.columns ?? []).reduce<Record<string, unknown>>((acc, column) => {
    acc[column.key] = ''
    return acc
  }, {})
  if (!Array.isArray(editingData[fieldKey])) {
    editingData[fieldKey] = []
  }
  ;(editingData[fieldKey] as Record<string, unknown>[]).push(template)
}

function delTableRow(fieldKey: string, index: number) {
  if (!Array.isArray(editingData[fieldKey])) {
    return
  }
  ;(editingData[fieldKey] as Record<string, unknown>[]).splice(index, 1)
}

function normalizeTableData(fieldKey: string): Record<string, unknown>[] {
  const value = editingData[fieldKey]
  return Array.isArray(value) ? (value as Record<string, unknown>[]) : []
}

function getRowInputValue(row: Record<string, unknown>, columnKey: string): string | number | null | undefined {
  const value = row[columnKey]
  if (typeof value === 'string' || typeof value === 'number' || value == null) {
    return value as string | number | null | undefined
  }
  return String(value)
}

function setRowInputValue(row: Record<string, unknown>, columnKey: string, value: string | number | null | undefined) {
  row[columnKey] = value
}

function getRowSelectValue(row: Record<string, unknown>, columnKey: string): SelectModelValue {
  const value = row[columnKey]
  if (value == null) {
    return value as null | undefined
  }
  if (typeof value === 'string' || typeof value === 'number' || typeof value === 'boolean') {
    return value
  }
  if (typeof value === 'object' && !Array.isArray(value)) {
    return value as Record<string, unknown>
  }
  return undefined
}

function setRowSelectValue(row: Record<string, unknown>, columnKey: string, value: SelectModelValue) {
  row[columnKey] = value
}

function getInputValue(fieldKey: string): string | number | null | undefined {
  const value = editingData[fieldKey]
  if (typeof value === 'string' || typeof value === 'number' || value == null) {
    return value as string | number | null | undefined
  }
  return String(value)
}

function setInputValue(fieldKey: string, value: string | number | null | undefined) {
  editingData[fieldKey] = value
}

type DateModelValue = string | number | Date | string[] | number[] | Date[] | null | undefined

function getDateValue(fieldKey: string): DateModelValue {
  const value = editingData[fieldKey]
  if (value == null) {
    return value as null | undefined
  }
  if (typeof value === 'string' || typeof value === 'number' || value instanceof Date) {
    return value
  }
  if (Array.isArray(value)) {
    return value as string[] | number[] | Date[]
  }
  return undefined
}

function setDateValue(fieldKey: string, value: DateModelValue) {
  editingData[fieldKey] = value
}

type SelectModelValue = string | number | boolean | Record<string, unknown> | null | undefined

function getSelectValue(fieldKey: string): SelectModelValue {
  const value = editingData[fieldKey]
  if (value == null) {
    return value as null | undefined
  }
  if (typeof value === 'string' || typeof value === 'number' || typeof value === 'boolean') {
    return value
  }
  if (typeof value === 'object' && !Array.isArray(value)) {
    return value as Record<string, unknown>
  }
  return undefined
}

function setSelectValue(fieldKey: string, value: SelectModelValue) {
  editingData[fieldKey] = value
}

function getMultiSelectValue(fieldKey: string): Array<string | number | boolean | Record<string, unknown>> {
  const value = editingData[fieldKey]
  return Array.isArray(value) ? (value as Array<string | number | boolean | Record<string, unknown>>) : []
}

function setMultiSelectValue(
  fieldKey: string,
  value: Array<string | number | boolean | Record<string, unknown>>,
) {
  editingData[fieldKey] = value
}

function getSwitchValue(fieldKey: string): string | number | boolean {
  const value = editingData[fieldKey]
  if (typeof value === 'string' || typeof value === 'number' || typeof value === 'boolean') {
    return value
  }
  return false
}

function setSwitchValue(fieldKey: string, value: string | number | boolean) {
  editingData[fieldKey] = value
}

async function saveForm(): Promise<boolean> {
  const valid = await formRef.value?.validate().catch(() => false)
  if (!valid) {
    ElMessage.warning('请先修正表单校验错误')
    return false
  }

  const schemaPayload = JSON.parse(JSON.stringify(dynamicSchema.value)) as SchemaObject
  // 使用 schema-adapter 进行数据转换（stringifyPersistedSchema 内部调用 normalizeToPersistedSchema）
  // 确保 schema > formSchema 的优先级规则被正确应用
  const payload = {
    formName: formName.value,
    schema: schemaPayload,
    formSchema: stringifyPersistedSchema({
      formName: formName.value,
      schema: dynamicSchema.value,
    }),
  }

  try {
    if (formId.value) {
      await formApi.update(formId.value, payload)
    } else {
      await formApi.create(payload)
    }
    ElMessage.success('保存成功')
    return true
  } catch (error) {
    const message = error instanceof Error ? error.message : '未知错误'
    ElMessage.error(`保存失败：${message}`)
    return false
  }
}

function cancel() {
  router.push('/forms')
}

async function loadForm() {
  if (!formId.value) {
    fields.value = []
    initEditingData()
    buildValidationRules()
    return
  }

  try {
    const res = await formApi.get(formId.value)
    const def = res.data.data
    const normalized = normalizeToDynamicSchema({
      formName: def.formName,
      schema: def.schema,
      formSchema: def.formSchema,
    })
    formName.value = normalized.formName || def.formName
    commonApi.value = normalized.commonApi || ''
    fields.value = normalized.fields
    initEditingData()
    buildValidationRules()
    await loadAllOptions(normalized)
  } catch {
    ElMessage.error('加载表单失败')
  }
}

const saveDraft = saveForm

async function publish() {
  if (!formId.value) { ElMessage.warning('请先创建表单'); return }
  const saveSuccess = await saveForm()
  if (!saveSuccess) {
    ElMessage.error('请先保存表单，再发布')
    return
  }
  try {
    await formApi.update(formId.value, {
      formSchema: stringifyPersistedSchema({
        formName: formName.value,
        schema: dynamicSchema.value,
      }),
      schema: JSON.parse(JSON.stringify(dynamicSchema.value)) as SchemaObject,
    })
    await formApi.publish(formId.value)
    ElMessage.success('发布成功')
    router.push('/forms')
  } catch {
    ElMessage.error('发布失败')
  }
}

watch(
  fields,
  () => {
    initEditingData()
    buildValidationRules()
  },
  { deep: true },
)

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

      <!-- 中间编辑与预览 -->
      <div class="panel center-panel">
        <div class="panel-title">编辑面板与只读预览</div>
        <div v-if="fields.length === 0" class="empty-tip">点击左侧字段类型添加字段</div>
        <div v-else class="editor-preview-grid">
          <div class="form-editor-panel">
            <div class="sub-title">编辑面板</div>
            <el-form
              ref="formRef"
              :model="editingData"
              :rules="validationRules"
              label-width="120px"
            >
              <el-form-item
                v-for="field in dynamicSchema.fields"
                :key="field.key"
                :label="field.label"
                :prop="field.key"
              >
                <el-input
                  v-if="field.type === 'input'"
                  :model-value="getInputValue(field.key)"
                  @update:model-value="setInputValue(field.key, $event)"
                  :placeholder="field.placeholder"
                />
                <el-input
                  v-else-if="field.type === 'textarea'"
                  :model-value="getInputValue(field.key)"
                  @update:model-value="setInputValue(field.key, $event)"
                  type="textarea"
                  :rows="3"
                  :placeholder="field.placeholder"
                />
                <el-date-picker
                  v-else-if="field.type === 'date'"
                  :model-value="getDateValue(field.key)"
                  @update:model-value="setDateValue(field.key, $event)"
                  type="date"
                  :placeholder="field.placeholder"
                  style="width: 100%"
                />
                <el-select
                  v-else-if="field.type === 'select'"
                  :model-value="getSelectValue(field.key)"
                  @update:model-value="setSelectValue(field.key, $event)"
                  :placeholder="field.placeholder || '请选择'"
                  style="width: 100%"
                >
                  <el-option
                    v-for="opt in optionMap[field.key]"
                    :key="opt.value"
                    :label="opt.label"
                    :value="opt.value"
                  />
                </el-select>
                <el-select
                  v-else-if="field.type === 'selectMultiple'"
                  :model-value="getMultiSelectValue(field.key)"
                  @update:model-value="setMultiSelectValue(field.key, $event)"
                  multiple
                  :placeholder="field.placeholder || '请选择'"
                  style="width: 100%"
                >
                  <el-option
                    v-for="opt in optionMap[field.key]"
                    :key="opt.value"
                    :label="opt.label"
                    :value="opt.value"
                  />
                </el-select>
                <el-switch
                  v-else-if="field.type === 'switch'"
                  :model-value="getSwitchValue(field.key)"
                  @update:model-value="setSwitchValue(field.key, $event)"
                />
                <div v-else-if="field.type === 'table'" class="table-editor-wrap">
                  <el-table :data="normalizeTableData(field.key)" border style="width: 100%">
                    <el-table-column
                      v-for="col in field.columns || []"
                      :key="col.key"
                      :label="col.label"
                      min-width="140"
                    >
                      <template #default="scope">
                        <el-input
                          v-if="col.type === 'input'"
                          :model-value="getRowInputValue(scope.row, col.key)"
                          @update:model-value="setRowInputValue(scope.row, col.key, $event)"
                          :placeholder="col.placeholder"
                        />
                        <el-select
                          v-else
                          :model-value="getRowSelectValue(scope.row, col.key)"
                          @update:model-value="setRowSelectValue(scope.row, col.key, $event)"
                          :placeholder="col.placeholder || '请选择'"
                          style="width: 100%"
                        >
                          <el-option
                            v-for="opt in optionMap[col.key] || []"
                            :key="opt.value"
                            :label="opt.label"
                            :value="opt.value"
                          />
                        </el-select>
                      </template>
                    </el-table-column>
                    <el-table-column label="操作" width="90" fixed="right">
                      <template #default="scope">
                        <el-button
                          type="danger"
                          link
                          @click="delTableRow(field.key, scope.$index)"
                        >
                          删除
                        </el-button>
                      </template>
                    </el-table-column>
                  </el-table>
                  <el-button
                    type="primary"
                    size="small"
                    class="table-add-row-btn"
                    @click="addTableRow(field.key)"
                  >
                    添加行
                  </el-button>
                </div>
              </el-form-item>
            </el-form>
            <div class="editor-actions">
              <el-button type="primary" @click="saveForm">保存</el-button>
              <el-button @click="cancel">取消</el-button>
            </div>
          </div>
          <div class="form-preview-panel">
            <div class="sub-title">只读预览</div>
            <DynamicFormEditor
              :key="dynamicSchemaKey"
              class="designer-preview"
              :schema="dynamicSchema"
              :editing-data="editingData"
              :readonly="true"
            />
          </div>
        </div>
      </div>

      <!-- 右侧属性 -->
      <div class="panel right-panel">
        <div class="panel-title">字段属性</div>
        <template v-if="selectedField">
          <el-form label-width="80px" size="small">
            <el-form-item label="字段Key">
              <el-input v-model="selectedField.key" />
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
            <template v-if="['select','selectMultiple'].includes(selectedField.type)">
              <el-form-item label="选项">
                <div v-for="(opt, oi) in selectedField.options || []" :key="oi" class="option-row">
                  <el-input v-model="opt.label" placeholder="标签" style="width:80px" />
                  <el-input v-model="opt.value" placeholder="值" style="width:80px;margin:0 4px" />
                  <el-button type="danger" link @click="removeOption(oi)">删</el-button>
                </div>
                <el-button size="small" @click="addOption">+ 添加选项</el-button>
              </el-form-item>
            </template>
            <template v-if="selectedField.type === 'table'">
              <el-form-item label="表格列">
                <div v-for="(col, ci) in selectedField.columns || []" :key="col.key" class="option-row">
                  <el-input v-model="col.label" placeholder="列名" style="width:80px" />
                  <el-input v-model="col.key" placeholder="列Key" style="width:80px;margin:0 4px" />
                  <el-select v-model="col.type" style="width: 90px">
                    <el-option label="输入框" value="input" />
                    <el-option label="下拉" value="select" />
                  </el-select>
                  <el-button type="danger" link @click="removeTableColumn(ci)">删</el-button>
                </div>
                <el-button size="small" @click="addTableColumn">+ 添加列</el-button>
              </el-form-item>
            </template>
            <el-form-item label="正则">
              <el-input v-model="selectedField.regex" placeholder="可选，例如 ^[0-9]+$" />
            </el-form-item>
            <el-form-item label="提示">
              <el-input v-model="selectedField.regexMsg" placeholder="正则校验失败提示" />
            </el-form-item>
            <el-form-item label="默认值">
              <el-input
                :model-value="String(selectedField.defaultValue ?? '')"
                @update:model-value="selectedField.defaultValue = $event"
              />
            </el-form-item>
          </el-form>
        </template>
        <div v-else class="empty-tip">请选择字段</div>
      </div>
    </div>

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
.center-panel { flex: 1; min-width: 0; }
.right-panel { width: 240px; flex-shrink: 0; }
.field-type-item { padding: 6px 10px; cursor: pointer; border-radius: 4px; margin-bottom: 4px; background: #f5f7fa; }
.field-type-item:hover { background: #ecf5ff; color: #409eff; }
.option-row { display: flex; align-items: center; margin-bottom: 4px; }
.empty-tip { color: #c0c4cc; text-align: center; padding: 20px; }
.designer-preview { width: 100%; }
.editor-preview-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 12px; align-items: start; }
.form-editor-panel, .form-preview-panel { border: 1px solid #ebeef5; border-radius: 4px; padding: 12px; }
.sub-title { font-size: 14px; font-weight: 600; margin-bottom: 10px; }
.table-editor-wrap { width: 100%; }
.table-add-row-btn { margin-top: 8px; }
.editor-actions { display: flex; gap: 8px; justify-content: flex-end; margin-top: 10px; }

@media (max-width: 1200px) {
  .editor-preview-grid { grid-template-columns: 1fr; }
}

:deep(.designer-preview.dynamic-form-editor) {
  max-width: none;
  padding: 0;
}

:deep(.designer-preview.dynamic-form-editor h3) {
  margin-top: 0;
  margin-bottom: 12px;
}

:deep(.designer-preview.dynamic-form-editor .el-form) {
  width: 100%;
}

:deep(.designer-preview.dynamic-form-editor .el-input),
:deep(.designer-preview.dynamic-form-editor .el-select),
:deep(.designer-preview.dynamic-form-editor .el-date-editor),
:deep(.designer-preview.dynamic-form-editor .el-input-number),
:deep(.designer-preview.dynamic-form-editor .el-textarea__inner) {
  width: 100% !important;
}
</style>
