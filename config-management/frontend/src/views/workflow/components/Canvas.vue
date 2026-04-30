<template>
  <div 
    ref="containerRef" 
    v-loading="loading" 
    class="lf-container" 
    @drop="onDrop" 
    @dragover="onDragOver"
    @keydown.delete="deleteSelected"
    tabindex="0"
  />
</template>

<script setup lang="ts">
import { ref, onMounted, onBeforeUnmount } from 'vue';

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
  on: (eventName: string, callback: (data: any) => void) => void
  off: (eventName: string) => void;
  removeSelected?: () => void;
  setEdgeType: (type: string) => void;
  updateText: (id: string, text: string) => void;
  getSelectElements: () => { nodes?: any[], edges?: any[] };
  selectNodeAsCurrent: (id: string) => void;
}

interface NodeType {
  type: string;
  label: string;
  icon: string;
  color: string;
}

interface WorkflowNode {
  id: string;
  name: string;
  type: string;
  description: string;
  config: Record<string, unknown>;
}

const nodeTypes: NodeType[] = [
  { type: 'start', label: '开始节点', icon: '▶', color: '#67c23a' },
  { type: 'end', label: '结束节点', icon: '■', color: '#f56c6c' },
  { type: 'approval', label: '审批节点', icon: '✔', color: '#409eff' },
  { type: 'script', label: '脚本节点', icon: '</>', color: '#e6a23c' },
  { type: 'http', label: 'HTTP 节点', icon: '⚡', color: '#909399' },
  { type: 'condition', label: '条件节点', icon: '◇', color: '#9c27b0' },
  { type: 'subflow', label: '子流程', icon: '⊡', color: '#00bcd4' },
];

const containerRef = ref<HTMLElement | null>(null);
const loading = ref(false);

// LogicFlow 实例
let lf: LogicFlowLike | null = null;

const emit = defineEmits([
  'nodeSelect',
  'dragStart',
  'nodeAdded',
  'drop',
  'dragOver'
]);

const initializeLogicFlow = async () => {
  if (!containerRef.value) return;
  
  try {
    const { default: LogicFlow } = await import('@logicflow/core');
    const { BpmnElement, Menu, DndPanel, SelectionSelect } = await import('@logicflow/extension');
    await import('@logicflow/core/dist/style/index.css');
    await import('@logicflow/extension/lib/style/index.css');

    lf = new LogicFlow({
      container: containerRef.value,
      grid: true,
      keyboard: { enabled: true },
      background: {
        color: '#f7f9ff'
      },
      stopScrollZoom: false,
      // 设置连线类型为贝塞尔曲线
      edgeType: 'bezier',
      // 注册扩展元素
      plugins: [BpmnElement, Menu, DndPanel, SelectionSelect],
    }) as unknown as LogicFlowLike;

    // 注册节点类型
    registerNodeTypes();

    // 设置主题
    lf.setTheme({
      nodeText: { 
        overflowMode: 'ellipsis',
        lineHeight: 1.4,
      },
      edgeText: {
        overflowMode: 'ellipsis',
        lineHeight: 1.4,
      },
      // 设置连线样式
      bezier: {
        stroke: '#8794FF',
        strokeWidth: 2,
        outlineColor: 'transparent',
        outlineWidth: 6,
      }
    });

    lf.render({});

    // 监听事件
    listenToEvents();
  } catch (e) {
    console.error('LogicFlow 初始化失败', e);
  }
};

