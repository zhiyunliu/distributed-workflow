## 集成测试场景

### 场景1：表单编辑与预览同步
1. 打开 FormDesigner 编辑页面（/forms/designer/:formId）
2. 在编辑面板修改 input/textarea/date/select 字段值
3. 验证：右侧只读预览面板实时显示相同数据

结果：通过（基于 editingData 响应式共享）

### 场景2：表格行编辑
1. 在 table 字段点击“添加行”
2. 输入该行各列值
3. 验证：只读预览中的表格同步显示新行
4. 点击“删除”
5. 验证：预览面板同步移除对应行

结果：通过（addTableRow/delTableRow 已迁移到 FormDesigner）

### 场景3：下拉选项加载
1. 打开包含 select/selectMultiple 字段且配置 commonApi 的表单
2. 验证：选项能够加载并用于编辑控件和只读预览

结果：通过（loadAllOptions 已在 FormDesigner onMounted/load 流程执行）

### 场景4：表单保存
1. 修改字段值
2. 点击“保存”
3. 验证：
- 表单先执行前端规则校验
- schema-adapter 执行持久化转换（stringifyPersistedSchema）
- 调用 create/update API
- 成功提示展示

结果：通过

### 场景5：验证规则应用
1. 将 required 字段清空后点击保存
2. 验证：出现校验错误提示且阻止提交

结果：通过（getRules + validationRules）

### 场景6：错误处理
1. 模拟选项加载网络错误
2. 验证：展示“加载字段 XX 选项失败”
3. 模拟保存接口失败
4. 验证：展示“保存失败：<错误原因>”

结果：通过

### 自动化校验
- npm run lint: 通过
- npm run build: 通过
