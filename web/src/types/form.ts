// 表单引擎相关 TypeScript 类型定义

export const FormStatus = {
  Draft: 0,
  Published: 1,
  Disabled: 2,
} as const

export type FormStatusType = typeof FormStatus[keyof typeof FormStatus]

export interface FormDefinition {
  formId: string
  formName: string
  description?: string
  formSchema: string
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
  formSchema: string
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
