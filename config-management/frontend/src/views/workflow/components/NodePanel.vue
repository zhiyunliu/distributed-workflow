<template>
  <div class="node-panel">
    <div class="panel-title">节点库</div>
    <div
      v-for="node in nodeTypes"
      :key="node.type"
      class="node-item"
      :data-type="node.type"
      draggable="true"
      @dragstart="onDragStart(node.type, $event)"
    >
      <span class="node-icon" :style="{ color: node.color }">{{ node.icon }}</span>
      <span class="node-label">{{ node.label }}</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { defineEmits } from 'vue';

interface NodeType {
  type: string;
  label: string;
  icon: string;
  color: string;
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

const emit = defineEmits(['dragStart']);

const onDragStart = (type: string, event: DragEvent) => {
  event.dataTransfer?.setData('nodeType', type);
  emit('dragStart', type);
};
</script>

<style scoped>
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
</style>