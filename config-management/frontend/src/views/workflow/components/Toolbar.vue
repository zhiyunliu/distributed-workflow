<template>
  <div class="toolbar">
    <div class="toolbar-left">
      <el-input
        v-model="workflowDef.name"
        placeholder="流程名称"
        style="width: 200px"
      />
      <el-input
        v-model="workflowDef.description"
        placeholder="流程描述"
        style="width: 300px"
      />
    </div>
    <div class="toolbar-right">
      <el-button @click="$emit('validate')">校验</el-button>
      <el-button :loading="saving" @click="$emit('saveDraft')">保存草稿</el-button>
      <el-button type="primary" :loading="saving" @click="$emit('publish')">发布</el-button>
      <el-button @click="$emit('back')">返回</el-button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { defineProps, defineEmits } from 'vue';
import type { WorkflowDef } from '@/types';

defineProps<{
  workflowDef: Partial<WorkflowDef>;
  saving: boolean;
}>();

defineEmits(['validate', 'saveDraft', 'publish', 'back']);
</script>

<style scoped>
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
</style>