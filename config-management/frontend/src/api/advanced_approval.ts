import request from '@/utils/request'

export interface AddSignRequest {
  workflowInstanceId: string
  nodeId: string
  operatorId: string
  signType: 'before' | 'after'
  signUsers: string[]
  comment?: string
}

export interface TransferRequest {
  workflowInstanceId: string
  nodeId: string
  operatorId: string
  targetUserId: string
  comment?: string
}

export interface ReturnRequest {
  workflowInstanceId: string
  nodeId: string
  operatorId: string
  comment?: string
}

export interface WithdrawRequest {
  workflowInstanceId: string
  operatorId: string
  comment?: string
}

export interface DelegateConfig {
  delegatorId: string
  agentId: string
  startTime: string
  endTime: string
  workflowDefId?: string
  comment?: string
}

export interface CCRequest {
  workflowInstanceId: string
  nodeId: string
  ccUserIds: string[]
  comment?: string
}

export interface CCRecord {
  id: number
  workflowInstanceId: string
  nodeId: string
  ccUserId: string
  comment?: string
  isRead: boolean
  createdAt: string
}

export const advancedApprovalApi = {
  // 加签
  addSign: (data: AddSignRequest) =>
    request.post('/api/approval/add-sign', data),

  // 转签
  transfer: (data: TransferRequest) =>
    request.post('/api/approval/transfer', data),

  // 退回
  return: (data: ReturnRequest) =>
    request.post('/api/approval/return', data),

  // 撤回
  withdraw: (data: WithdrawRequest) =>
    request.post('/api/approval/withdraw', data),

  // 设置代理
  setDelegate: (data: DelegateConfig) =>
    request.post('/api/approval/delegate', data),

  // 获取代理列表
  getDelegates: (userId: string) =>
    request.get<DelegateConfig[]>(`/api/approval/delegates/${userId}`),

  // 发送抄送
  sendCC: (data: CCRequest) =>
    request.post('/api/approval/cc', data),

  // 获取抄送列表
  getCCList: (params: { userId: string; page: number; pageSize: number }) =>
    request.get<{ list: CCRecord[]; total: number }>('/api/approval/cc/list', { params }),

  // 标记抄送已读
  markCCRead: (ccId: number) =>
    request.put(`/api/approval/cc/${ccId}/read`),
}
