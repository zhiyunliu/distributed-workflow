## 删除清单

### Script中删除的方法
- [x] onMounted() - Line 142:148 - 初始化 formData 和加载选项
- [x] loadAllOptions() - Line 151:165 - 异步加载下拉选项数据
- [x] getRules() - Line 168:181 - 生成表单校验规则
- [x] addTableRow() - Line 183:185 - 添加表格行
- [x] delTableRow() - Line 187:189 - 删除表格行
- [x] submitForm() - Line 192:195 - 提交表单

### Template中删除的交互
- [x] "添加行"按钮 - Line 113:114
- [x] "删除行"按钮 - Line 107:109
- [x] "提交表单"按钮 - Line 120:120
- [x] v-model 双向绑定（改为只读 model-value）- 共 8 处（Line 16/24/34/43/58/64/89/94）
