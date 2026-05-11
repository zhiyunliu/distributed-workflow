## 迁移后的架构设计

### 职责清晰性
| 组件 | 职责 | 权限 |
|------|------|------|
| FormDesigner | 编辑编排、状态管理、选项加载、规则生成、保存提交流程 | 读写 editingData |
| DynamicFormEditor | 只读字段渲染与数据显示 | 只读 editingData |
| schema-adapter | schema/formSchema 双向兼容转换 | 无状态转换 |

### 数据流向
用户输入 -> FormDesigner(editingData)

FormDesigner(editingData + schema) -> DynamicFormEditor(只读预览)

保存动作 -> FormDesigner.saveForm -> schema-adapter(stringifyPersistedSchema) -> create/update API

### 通讯协议
- FormDesigner -> DynamicFormEditor
  - props: schema
  - props: editingData
  - props: readonly=true
- DynamicFormEditor -> FormDesigner
  - 无事件回传（纯展示）
- FormDesigner -> Backend
  - formApi.create / formApi.update
  - payload: formName + schema + formSchema

### 关键边界
- 编辑交互（addTableRow/delTableRow/saveForm）只在 FormDesigner
- 只读渲染只在 DynamicFormEditor
- 适配转换只在 schema-adapter

### 完成标志
- FormDesigner 已包含完整编辑逻辑
- DynamicFormEditor 继续保持只读展示
- editingData -> 预览面板实时同步
- lint/build 均通过
