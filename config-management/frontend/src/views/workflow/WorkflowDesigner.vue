<script setup lang="ts">
/**
 * 工作流设计器页面
 * 使用 LogicFlow 实现拖拽式流程图设计
 * 支持节点类型：start、end、approval、script、http、subflow、condition
 */
import { ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { workflowApi } from '@/api/workflow'
import { ElMessage } from 'element-plus'
import type { WorkflowDef, WorkflowNode, WorkflowConnection } from '@/types'

import Toolbar from './components/Toolbar.vue'
import NodePanel from './components/NodePanel.vue'
import Canvas from './components/Canvas.vue'
import PropertiesPanel from './components/PropertiesPanel.vue'

const route = useRoute()
const router = useRouter()

const workflowId = ref<string>((route.params.id as string) || '')
const currentDef = ref<Partial<WorkflowDef>>({ name: '新建流程', description: '无' })
const loading = ref(false)
const saving = ref(false)
const selectedNode = ref<WorkflowNode | null>(null)

// 画布引用
const canvasRef = ref<InstanceType<typeof Canvas> | null>(null)

// 用于存储拖拽的节点类型
declare global {
  interface Window {
    draggingNodeType: string | null
  }
}

if (typeof window !== 'undefined') {
  window.draggingNodeType = null
}

const onDragStart = (type: string) => {
  window.draggingNodeType = type
}

const handleNodeSelect = (node: WorkflowNode) => {
  selectedNode.value = node
}

const updateNodeProperty = (propName: string, value: any) => {
  if (!selectedNode.value) return
  
  // 更新本地选中节点
  selectedNode.value = {
    ...selectedNode.value,
    config: {
      ...selectedNode.value.config,
      [propName]: value
    }
  }
}

async function loadWorkflow() {
  if (!workflowId.value) return
  loading.value = true
  try {
    const res = await workflowApi.getTopology(workflowId.value)
    const data = res.data.data
    currentDef.value = data.definition || {}
    // TODO: 将后端节点/连线数据转换为 LogicFlow 图形数据并渲染
  } catch {
    // error handled by interceptor
  } finally {
    loading.value = false
  }
}

// 从画布获取节点和连线数据
const getWorkflowDataFromCanvas = () => {
  if (!canvasRef.value || !canvasRef.value.getGraphData) {
    return { nodes: {}, connections: [] }
  }
  
  const graphData = canvasRef.value.getGraphData()
  if (!graphData) {
    return { nodes: {}, connections: [] }
  }
  
  // 转换画布节点数据为工作流节点格式
  const nodes = graphData.nodes.reduce((acc, node) => {
    acc[node.id] = {
      id: node.id,
      name: (node.properties?.label as string) || (node.type as string) || '未知节点',
      type: node.type,
      description: (node.properties?.description as string) || '',
      config: node.properties || {},
    }
    return acc
  }, {} as Record<string, WorkflowNode>)
  
  // 转换画布连线数据为工作流连接格式
  const connections = graphData.edges.map(edge => ({
    id: edge.id,
    sourceNodeId: edge.sourceNodeId,
    targetNodeId: edge.targetNodeId,
    type: 'sequence',
    condition: edge.properties?.condition || ''
  })) as WorkflowConnection[]
  
  return { nodes, connections }
}

async function saveDraft() {
  saving.value = true
  try {
    // 从画布获取最新的节点和连线数据
    const { nodes, connections } = getWorkflowDataFromCanvas()
    
    const payload: Partial<WorkflowDef> = {
      ...currentDef.value,
      nodes,
      connections,
    }
    
    const res = await workflowApi.saveDraft(payload)
    const newId = res.data.data?.id
    if (newId && !workflowId.value) {
      workflowId.value = newId
      router.replace(`/workflow/designer/${newId}`)
    }
    ElMessage.success('保存草稿成功')
  } catch {
    // error handled
  } finally {
    saving.value = false
  }
}

async function publishWorkflow() {
  await saveDraft()
  if (!workflowId.value) return
  try {
    await workflowApi.publish(currentDef.value)
    ElMessage.success('发布成功')
  } catch {
    // error handled
  }
}

async function validateWorkflow() {
  if (!workflowId.value) {
    ElMessage.warning('请先保存流程')
    return
  }
  try {
    const res = await workflowApi.validate(workflowId.value, currentDef.value)
    const data = res.data.data
    if (data.valid) {
      ElMessage.success('流程校验通过')
    } else {
      ElMessage.error('校验失败: ' + data.errors?.join(', '))
    }
  } catch {
    // error handled
  }
}

onMounted(() => {
  if (workflowId.value) {
    loadWorkflow()
  }
})
</script>

<template>
  <div class="designer-page">
    <!-- 工具栏 -->
    <Toolbar
      :workflowDef="currentDef"
      :saving="saving"
      @validate="validateWorkflow"
      @saveDraft="saveDraft"
      @publish="publishWorkflow"
      @back="router.back()"
    />

    <div class="designer-body">
      <!-- 节点面板 -->
      <NodePanel @dragStart="onDragStart" />

      <!-- 画布 -->
      <Canvas
        ref="canvasRef"
        :loading="loading"
        @nodeSelect="handleNodeSelect"
        @dragStart="onDragStart"
        @nodeAdded="() => {}"
      />

      <!-- 属性面板 -->
      <PropertiesPanel
        :selectedNode="selectedNode"
        @updateProperty="updateNodeProperty"
      />
    </div>
  </div>
</template>

<style scoped>
.designer-page {
  height: calc(100vh - 60px);
  display: flex;
  flex-direction: column;
  background: #f0f2f5;
}

.designer-body {
  flex: 1;
  display: flex;
  overflow: hidden;
}
</style>