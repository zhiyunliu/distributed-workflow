<template>
  <div class="dynamic-form-editor">
    <h3>{{ schema.formName }}</h3>
    <el-form :model="formData" :rules="validationRules" label-width="120px" class="mt-4" ref="formRef">
      <!-- 循环渲染所有表单项 -->
      <el-form-item
        v-for="field in schema.fields"
        :key="field.key"
        :label="field.label"
        :prop="field.key"
      >
        <!-- 1. 文本框 -->
        <el-input
          v-if="field.type === 'input'"
          :model-value="getInputValue(field.key)"
          @update:model-value="setInputValue(field.key, $event)"
          :placeholder="field.placeholder"
          :disabled="readonly"
          style="width: 400px"
        />

        <!-- 2. 文本域 -->
        <el-input
          v-else-if="field.type === 'textarea'"
          :model-value="getInputValue(field.key)"
          @update:model-value="setInputValue(field.key, $event)"
          :placeholder="field.placeholder"
          type="textarea"
          :rows="3"
          :disabled="readonly"
          style="width: 400px"
        />

        <!-- 3. 日期选择器 -->
        <el-date-picker
          v-else-if="field.type === 'date'"
          :model-value="getDateValue(field.key)"
          @update:model-value="setDateValue(field.key, $event)"
          type="date"
          :placeholder="field.placeholder"
          :disabled="readonly"
          style="width: 400px"
        />

        <!-- 4. 下拉单选 -->
        <el-select
          v-else-if="field.type === 'select'"
          :model-value="getSelectValue(field.key)"
          @update:model-value="setSelectValue(field.key, $event)"
          :placeholder="field.placeholder"
          :disabled="readonly"
          style="width: 400px"
        >
          <el-option
            v-for="opt in optionMap[field.key]"
            :key="opt.value"
            :label="opt.label"
            :value="opt.value"
          />
        </el-select>

        <!-- 5. 滑动开关 -->
        <el-switch
          v-else-if="field.type === 'switch'"
          :model-value="getSwitchValue(field.key)"
          @update:model-value="setSwitchValue(field.key, $event)"
          :disabled="readonly"
        />

        <!-- 6. 复选下拉 -->
        <el-select
          v-else-if="field.type === 'selectMultiple'"
          :model-value="getMultiSelectValue(field.key)"
          @update:model-value="setMultiSelectValue(field.key, $event)"
          multiple
          :placeholder="field.placeholder"
          :disabled="readonly"
          style="width: 400px"
        >
          <el-option
            v-for="opt in optionMap[field.key]"
            :key="opt.value"
            :label="opt.label"
            :value="opt.value"
          />
        </el-select>

        <!-- 7. 表格多行 -->
        <div v-else-if="field.type === 'table'" style="width: 100%">
          <el-table :data="normalizeToArray(formData[field.key])" border style="width: 100%">
            <el-table-column
              v-for="col in field.columns"
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
                  :disabled="readonly"
                />
                <el-select
                  v-else
                  :model-value="getRowSelectValue(scope.row, col.key)"
                  @update:model-value="setRowSelectValue(scope.row, col.key, $event)"
                  :placeholder="col.placeholder || '请选择'"
                  :disabled="readonly"
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
            <el-table-column v-if="!readonly" label="操作" width="90" fixed="right">
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
            v-if="!readonly"
            type="primary"
            size="small"
            :disabled="readonly"
            class="table-add-row-btn"
            @click="addTableRow(field.key)"
          >
            添加行
          </el-button>
        </div>
      </el-form-item>
    </el-form>
    <div v-if="!readonly" class="form-actions">
      <el-button type="primary" @click="submitForm">提交表单</el-button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch, onMounted } from 'vue'
import type { PropType } from 'vue'
import type { FormInstance as ElFormInstance, FormRules } from 'element-plus'
import { ElMessage } from 'element-plus'
import request from '@/utils/request'
import type { DynamicFormField, DynamicFormSchema } from '@/types/form'

type FormDataSnapshot = Record<string, unknown>
type OptionItem = { label: string; value: string }

// 1. 接收父组件传递的 JSON 表单配置和初始数据
const props = defineProps({
  schema: {
    type: Object as PropType<DynamicFormSchema>,
    required: true,
    validator: (value: DynamicFormSchema) => Array.isArray(value?.fields),
  },
  editingData: {
    type: Object as PropType<FormDataSnapshot>,
    default: () => ({})
  },
  readonly: {
    type: Boolean,
    default: false,
  }
})

const emit = defineEmits<{
  'update:editingData': [value: FormDataSnapshot]
}>()

// 2. 本地状态管理 - 支持独立编辑和双向绑定
const schema = computed(() => props.schema)
const readonly = computed(() => props.readonly)
const formData = reactive<FormDataSnapshot>({})
const optionMap = ref<Record<string, OptionItem[]>>({})
const validationRules = reactive<FormRules>({})
const formRef = ref<ElFormInstance>()

// 3. 初始化表单数据
function initFormData() {
  const nextData: FormDataSnapshot = {}
  for (const field of schema.value.fields) {
    nextData[field.key] = formData[field.key] ?? props.editingData[field.key] ?? getDefaultValue(field)
  }
  for (const key of Object.keys(formData)) {
    if (!(key in nextData)) {
      delete formData[key]
    }
  }
  Object.assign(formData, nextData)
}

