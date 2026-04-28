// 系统管理相关 TypeScript 类型定义

export interface SystemUser {
  id: number
  username: string
  realName: string
  email: string
  phone: string
  avatar: string
  status: number
  deptId: number
  createTime: string
}

export interface SystemRole {
  id: number
  roleName: string
  roleCode: string
  description: string
  status: number
  dataScope: number
  createTime: string
}

export interface SystemMenu {
  id: number
  parentId: number
  menuName: string
  menuType: 'M' | 'C' | 'F'
  path: string
  component: string
  perms: string
  icon: string
  sort: number
  visible: number
  isFrame: number
  children?: SystemMenu[]
}

export interface DictionaryItem {
  dicId: number
  dictType: string
  dictName: string
  dictValue: string
  dictGroup: string
  sort: number
  status: number
  remark: string
  createTime: string
}

export interface LoginRequest {
  username: string
  password: string
}

export interface LoginResponse {
  token: string
  expiresIn: number
  user: SystemUser
  roles: SystemRole[]
  perms: string[]
}

// 工作流相关类型
export interface WorkflowDef {
  id: string
  name: string
  description: string
  version: number
  nodes: Record<string, WorkflowNode>
  connections: WorkflowConnection[]
  createdAt: string
  updatedAt: string
  createdBy: string
}

export interface WorkflowNode {
  id: string
  name: string
  type: string
  description: string
  config: Record<string, unknown>
  retryPolicy?: RetryPolicy
}

export interface WorkflowConnection {
  id: string
  sourceNodeId: string
  targetNodeId: string
  type: string
  condition: string
}

export interface RetryPolicy {
  maxRetries: number
  retryInterval: number
  retryOnErrors: string[]
}

export interface WorkflowInstance {
  id: string
  workflowId: string
  workflowVersion: number
  status: string
  startTime: string
  endTime?: string
  inputData: Record<string, unknown>
  outputData: Record<string, unknown>
  createdBy: string
  errorMessage: string
}

export interface ApprovalTask {
  instanceId: string
  nodeId: string
  workflowId: string
  nodeName: string
  approver: string
  approvalStatus: string
  createTime: string
}

export interface ApprovalRecord {
  id: string
  instanceId: string
  nodeId: string
  approver: string
  action: string
  comment: string
  operateTime: string
}

export interface WorkflowEndpoint {
  id: string
  name: string
  type: string
  workflowId: string
  config: Record<string, unknown>
  disabled: boolean
  triggerCount: number
  createTime: string
}

export interface AuditLog {
  id: string
  instanceId: string
  workflowId: string
  operationType: string
  operator: string
  operateTime: string
  operateIP: string
  detail: string
  beforeData: string
  afterData: string
}

export interface WorkflowTemplateCategory {
  categoryId: number
  categoryName: string
  parentId: number
  sort: number
  description: string
  createdAt: string
}

export interface WorkflowTemplate {
  templateId: string
  templateName: string
  categoryId: number
  categoryName: string
  description: string
  workflowDef?: WorkflowDef
  version: string
  author: string
  tags: string[]
  icon: string
  status: number
  visibleScope: number
  visibleRange: string[]
  installCount: number
  startCount: number
  createdAt: string
  updatedAt: string
}

export interface WorkflowTemplateInstallResult {
  templateId: string
  workflowId: string
}

// 分页响应
export interface PageResult<T> {
  list: T[]
  total: number
}

// 通用 API 响应
export interface ApiResponse<T> {
  code: number
  message: string
  data: T
}
