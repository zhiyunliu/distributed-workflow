## FormDesigner编辑逻辑迁移总结

### 新增状态管理
- editingData (reactive) - 集中管理所有字段值，并作为只读预览的数据源
- optionMap (ref) - 下拉选项数据缓存（静态 options + commonApi 动态加载）
- validationRules (reactive) - 动态规则缓存，通过 getRules 生成并绑定 el-form

### 新增方法（来自 DynamicFormEditor 迁移）
- loadAllOptions(schema) - 遍历 select/selectMultiple 字段，按 commonApi + apiParams 异步加载选项
- getRules(field) - 生成 required 与 regex 规则（含非法 regex 容错）
- addTableRow(fieldKey) - 表格字段追加行数据（按 columns 自动构造模板）
- delTableRow(fieldKey, index) - 删除指定表格行
- saveForm() - 提交保存（校验 -> schema-adapter 持久化转换 -> create/update API）

### 模板结构
- 编辑面板（form-editor-panel）
  - 逐字段渲染编辑控件（input/textarea/date/select/selectMultiple/switch/table）
  - 表格字段支持“添加行/删除行”
  - 保存/取消按钮集中在编辑面板
- 只读预览面板（form-preview-panel）
  - DynamicFormEditor 仅接收 schema + editingData + readonly
  - readonly 固定 true，纯展示
- 数据流
  - 用户输入 -> FormDesigner.editingData -> DynamicFormEditor 只读预览实时同步

### 关键集成点
- schema-adapter 在 saveForm 中通过 stringifyPersistedSchema 生成持久化 formSchema
- schema 对象同时回传，保证 schema/formSchema 双兼容
- API 调用统一使用 formApi.create/formApi.update
- 选项加载失败与保存失败均提供 ElMessage 友好提示

### 实施文件
- config-management/frontend/src/views/FormDesigner.vue

### 编译验证
- npm run lint: 通过
- npm run build: 通过
