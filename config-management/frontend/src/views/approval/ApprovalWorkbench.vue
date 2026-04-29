<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { approvalApi } from '@/api/approval'
import { ElMessage, ElMessageBox } from 'element-plus'
import type { ApprovalTask } from '@/types'

const loading = ref(false)
const todoList = ref<ApprovalTask[]>([])
const historyList = ref<ApprovalTask[]>([])
const activeTab = ref('todo')
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)

const approvalDialog = ref(false)
const currentTask = ref<ApprovalTask | null>(null)
const approvalForm = ref({ action: 'approve', comment: '' })

async function loadTodo() {
  loading.value = true
  try {
    const res = await approvalApi.todoList({ page: page.value, pageSize: pageSize.value })
    const data = res.data.data
    todoList.value = data?.list || data || []
    total.value = data?.total ?? todoList.value.length
  } catch {
    // handled
  } finally {
    loading.value = false
  }
}

async function loadHistory() {
  loading.value = true
  try {
    const res = await approvalApi.historyList({ page: page.value, pageSize: pageSize.value })
    const data = res.data.data
    historyList.value = data?.list || data || []
  } catch {
    // handled
  } finally {
    loading.value = false
  }
}

function openApproval(task: ApprovalTask) {
  currentTask.value = task
  approvalForm.value = { action: 'approve', comment: '' }
  approvalDialog.value = true
}

async function submitApproval() {
  if (!currentTask.value) return
  await ElMessageBox.confirm('确认提交审批操作？', '提示')
  try {
    await approvalApi.approve(
      currentTask.value.instanceId,
      currentTask.value.nodeId,
      approvalForm.value,
    )
    ElMessage.success('审批成功')
    approvalDialog.value = false
    loadTodo()
  } catch {
    // handled
  }
}

onMounted(loadTodo)
</script>

<template>
  <div class="page-container">
    <el-card>
      <template #header>审批工作台</template>
      <el-tabs v-model="activeTab" @tab-click="activeTab === 'todo' ? loadTodo() : loadHistory()">
        <el-tab-pane label="待审批" name="todo">
          <el-table v-loading="loading" :data="todoList" stripe>
            <el-table-column prop="instanceId" label="实例ID" width="200" />
            <el-table-column prop="nodeName" label="节点名称" />
            <el-table-column prop="workflowId" label="流程ID" width="180" />
            <el-table-column prop="approver" label="审批人" width="120" />
            <el-table-column prop="createTime" label="创建时间" width="180" />
            <el-table-column label="操作" width="120" fixed="right">
              <template #default="{ row }">
                <el-button type="primary" link @click="openApproval(row)">审批</el-button>
              </template>
            </el-table-column>
          </el-table>
          <el-pagination
            v-model:current-page="page"
            v-model:page-size="pageSize"
            :total="total"
            layout="total, prev, pager, next"
            class="pagination"
            @change="loadTodo"
          />
        </el-tab-pane>

        <el-tab-pane label="已处理" name="history">
          <el-table v-loading="loading" :data="historyList" stripe>
            <el-table-column prop="instanceId" label="实例ID" width="200" />
            <el-table-column prop="nodeName" label="节点名称" />
            <el-table-column prop="approvalStatus" label="审批状态" width="120" />
            <el-table-column prop="createTime" label="创建时间" width="180" />
          </el-table>
        </el-tab-pane>
      </el-tabs>
    </el-card>

    <!-- 审批弹框 -->
    <el-dialog v-model="approvalDialog" title="审批操作" width="500px">
      <el-form :model="approvalForm" label-width="80px">
        <el-form-item label="审批意见">
          <el-radio-group v-model="approvalForm.action">
            <el-radio value="approve">同意</el-radio>
            <el-radio value="reject">拒绝</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="备注">
          <el-input
            v-model="approvalForm.comment"
            type="textarea"
            :rows="3"
            placeholder="请输入审批备注"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="approvalDialog = false">取消</el-button>
        <el-button type="primary" @click="submitApproval">确定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.page-container {
  padding: 16px;
}
.pagination {
  margin-top: 16px;
  display: flex;
  justify-content: flex-end;
}
</style>
