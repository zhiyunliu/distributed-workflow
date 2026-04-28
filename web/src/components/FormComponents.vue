<script setup lang="ts">
import { computed } from 'vue'

interface FieldOption {
  label: string
  value: string | number
}

interface FormField {
  name: string
  label: string
  type: 'text' | 'textarea' | 'number' | 'select' | 'date' | 'checkbox' | 'radio'
  required?: boolean
  options?: FieldOption[]
  placeholder?: string
}

const props = defineProps<{
  fields: FormField[]
  modelValue: Record<string, unknown>
  readonly?: boolean
}>()

const emit = defineEmits<{
  'update:modelValue': [value: Record<string, unknown>]
}>()

const formData = computed(() => props.modelValue)

function handleChange(name: string, val: unknown) {
  emit('update:modelValue', { ...props.modelValue, [name]: val })
}

const rules = computed(() => {
  const r: Record<string, { required: boolean; message: string; trigger: string }[]> = {}
  for (const f of props.fields) {
    if (f.required) {
      r[f.name] = [{ required: true, message: `${f.label}不能为空`, trigger: 'blur' }]
    }
  }
  return r
})
</script>

<template>
  <el-form :model="formData" :rules="rules" label-width="100px">
    <el-form-item
      v-for="field in fields"
      :key="field.name"
      :label="field.label"
      :prop="field.name"
    >
      <template v-if="field.type === 'text'">
        <el-input
          :model-value="formData[field.name] as string"
          :placeholder="field.placeholder"
          :readonly="readonly"
          @update:model-value="handleChange(field.name, $event)"
        />
      </template>

      <template v-else-if="field.type === 'textarea'">
        <el-input
          type="textarea"
          :model-value="formData[field.name] as string"
          :placeholder="field.placeholder"
          :readonly="readonly"
          :rows="3"
          @update:model-value="handleChange(field.name, $event)"
        />
      </template>

      <template v-else-if="field.type === 'number'">
        <el-input-number
          :model-value="formData[field.name] as number"
          :disabled="readonly"
          @update:model-value="handleChange(field.name, $event)"
        />
      </template>

      <template v-else-if="field.type === 'select'">
        <el-select
          :model-value="formData[field.name] as string"
          :placeholder="field.placeholder"
          :disabled="readonly"
          @update:model-value="handleChange(field.name, $event)"
        >
          <el-option
            v-for="opt in field.options"
            :key="opt.value"
            :label="opt.label"
            :value="opt.value"
          />
        </el-select>
      </template>

      <template v-else-if="field.type === 'date'">
        <el-date-picker
          :model-value="formData[field.name] as string"
          type="date"
          :placeholder="field.placeholder"
          :disabled="readonly"
          value-format="YYYY-MM-DD"
          @update:model-value="handleChange(field.name, $event)"
        />
      </template>

      <template v-else-if="field.type === 'checkbox'">
        <el-checkbox-group
          :model-value="formData[field.name] as (string | number)[]"
          :disabled="readonly"
          @update:model-value="handleChange(field.name, $event)"
        >
          <el-checkbox v-for="opt in field.options" :key="opt.value" :label="opt.value">
            {{ opt.label }}
          </el-checkbox>
        </el-checkbox-group>
      </template>

      <template v-else-if="field.type === 'radio'">
        <el-radio-group
          :model-value="formData[field.name] as string | number"
          :disabled="readonly"
          @update:model-value="handleChange(field.name, $event)"
        >
          <el-radio v-for="opt in field.options" :key="opt.value" :label="opt.value">
            {{ opt.label }}
          </el-radio>
        </el-radio-group>
      </template>
    </el-form-item>
  </el-form>
</template>

<style scoped>
.el-form {
  width: 100%;
}
</style>
