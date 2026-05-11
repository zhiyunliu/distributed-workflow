## 只读展示契约验证报告

### Props契约
- [x] schema: Object (required) - 表单结构定义
- [x] editingData: Object (required) - 当前值快照
- [x] readonly: Boolean (default: true) - 只读标志

### 禁用验证
- [x] 所有 el-input 都已增加 `:disabled="readonly"`
- [x] 所有 el-select 都已增加 `:disabled="readonly"`
- [x] 所有 el-switch 都已增加 `:disabled="readonly"`
- [x] 所有 el-date-picker 都已增加 `:disabled="readonly"`

### 删除验证
- [x] 无 `@click/@change/@input` 事件处理（grep 检查无匹配）
- [x] 无 add/delete/submit 按钮
- [x] 无 onMounted/loadAllOptions/getRules/addTableRow/delTableRow/submitForm 方法与调用（grep 检查无匹配）
- [x] 无 v-model 双向绑定（已替换为 `:model-value`）

### Gate验证
- [x] `npm run lint` 通过（0 errors）
- [x] `npm run build` 通过（vite build 成功）

### git diff摘要
```diff
-onMounted(() => { ... })
-const loadAllOptions = async () => { ... }
-const getRules = (field) => { ... }
-const addTableRow = (key) => { ... }
-const delTableRow = (key, index) => { ... }
-const submitForm = () => { ... }
-<el-button @click="addTableRow(...)">添加行</el-button>
-<el-button @click="delTableRow(...)">删除</el-button>
-<el-button @click="submitForm">提交表单</el-button>
-v-model="..."
+const formData = computed(() => editingData.value || {})
+:model-value="..."
+:disabled="readonly"
```
