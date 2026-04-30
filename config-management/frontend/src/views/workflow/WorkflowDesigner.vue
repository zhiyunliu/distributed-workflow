<script setup lang="ts">
/**
 * 工作流设计器页面
 * 使用 LogicFlow 实现拖拽式流程图设计
 * 支持节点类型：start、end、approval、script、http、subflow、condition
 */
import { ref, onMounted, onBeforeUnmount } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { workflowApi } from '@/api/workflow'
import { ElMessage } from 'element-plus'
import type { WorkflowDef, WorkflowConnection, WorkflowNode } from '@/types'

interface GraphNode {
  id: string
  type: string
  properties?: Record<string, unknown>
}

interface GraphEdge {
  id: string
  sourceNodeId: string
  targetNodeId: string
  properties?: Record<string, unknown>
}

interface LogicFlowLike {
  render(data: Record<string, unknown>): void
  getGraphData(): { nodes: GraphNode[]; edges: GraphEdge[] }
  destroy?: () => void
  dnd?: { startDrag: (node: { type: string }) => void }
  register: (type: string, definition: any) => void
  addNode: (node: any) => void
  getNodeModelById: (id: string) => any
  setTheme(theme: Record<string, any>): void
}

const route = useRoute()
const router = useRouter()

const workflowId = ref<string>((route.params.id as string) || '')
const containerRef = ref<HTMLElement | null>(null)
const currentDef = ref<Partial<WorkflowDef>>({ name: '新建流程', description: '' })
const loading = ref(false)
const saving = ref(false)

// LogicFlow 实例
let lf: LogicFlowLike | null = null

// 节点面板配置
const nodeTypes = [
  { type: 'start', label: '开始节点', icon: '▶', color: '#67c23a' },
  { type: 'end', label: '结束节点', icon: '■', color: '#f56c6c' },
  { type: 'approval', label: '审批节点', icon: '✔', color: '#409eff' },
  { type: 'script', label: '脚本节点', icon: '</>', color: '#e6a23c' },
  { type: 'http', label: 'HTTP 节点', icon: '⚡', color: '#909399' },
  { type: 'condition', label: '条件节点', icon: '◇', color: '#9c27b0' },
  { type: 'subflow', label: '子流程', icon: '⊡', color: '#00bcd4' },
]

// 注册节点类型
function registerNodeTypes() {
  if (!lf) return

  // 为每个节点类型注册
  nodeTypes.forEach(node => {
    // 注册自定义节点
    (lf as LogicFlowLike).register(node.type, ({ RectNode, RectNodeModel }: { RectNode: any; RectNodeModel: any }) => {
      // 自定义视图
      class View extends RectNode {
        static extendKey = `${node.type.toUpperCase()}_NODE_VIEW`
      }

      // 自定义模型
      class Model extends RectNodeModel {
        static extendKey = `${node.type.toUpperCase()}_NODE_MODEL`

        setAttributes() {
          super.setAttributes()
          
          // 设置节点样式
          this.fill = '#FFFFFF'
          this.stroke = node.color
          this.radius = 4
          
          // 设置节点文本样式
          this.text.style = {
            fontSize: 12,
            fill: '#333',
          }
        }
      }

      return {
        view: View,
        model: Model,
      }
    })
  })
}

