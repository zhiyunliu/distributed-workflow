import request from '@/utils/request'
import type { ApiResponse } from '@/types'
import type {
  FormDefinition,
  FormInstance,
  FormVersionHistory,
  FormListParams,
  PageResult,
} from '@/types/form'

export const formApi = {
  // 创建表单
  create: (data: Partial<FormDefinition>) =>
    request.post<ApiResponse<FormDefinition>>('/api/forms', data),

  // 查询表单列表
  list: (params: FormListParams) =>
    request.get<ApiResponse<PageResult<FormDefinition>>>('/api/forms', { params }),

  // 获取表单详情
  get: (formId: string) =>
    request.get<ApiResponse<FormDefinition>>(`/api/forms/${formId}`),

  // 更新表单
  update: (formId: string, data: Partial<FormDefinition>) =>
    request.put<ApiResponse<FormDefinition>>(`/api/forms/${formId}`, data),

  // 发布表单
  publish: (formId: string) =>
    request.post(`/api/forms/${formId}/publish`),

  // 回滚版本
  rollback: (formId: string, targetVersion: number) =>
    request.post(`/api/forms/${formId}/rollback`, { targetVersion }),

  // 获取版本历史
  getVersions: (formId: string) =>
    request.get<ApiResponse<FormVersionHistory[]>>(`/api/forms/${formId}/versions`),

  // 保存/更新表单实例
  saveInstance: (data: Partial<FormInstance>) =>
    request.post<ApiResponse<FormInstance>>('/api/form-instances', data),

  // 获取表单实例
  getInstance: (instanceId: string) =>
    request.get<ApiResponse<FormInstance>>(`/api/form-instances/${instanceId}`),

  // 按工作流实例获取表单
  getInstanceByWorkflow: (workflowInstanceId: string) =>
    request.get<ApiResponse<FormInstance>>(`/api/form-instances/workflow/${workflowInstanceId}`),
}
