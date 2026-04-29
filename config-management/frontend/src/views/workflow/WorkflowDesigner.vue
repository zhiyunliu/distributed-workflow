<script setup lang="ts">
/**
 * 工作流设计器页面
 * 使用 LogicFlow 实现拖拽式流程图设计
 * 支持节点类型：start、end、approval、script、http、subflow、condition
 */
import { ref, onMounted, onBeforeUnmount } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { workflowApi } from '@/api/workflow'
import { ElMessage, ElLoading } from 'element-plus'
import type { WorkflowDef } from '@/types'

const route = useRoute()
const router = useRouter()

const workflowId = ref<string>((route.params.id as string) || '')
const containerRef = ref<HTMLElement | null>(null)
const currentDef = ref<Partial<WorkflowDef>>({ name: '新建流程', description: '' })
const loading = ref(false)
const saving = ref(false)
const dialogVisible = ref(false)

// LogicFlow 实例
let lf: unknown = null

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

async function initLogicFlow() {
  if (!containerRef.value) return
  try {
    const { default: LogicFlow } = await import('@logicflow/core')
    await import('@logicflow/core/es/index.css')
    const { Menu, DndPanel, SelectionSelect, MiniMap } = await import('@logicflow/extension')
    await import('@logicflow/extension/es/index.css')

    lf = new LogicFlow({
      container: containerRef.value,
      plugins: [Menu, DndPanel, SelectionSelect, MiniMap],
      grid: true,
      keyboard: { enabled: true },
    })

    ;(lf as Record<string, Function>).render({})

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
    const graphData = lf ? (lf as Record<string, Function>).getGraphData() : { nodes: [], edges: [] }
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

function transformNodes(graphData: Record<string, unknown>) {
  // 将 LogicFlow nodes 转换为后端 WorkflowNode 格式
  const nodes = (graphData.nodes as unknown[]) || []
  return nodes.reduce<Record<string, unknown>>((acc, n) => {
    const node = n as Record<string, unknown>
    const id = node.id as string
    acc[id] = {
      id,
      name: (node.properties as Record<string, unknown>)?.label || node.type,
      type: node.type,
      description: '',
      config: node.properties || {},
    }
    return acc
  }, {})
}

function transformEdges(graphData: Record<string, unknown>) {
  const edges = (graphData.edges as unknown[]) || []
  return edges.map((e) => {
    const edge = e as Record<string, unknown>
    return {
      id: edge.id,
      sourceNodeId: edge.sourceNodeId,
      targetNodeId: edge.targetNodeId,
      type: 'sequence',
      condition: (edge.properties as Record<string, unknown>)?.condition || '',
    }
  })
}

function onDragNode(type: string) {
  if (lf) {
    ;(lf as Record<string, Function>).dnd.startDrag({ type })
  }
}

onMounted(initLogicFlow)

onBeforeUnmount(() => {
  if (lf) {
    ;(lf as Record<string, Function>).destroy?.()
    lf = null
  }
})
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
          draggable="true"
          @mousedown="onDragNode(node.type)"
        >
          <span class="node-icon" :style="{ color: node.color }">{{ node.icon }}</span>
          <span class="node-label">{{ node.label }}</span>
        </div>
      </div>

      <!-- 画布 -->
      <div ref="containerRef" v-loading="loading" class="lf-container" />

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
