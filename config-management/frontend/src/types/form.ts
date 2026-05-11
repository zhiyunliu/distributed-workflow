// 表单引擎相关 TypeScript 类型定义

export const FormStatus = {
  Draft: 0,
  Published: 1,
  Disabled: 2,
} as const

export type FormStatusType = typeof FormStatus[keyof typeof FormStatus]

export type SchemaObject = Record<string, unknown>
export type SchemaValue = string | SchemaObject

export interface LegacyFieldOption {
  label: string
  value: string
}

export interface LegacyFormField {
  name: string
  label: string
  type: 'text' | 'textarea' | 'number' | 'select' | 'date' | 'checkbox' | 'radio'
  required: boolean
  options: LegacyFieldOption[]
  placeholder: string
}

export interface PersistedFormSchema {
  formName?: string
  commonApi?: string
  fields: LegacyFormField[]
}

export interface DynamicFieldOption {
  label: string
  value: string
}

export interface DynamicTableColumn {
  key: string
  label: string
  type: 'input' | 'select'
  placeholder?: string
}

export interface DynamicFormField {
  key: string
  label: string
  type: 'input' | 'textarea' | 'date' | 'select' | 'switch' | 'selectMultiple' | 'table'
  required: boolean
  placeholder?: string
  defaultValue?: unknown
  apiParams?: Record<string, string | number | boolean>
  regex?: string
  regexMsg?: string
  columns?: DynamicTableColumn[]
  options?: DynamicFieldOption[]
}

export interface DynamicFormSchema {
  formName?: string
  commonApi?: string
  fields: DynamicFormField[]
}

export interface FormDefinition {
  formId: string
  formName: string
  description?: string
  schema?: SchemaValue
  formSchema: SchemaValue
  status: FormStatusType
  version: number
  createdBy: string
  createdAt: string
  updatedAt: string
}

export interface FormInstance {
  instanceId: string
  formId: string
  workflowInstanceId: string
  formData: string
  submittedBy: string
  submittedAt: string
  updatedAt: string
}

export interface FormVersionHistory {
  id: number
  formId: string
  version: number
  schema?: SchemaValue
  formSchema: SchemaValue
  publishedBy: string
  publishedAt: string
}

export interface FormListParams {
  page: number
  pageSize: number
  keyword?: string
}

export interface PageResult<T> {
  list: T[]
  total: number
  page: number
  pageSize: number
}