function getDefaultValue(field: DynamicFormField): unknown {
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

// 4. 生成验证规则
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

function buildValidationRules() {
  const nextRules: FormRules = {}
  for (const field of schema.value.fields) {
    nextRules[field.key] = getRules(field)
  }
  for (const key of Object.keys(validationRules)) {
    delete validationRules[key]
  }
  Object.assign(validationRules, nextRules)
}

// 5. 规范选项数据
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

// 6. 异步加载下拉选项
async function loadAllOptions(schemaObj: DynamicFormSchema) {
  optionMap.value = {}
  const apiFields = schemaObj.fields.filter((f) => f.type === 'select' || f.type === 'selectMultiple')

  for (const field of apiFields) {
    if (Array.isArray(field.options) && field.options.length > 0) {
      optionMap.value[field.key] = field.options
      continue
    }
    if (!schemaObj.commonApi) {
      optionMap.value[field.key] = []
      continue
    }

    try {
      const res = await request.get(schemaObj.commonApi, {
        params: field.apiParams,
      })
      const rawData = (res.data as { data?: unknown })?.data ?? res.data
      const options = normalizeOptions(rawData)
      optionMap.value[field.key] = options
    } catch {
      optionMap.value[field.key] = []
      ElMessage.error(`加载字段 ${field.label} 选项失败`)
    }
  }
}

// 7. 表格行操作
function addTableRow(fieldKey: string) {
  const targetField = schema.value.fields.find((item) => item.key === fieldKey)
  const template = (targetField?.columns ?? []).reduce<Record<string, unknown>>((acc, column) => {
    acc[column.key] = ''
    return acc
  }, {})
  if (!Array.isArray(formData[fieldKey])) {
    formData[fieldKey] = []
  }
  ;(formData[fieldKey] as Record<string, unknown>[]).push(template)
  emitUpdate()
}

function delTableRow(fieldKey: string, index: number) {
  // 二层防护：检查权限
  if (readonly.value) {
    ElMessage.warning('只读模式下无法删除行')
    return
  }
  if (!Array.isArray(formData[fieldKey])) {
    return
  }
  ;(formData[fieldKey] as Record<string, unknown>[]).splice(index, 1)
  emitUpdate()
}

// 8. 提交表单
async function submitForm() {
  const valid = await formRef.value?.validate().catch(() => false)
  if (!valid) {
    ElMessage.warning('请先修正表单校验错误')
    return
  }
  ElMessage.success('表单提交成功')
  emitUpdate()
}

// 9. 向父组件同步数据
function emitUpdate() {
  emit('update:editingData', { ...formData })
}

// 10. Getter/Setter 方法 - 用于类型安全的值处理
function getInputValue(fieldKey: string): string | number | null | undefined {
  const value = formData[fieldKey]
  if (typeof value === 'string' || typeof value === 'number' || value == null) {
    return value as string | number | null | undefined
  }
  return String(value)
}

function setInputValue(fieldKey: string, value: string | number | null | undefined) {
  formData[fieldKey] = value
}

type DateModelValue = string | number | Date | string[] | number[] | Date[] | null | undefined

function getDateValue(fieldKey: string): DateModelValue {
  const value = formData[fieldKey]
  if (value == null) {
    return value as null | undefined
  }
  if (typeof value === 'string' || typeof value === 'number' || value instanceof Date) {
    return value
  }
  if (Array.isArray(value)) {
    if (value.every((item) => typeof item === 'string' || item instanceof Date)) {
      return value as string[] | Date[]
    }
    if (value.every((item) => typeof item === 'number')) {
      return value as number[]
    }
    return undefined
  }
  return undefined
}

function setDateValue(fieldKey: string, value: DateModelValue) {
  formData[fieldKey] = value
}

type SelectModelValue = string | number | boolean | Record<string, unknown> | null | undefined

function getSelectValue(fieldKey: string): SelectModelValue {
  const value = formData[fieldKey]
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
  formData[fieldKey] = value
}

function getMultiSelectValue(fieldKey: string): unknown[] {
  const value = formData[fieldKey]
  return Array.isArray(value) ? value : []
}

function setMultiSelectValue(fieldKey: string, value: unknown[]) {
  formData[fieldKey] = value
}

function getSwitchValue(fieldKey: string): string | number | boolean {
  const value = formData[fieldKey]
  if (typeof value === 'string' || typeof value === 'number' || typeof value === 'boolean') {
    return value
  }
  return false
}

function setSwitchValue(fieldKey: string, value: string | number | boolean) {
  formData[fieldKey] = value
}

// 11. 辅助函数
function normalizeToArray(value: unknown): Record<string, unknown>[] {
  if (Array.isArray(value)) {
    return value as Record<string, unknown>[]
  }
  return []
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

// 12. 监听 props 变化
watch(() => props.editingData, () => {
  initFormData()
}, { deep: true })

watch(() => props.schema, () => {
  initFormData()
  buildValidationRules()
  loadAllOptions(schema.value)
}, { deep: true })

// 13. 监听 formData 变化以同步父组件
watch(() => formData, () => {
  emitUpdate()
}, { deep: true, immediate: false })

// 14. 初始化生命周期
onMounted(() => {
  initFormData()
  buildValidationRules()
  loadAllOptions(schema.value)
})
</script>

<style scoped>
.dynamic-form-editor {
  padding: 20px;
  max-width: 1200px;
}

.form-actions {
  display: flex;
  gap: 8px;
  justify-content: flex-end;
  margin-top: 20px;
}

.table-add-row-btn {
  margin-top: 8px;
}
</style>