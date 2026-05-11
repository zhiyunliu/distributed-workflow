<template>
  <div class="dynamic-form-editor">
    <h3>{{ schema.formName }}</h3>
    <el-form :model="formData" label-width="120px" class="mt-4">
      <!-- 循环渲染所有表单项 -->
      <el-form-item
        v-for="field in schema.fields"
        :key="field.key"
        :label="field.label"
        :prop="field.key"
        :rules="getRules(field)"
      >
        <!-- 1. 文本框 -->
        <el-input
          v-if="field.type === 'input'"
          v-model="formData[field.key]"
          :placeholder="field.placeholder"
          style="width: 400px"
        />

        <!-- 2. 文本域 -->
        <el-input
          v-else-if="field.type === 'textarea'"
          v-model="formData[field.key]"
          :placeholder="field.placeholder"
          type="textarea"
          :rows="3"
          style="width: 400px"
        />

        <!-- 3. 日期选择器 -->
        <el-date-picker
          v-else-if="field.type === 'date'"
          v-model="formData[field.key]"
          type="date"
          :placeholder="field.placeholder"
          style="width: 400px"
        />

        <!-- 4. 下拉单选 -->
        <el-select
          v-else-if="field.type === 'select'"
          v-model="formData[field.key]"
          :placeholder="field.placeholder"
          style="width: 400px"
        >
          <el-option
            v-for="opt in optionMap[field.key]"
            :key="opt.value"
            :label="opt.label"
            :value="opt.value"
          />
        </el-select>

        <!-- 5. 滑动开关 -->
        <el-switch
          v-else-if="field.type === 'switch'"
          v-model="formData[field.key]"
        />

        <!-- 6. 复选下拉 -->
        <el-select
          v-else-if="field.type === 'selectMultiple'"
          v-model="formData[field.key]"
          multiple
          placeholder="请选择"
          style="width: 400px"
        >
          <el-option
            v-for="opt in optionMap[field.key]"
            :key="opt.value"
            :label="opt.label"
            :value="opt.value"
          />
        </el-select>

        <!-- 7. 表格多行 -->
        <div v-else-if="field.type === 'table'" style="width: 100%">
          <el-table :data="formData[field.key]" border style="width: 100%">
            <el-table-column
              v-for="col in field.columns"
              :key="col.key"
              :label="col.label"
            >
              <template #default="scope">
                <!-- 表格内控件（复用外部控件逻辑） -->
                <el-input
                  v-if="col.type === 'input'"
                  v-model="scope.row[col.key]"
                  :placeholder="col.placeholder"
                />
                <el-select
                  v-else-if="col.type === 'select'"
                  v-model="scope.row[col.key]"
                >
                  <el-option
                    v-for="opt in optionMap[col.key]"
                    :key="opt.value"
                    :label="opt.label"
                    :value="opt.value"
                  />
                </el-select>
              </template>
            </el-table-column>
            <el-table-column label="操作">
              <template #default="scope">
                <el-button type="danger" size="small" @click="delTableRow(field.key, scope.$index)">
                  删除
                </el-button>
              </template>
            </el-table-column>
          </el-table>
          <el-button type="primary" size="small" class="mt-2" @click="addTableRow(field.key)">
            添加行
          </el-button>
        </div>
      </el-form-item>
    </el-form>

    <!-- 提交按钮 -->
    <el-button type="primary" class="mt-6" @click="submitForm">提交表单</el-button>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted, computed } from 'vue'
import axios from 'axios'
import { ElMessage } from 'element-plus'

// 1. 接收父组件传递的JSON表单配置
const props = defineProps({
  schema: {
    type: Object,
    required: true
  }
})

// 2. 表单数据双向绑定
const formData = reactive({})
// 存储所有下拉选项数据（key: 字段key, value: 选项数组）
const optionMap = ref({})

// 3. 初始化：赋值默认值 + 加载下拉数据
onMounted(() => {
  // 初始化表单默认值
  props.schema.fields.forEach(field => {
    formData[field.key] = field.defaultValue
  })
  // 加载所有下拉/多选下拉数据源
  loadAllOptions()
})

// 4. 统一加载下拉数据
const loadAllOptions = async () => {
  const { commonApi, fields } = props.schema
  // 筛选需要请求接口的字段
  const apiFields = fields.filter(f => f.type === 'select' || f.type === 'selectMultiple')
  
  for (const field of apiFields) {
    try {
      const res = await axios.get(commonApi, { params: field.apiParams })
      optionMap.value[field.key] = res.data
    } catch (e) {
      ElMessage.error('加载选项失败')
      optionMap.value[field.key] = []
    }
  }
}

// 5. 生成表单校验规则
const getRules = (field) => {
  const rules = []
  if (field.required) {
    rules.push({ required: true, message: `请输入${field.label}`, trigger: 'blur' })
  }
  if (field.regex) {
    rules.push({
      pattern: new RegExp(field.regex),
      message: field.regexMsg,
      trigger: 'blur'
    })
  }
  return rules
}

// 6. 表格添加行
const addTableRow = (key) => {
  formData[key].push({})
}

// 7. 表格删除行
const delTableRow = (key, index) => {
  formData[key].splice(index, 1)
}

// 8. 表单提交（输出最终数据）
const submitForm = () => {
  console.log('表单最终数据：', formData)
  ElMessage.success('提交成功！')
}
</script>

<style scoped>
.dynamic-form-editor {
  padding: 20px;
  max-width: 1200px;
}
</style>