async function initLogicFlow() {
  if (!containerRef.value) return
  try {
    const { default: LogicFlow } = await import('@logicflow/core')
    await import('@logicflow/core/dist/style/index.css')

    lf = new LogicFlow({
      container: containerRef.value,
      grid: true,
      keyboard: { enabled: true },
      background: {
        color: '#f7f9ff'
      },
      // 设置缩放限制
      stopScrollZoom: false,
    }) as unknown as LogicFlowLike

    // 注册节点类型
    registerNodeTypes()

    // 设置主题
    lf.setTheme({
      nodeText: { 
        overflowMode: 'ellipsis',
        lineHeight: 1.4,
      },
      edgeText: {
        overflowMode: 'ellipsis',
        lineHeight: 1.4,
      }
    })

    lf.render({})

    if (workflowId.value) {
      await loadWorkflow()
    }
  } catch (e) {
    console.error('LogicFlow 初始化失败', e)
    ElMessage.warning('流程设计器加载中，请稍候...')
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

async function saveDraft() {
  saving.value = true
  try {
    // 获取 LogicFlow 的图形数据并转换为后端格式
    const graphData = lf ? lf.getGraphData() : { nodes: [], edges: [] }
    const payload: Partial<WorkflowDef> = {
      ...currentDef.value,
      nodes: transformNodes(graphData),
      connections: transformEdges(graphData),
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

function transformNodes(graphData: { nodes: GraphNode[] }): Record<string, WorkflowNode> {
  // 将 LogicFlow nodes 转换为后端 WorkflowNode 格式
  const nodes = graphData.nodes || []
  return nodes.reduce<Record<string, WorkflowNode>>((acc, node) => {
    const id = node.id
    const props = node.properties || {}
    acc[id] = {
      id,
      name: (props.label as string) || node.type,
      type: node.type,
      description: '',
      config: props,
    }
    return acc
  }, {})
}

function transformEdges(graphData: { edges: GraphEdge[] }): WorkflowConnection[] {
  const edges = graphData.edges || []
  return edges.map((edge) => {
    const condition = edge.properties?.condition
    return {
      id: edge.id,
      sourceNodeId: edge.sourceNodeId,
      targetNodeId: edge.targetNodeId,
      type: 'sequence',
      condition: typeof condition === 'string' ? condition : '',
    }
  })
}

function onDragStart(type: string) {
  // 保存当前拖拽的节点类型，以便drop事件处理
  window.draggingNodeType = type
}

function onDrop(e: DragEvent) {
  if (!lf || !window.draggingNodeType) return

  const nodeId = `${window.draggingNodeType}_${Date.now()}`
  const rect = containerRef.value?.getBoundingClientRect()
  
  if (rect) {
    const x = e.clientX - rect.left
    const y = e.clientY - rect.top
    
    lf.addNode({
      id: nodeId,
      type: window.draggingNodeType,
      x,
      y,
      properties: {
        label: nodeTypes.find(nt => nt.type === window.draggingNodeType)?.label || window.draggingNodeType
      }
    })
    
    window.draggingNodeType = null
  }
}

function onDragOver(e: DragEvent) {
  e.preventDefault() // 必须阻止默认行为才能触发drop事件
}

onMounted(() => {
  initLogicFlow()
  
  // 添加拖放事件监听器
  const container = containerRef.value
  if (container) {
    container.addEventListener('drop', onDrop as EventListener)
    container.addEventListener('dragover', onDragOver as EventListener)
  }
})

onBeforeUnmount(() => {
  // 移除事件监听器
  const container = containerRef.value
  if (container) {
    container.removeEventListener('drop', onDrop as EventListener)
    container.removeEventListener('dragover', onDragOver as EventListener)
  }
  
  if (lf) {
    lf.destroy?.()
    lf = null
  }
})

// 全局变量用于存储拖拽的节点类型
declare global {
  interface Window {
    draggingNodeType: string | null
  }
}

if (typeof window !== 'undefined') {
  window.draggingNodeType = null
}
</script>

<template>
  <div class="designer-page">
    <!-- 工具栏 -->
    <div class="toolbar">
      <div class="toolbar-left">
        <el-input
          v-model="currentDef.name"
          placeholder="流程名称"
          style="width: 200px"
        />
        <el-input
          v-model="currentDef.description"
          placeholder="流程描述"
          style="width: 300px"
        />
      </div>
      <div class="toolbar-right">
        <el-button @click="validateWorkflow">校验</el-button>
        <el-button :loading="saving" @click="saveDraft">保存草稿</el-button>
        <el-button type="primary" :loading="saving" @click="publishWorkflow">发布</el-button>
        <el-button @click="router.back()">返回</el-button>
      </div>
    </div>

    <div class="designer-body">
      <!-- 节点面板 -->
      <div class="node-panel">
        <div class="panel-title">节点库</div>
        <div
          v-for="node in nodeTypes"
          :key="node.type"
          class="node-item"
          :data-type="node.type"
          draggable="true"
          @dragstart="($event as DragEvent).dataTransfer?.setData('nodeType', node.type); onDragStart(node.type)"
        >
          <span class="node-icon" :style="{ color: node.color }">{{ node.icon }}</span>
          <span class="node-label">{{ node.label }}</span>
        </div>
      </div>

      <!-- 画布 -->
      <div 
        ref="containerRef" 
        v-loading="loading" 
        class="lf-container" 
        @drop="onDrop" 
        @dragover="onDragOver"
      />

      <!-- 属性面板 -->
      <div class="props-panel">
        <div class="panel-title">属性配置</div>
        <el-empty description="选择节点以配置属性" :image-size="80" />
      </div>
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

.toolbar {
  height: 52px;
  background: #fff;
  border-bottom: 1px solid #e6e6e6;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 16px;
  gap: 12px;
  flex-shrink: 0;
}

.toolbar-left,
.toolbar-right {
  display: flex;
  align-items: center;
  gap: 8px;
}

.designer-body {
  flex: 1;
  display: flex;
  overflow: hidden;
}

.node-panel {
  width: 180px;
  background: #fff;
  border-right: 1px solid #e6e6e6;
  padding: 12px 0;
  flex-shrink: 0;
  overflow-y: auto;
}

.panel-title {
  font-size: 13px;
  font-weight: 600;
  color: #606266;
  padding: 0 16px 8px;
  border-bottom: 1px solid #f0f0f0;
  margin-bottom: 8px;
}

.node-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 16px;
  cursor: grab;
  transition: background 0.2s;
  user-select: none;
}

.node-item:hover {
  background: #f5f7fa;
}

.node-icon {
  font-size: 18px;
}

.node-label {
  font-size: 13px;
  color: #303133;
}

.lf-container {
  flex: 1;
  background: #fff;
  overflow: hidden;
}

.props-panel {
  width: 260px;
  background: #fff;
  border-left: 1px solid #e6e6e6;
  padding: 12px;
  flex-shrink: 0;
  overflow-y: auto;
}
</style>