## 迁移映射表

| 原位置 | 方法名 | 新位置 | 描述 |
|--------|--------|--------|------|
| DynamicFormEditor.vue:142 | onMounted | FormDesigner.vue | 迁移为 FormDesigner 生命周期内的编辑态初始化（读取并组织 editingData） |
| DynamicFormEditor.vue:151 | loadAllOptions | FormDesigner.vue | 迁移为 FormDesigner 的数据准备逻辑，完成后通过 props 传入只读展示层 |
| DynamicFormEditor.vue:168 | getRules | FormDesigner.vue | 迁移为设计器/提交流程校验逻辑（如需校验） |
| DynamicFormEditor.vue:183 | addTableRow | FormDesigner.vue | 迁移为设计器的表格编辑方法集合 |
| DynamicFormEditor.vue:187 | delTableRow | FormDesigner.vue | 迁移为设计器的表格编辑方法集合 |
| DynamicFormEditor.vue:192 | submitForm | FormDesigner.vue | 迁移为 FormDesigner 的保存/发布流程方法 |

> 说明：本次 T-FE-04 仅完成 DynamicFormEditor 展示层只读化，不在该组件中保留任何编辑入口。
