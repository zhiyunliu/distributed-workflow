import request from '@/utils/request'
import type { WorkflowTemplate } from '@/types'

export interface TemplateMarketQuery {
  keyword?: string
  categoryId?: number
  page?: number
  pageSize?: number
}

export const templateApi = {
  market: (params: TemplateMarketQuery) => request.get('/template/market', { params }),
  categories: () => request.get('/template/categories'),
  detail: (templateId: string) => request.get(`/template/${templateId}`),
  install: (templateId: string) => request.post(`/template/${templateId}/install`),
  create: (data: WorkflowTemplate) => request.post('/template/create', data),
  update: (templateId: string, data: WorkflowTemplate) =>
    request.put(`/template/${templateId}/update`, data),
  publish: (templateId: string) => request.post(`/template/${templateId}/publish`),
  importTemplate: (data: WorkflowTemplate) => request.post('/template/import', data),
  exportTemplate: (templateId: string) =>
    request.get(`/template/${templateId}/export`, { responseType: 'blob' }),
}
