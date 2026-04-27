import request from '@/utils/request'
import type { WorkflowDef } from '@/types'

export const workflowApi = {
  // 设计器接口
  saveDraft: (data: Partial<WorkflowDef>) =>
    request.post('/workflow/def/save-draft', data),
  publish: (data: Partial<WorkflowDef>) =>
    request.post('/workflow/def/publish', data),
  getVersions: (id: string) => request.get(`/workflow/def/${id}/versions`),
  getVersion: (workflowId: string, version: number) =>
    request.get(`/workflow/def/version/${version}`, { params: { workflowId } }),
  rollback: (workflowId: string, targetVersion: number) =>
    request.post('/workflow/def/rollback', { workflowId, targetVersion }),
  importDef: (data: WorkflowDef) =>
    request.post('/workflow/def/import', data),
  exportDef: (id: string) =>
    request.get(`/workflow/def/${id}/export`, { responseType: 'blob' }),
  validate: (id: string, data: Partial<WorkflowDef>) =>
    request.post(`/workflow/def/${id}/validate`, data),
  list: () => request.get('/workflow/def/list'),

  // 实例接口（原有）
  getTopology: (id: string) =>
    request.get(`/workflow/instance/${id}/topology`),
  getInstanceNodes: (id: string) =>
    request.get(`/workflow/instance/${id}/nodes`),
  getInstanceContext: (id: string) =>
    request.get(`/workflow/instance/${id}/context`),
  getInstanceEvents: (id: string) =>
    request.get(`/workflow/instance/${id}/events`),
  retryNode: (instanceId: string, nodeId: string) =>
    request.post(`/workflow/instance/${instanceId}/node/${nodeId}/retry`),
}
