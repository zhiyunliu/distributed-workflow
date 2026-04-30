<template>
  <div class="props-panel">
    <div class="panel-title">属性配置</div>
    <div v-if="selectedNode">
      <el-form label-width="80px" size="default">
        <el-form-item label="节点ID">
          <el-input v-model="selectedNode.id" disabled />
        </el-form-item>
        <el-form-item label="节点名称">
          <el-input 
            v-model="(selectedNode as any).config.label" 
            @input="(value: any) => updateProperty('label', value)"
          />
        </el-form-item>
        <el-form-item label="节点类型">
          <el-input v-model="selectedNode.type" disabled />
        </el-form-item>
        <el-form-item label="描述">
          <el-input 
            v-model="(selectedNode as any).config.description" 
            type="textarea"
            @input="(value: any) => updateProperty('description', value)"
          />
        </el-form-item>
        
        <!-- 根据节点类型显示特定配置 -->
        <div v-if="selectedNode.type === 'approval'">
          <el-divider>审批配置</el-divider>
          <el-form-item label="审批人">
            <el-input 
              v-model="(selectedNode as any).config.approver" 
              placeholder="输入审批人"
              @input="(value: any) => updateProperty('approver', value)"
            />
          </el-form-item>
          <el-form-item label="审批类型">
            <el-select 
              v-model="(selectedNode as any).config.approvalType"
              @change="(value: any) => updateProperty('approvalType', value)"
            >
              <el-option label="单人审批" value="single"></el-option>
              <el-option label="多人会签" value="multi"></el-option>
              <el-option label="依次审批" value="sequential"></el-option>
            </el-select>
          </el-form-item>
        </div>
        
        <div v-if="selectedNode.type === 'http'">
          <el-divider>HTTP配置</el-divider>
          <el-form-item label="请求地址">
            <el-input 
              v-model="(selectedNode as any).config.url" 
              placeholder="输入请求地址"
              @input="(value: any) => updateProperty('url', value)"
            />
          </el-form-item>
          <el-form-item label="请求方法">
            <el-select 
              v-model="(selectedNode as any).config.method"
              @change="(value: any) => updateProperty('method', value)"
            >
              <el-option label="GET" value="GET"></el-option>
              <el-option label="POST" value="POST"></el-option>
              <el-option label="PUT" value="PUT"></el-option>
              <el-option label="DELETE" value="DELETE"></el-option>
            </el-select>
          </el-form-item>
        </div>
        
        <div v-if="selectedNode.type === 'script'">
          <el-divider>脚本配置</el-divider>
          <el-form-item label="脚本语言">
            <el-select 
              v-model="(selectedNode as any).config.language"
              @change="(value: any) => updateProperty('language', value)"
            >
              <el-option label="JavaScript" value="javascript"></el-option>
              <el-option label="Python" value="python"></el-option>
            </el-select>
          </el-form-item>
          <el-form-item label="脚本内容">
            <el-input 
              v-model="(selectedNode as any).config.script" 
              type="textarea"
              :rows="4"
              placeholder="输入脚本内容"
              @input="(value: any) => updateProperty('script', value)"
            />
          </el-form-item>
        </div>
        
        <div v-if="selectedNode.type === 'condition'">
          <el-divider>条件配置</el-divider>
          <el-form-item label="条件表达式">
            <el-input 
              v-model="(selectedNode as any).config.expression" 
              type="textarea"
              placeholder="输入条件表达式，如: amount > 1000"
              @input="(value: any) => updateProperty('expression', value)"
            />
          </el-form-item>
        </div>
      </el-form>
    </div>
    <el-empty v-else description="请选择节点以配置属性" :image-size="80" />
  </div>
</template>

<script setup lang="ts">
import { defineProps, defineEmits } from 'vue';
import type { WorkflowNode } from '@/types';

interface Props {
  selectedNode: WorkflowNode | null;
}

defineProps<Props>();

const emit = defineEmits(['updateProperty']);

const updateProperty = (propName: string, value: any) => {
  emit('updateProperty', propName, value);
};
</script>

<style scoped>
.props-panel {
  width: 320px;
  background: #fff;
  border-left: 1px solid #e6e6e6;
  padding: 12px;
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

.el-divider {
  margin: 16px 0;
}
</style>