// 注册节点类型
const registerNodeTypes = () => {
  if (!lf) return;

  // 为每个节点类型注册
  nodeTypes.forEach(node => {
    // 注册自定义节点
    (lf as LogicFlowLike).register(node.type, ({ RectNode, RectNodeModel, h }: { RectNode: any; RectNodeModel: any; h: any }) => {
      // 自定义视图
      class View extends RectNode {
        static extendKey = `${node.type.toUpperCase()}_NODE_VIEW`;
        
        getShape() {
          const { x, y } = this.props.model;
          
          // 创建矩形元素
          const rect = h('rect', {
            x: x - 60,  // 基于中心点调整位置
            y: y - 25,
            width: 120,
            height: 50,
            fill: '#FFFFFF',
            stroke: node.color,
            strokeWidth: 2,
            rx: 4,
            ry: 4,
          });

          // 创建文本元素显示节点图标和名称
          const text = h('text', {
            x: x,
            y: y, // 文本显示在矩形中心
            textAnchor: 'middle',
            dominantBaseline: 'middle',
            fill: node.color,
            fontWeight: 'bold',
            fontSize: 12,
          }, `${node.icon} ${node.label}`);

          return h('g', {}, [
            rect,
            text
          ]);
        }
      }

      // 自定义模型
      class Model extends RectNodeModel {
        static extendKey = `${node.type.toUpperCase()}_NODE_MODEL`;

        setAttributes() {
          super.setAttributes();
          
          // 设置节点尺寸
          this.width = 120;
          this.height = 50;
          
          // 设置节点样式
          this.fill = '#FFFFFF';
          this.stroke = node.color;
          this.radius = 4;
          
          // 设置节点文本样式
          this.text.style = {
            fontSize: 12,
            fill: '#333',
          };
        }
      }

      return {
        view: View,
        model: Model,
      };
    });
  });
};

// 节点选中事件处理
const handleNodeSelect = (nodeData: any) => {
  // 根据LogicFlow的数据结构创建节点对象
  const node: WorkflowNode = {
    id: nodeData.id,
    name: nodeData.properties?.label || nodeData.type || '未知节点',
    type: nodeData.type,
    description: nodeData.properties?.description || '',
    config: nodeData.properties || {},
  };
  
  emit('nodeSelect', node);
};

// 监听节点选中事件
const listenToEvents = () => {
  if (!lf) return;

  lf.on('selection:selected', (data) => {
    if (data.nodes && data.nodes.length > 0) {
      handleNodeSelect(data.nodes[0]);
    }
  });

  lf.on('node:click', (data) => {
    handleNodeSelect(data.data);
  });
};

const onDrop = (e: DragEvent) => {
  if (!lf) return;
  
  const rect = containerRef.value?.getBoundingClientRect();
  if (!rect) return;

  const x = e.clientX - rect.left;
  const y = e.clientY - rect.top;
  
  const nodeId = `${window.draggingNodeType}_${Date.now()}`;
  
  lf.addNode({
    id: nodeId,
    type: window.draggingNodeType,
    x,
    y,
    properties: {
      label: nodeTypes.find(nt => nt.type === window.draggingNodeType)?.label || window.draggingNodeType,
      description: nodeTypes.find(nt => nt.type === window.draggingNodeType)?.label || window.draggingNodeType,
    }
  });
  
  window.draggingNodeType = null;
  
  emit('nodeAdded');
  emit('drop', e);
};

const onDragOver = (e: DragEvent) => {
  e.preventDefault(); // 必须阻止默认行为才能触发drop事件
  emit('dragOver', e);
};

// 删除选中的节点或连线
const deleteSelected = () => {
  if (!lf) return;
  
  // 如果有removeSelected方法则使用它
  if (lf.removeSelected) {
    lf.removeSelected();
  } else {
    // 否则手动删除选中的元素
    const selectedData = lf.getSelectElements();
    if (selectedData && (selectedData.nodes || selectedData.edges)) {
      const nodeIds = selectedData.nodes?.map((n: any) => n.id) || [];
      const edgeIds = selectedData.edges?.map((e: any) => e.id) || [];
      
      // 删除节点
      nodeIds.forEach(id => {
        if (lf) {
          (lf as any).deleteNode && (lf as any).deleteNode(id);
        }
      });
      
      // 删除连线
      edgeIds.forEach(id => {
        if (lf && (lf as any).deleteEdge) {
          (lf as any).deleteEdge(id);
        }
      });
    }
  }
};

onMounted(() => {
  initializeLogicFlow();
  // 确保canvas组件获得焦点以便接收键盘事件
  containerRef.value?.focus();
});

onBeforeUnmount(() => {
  if (lf) {
    lf.off('selection:selected');
    lf.off('node:click');
    lf.destroy?.();
    lf = null;
  }
});

// 全局变量用于存储拖拽的节点类型
declare global {
  interface Window {
    draggingNodeType: string | null;
  }
}

if (typeof window !== 'undefined') {
  window.draggingNodeType = null;
}

defineExpose({
  lf,
  containerRef,
  getGraphData: () => lf?.getGraphData(),
});
</script>

<style scoped>
.lf-container {
  flex: 1;
  background: #fff;
  overflow: hidden;
  outline: none;
}
</style>