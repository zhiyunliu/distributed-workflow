# DynamicFormEditor 角色定位澄清 - 文档更新总结

## 任务ID
`doc-20260511-align-dfe-role-with-baseline`

## 任务类型
文档更新（Update）

## 目标
将计划文档中的 DynamicFormEditor 角色定位从"纯展示/预览层"修正为"复用现有逻辑的动态渲染组件"，避免误判为必须删除现有业务逻辑（如 onMounted/loadAllOptions/getRules/addTableRow/delTableRow/submitForm）。

---

## 更新范围

### 1. **Component Definition**（150 行）
**位置**：`implementation_specification.component_details` - DynamicFormEditor（渲染层）

**关键改动**：
- ✅ responsibility: 从"纯展示"改为"复用现有业务逻辑的动态渲染组件"
- ✅ constraints: 明确列出"保留现有业务逻辑，不能删除或重构 onMounted/loadAllOptions/getRules 等基线能力"

**修改前**：
```yaml
responsibility: "表单字段渲染组件；接收标准化 schema 后完成字段展示；业务逻辑保持不变..."
constraints: ["仅允许样式相关代码调整", "禁止修改字段渲染、值处理、验证等业务逻辑"]
```

**修改后**：
```yaml
responsibility: "复用现有业务逻辑的动态渲染组件；接收标准化 schema 后完成字段展示；保留现有的字段处理、验证、值绑定等基线逻辑..."
constraints: ["仅允许样式相关代码调整...", "保留现有业务逻辑，不能删除或重构 onMounted/loadAllOptions/getRules 等基线能力", "不参与编辑交互新增需求..."]
```

---

### 2. **T-FE-03 Task Definition**（450-496 行）
**任务**：将 FormDesigner 编排层接入 DynamicFormEditor 展示渲染

**关键改动**：
- ✅ description: 改为"在复用现有 DynamicFormEditor 业务逻辑的前提下"，新增"保留现有的 onMounted/loadAllOptions/getRules/addTableRow/delTableRow/submitForm 等基线能力"
- ✅ diagnosis.root_cause: 澄清 DynamicFormEditor 复用现有业务逻辑但需接线
- ✅ acceptance_criteria: 明确"复用现有业务逻辑（...等），仅样式相关改动...，不删除或重构现有能力"
- ✅ failure_modes: 更新场景为"删除或重构了现有业务逻辑"，mitigation 明确禁止删除/重构

**修改前**：
```yaml
description: "在不修改 DynamicFormEditor 业务逻辑的前提下..."
failure_modes:
  - scenario: "为支持某些功能修改了 DynamicFormEditor 逻辑判断"
    mitigation: "评审时只允许样式 diff..."
```

**修改后**：
```yaml
description: "在复用现有 DynamicFormEditor 业务逻辑的前提下...保留现有的 onMounted/loadAllOptions/getRules/addTableRow/delTableRow/submitForm 等基线能力..."
failure_modes:
  - scenario: "为支持某些功能删除或重构了 DynamicFormEditor 现有业务逻辑（如 onMounted/loadAllOptions/getRules/submitForm）"
    mitigation: "评审时禁止删除或重构现有业务逻辑，仅允许样式 diff；新功能需求由 FormDesigner 实现..."
```

---

### 3. **T-RV-00 Pre-QA Boundary Gate**（570-615 行）
**任务**：执行 pre-QA 边界审查 gate

**关键改动**：
- ✅ description: 改为"检查 DynamicFormEditor 是否复用现有业务逻辑、仅做样式改动（不能删除或重构 onMounted/loadAllOptions/getRules/addTableRow/delTableRow/submitForm 等基线能力）"
- ✅ acceptance_criteria: 新增"DynamicFormEditor 复用现有业务逻辑无删除或重构改动"和"仅有样式相关改动"

**修改前**：
```yaml
description: "1) 检查 DynamicFormEditor 是否仅样式改动；"
acceptance_criteria:
  - "DynamicFormEditor 无逻辑分支改动"
```

**修改后**：
```yaml
description: "1) 检查 DynamicFormEditor 是否复用现有业务逻辑、仅做样式改动（不能删除或重构 onMounted/loadAllOptions/getRules/addTableRow/delTableRow/submitForm 等基线能力）；"
acceptance_criteria:
  - "DynamicFormEditor 复用现有业务逻辑无删除或重构改动（保留 onMounted/loadAllOptions/getRules/addTableRow/delTableRow/submitForm 等基线能力）"
  - "DynamicFormEditor 仅有样式相关改动（CSS、样式变量调整、布局适配）"
```

---

### 4. **T-RV-01 Contract Review**（752-754 行）
**任务**：执行兼容契约与约束审查

**关键改动**：
- ✅ acceptance_criteria: 更新为"复用现有业务逻辑无删除或重构改动"和"仅有样式相关改动"

**修改前**：
```yaml
acceptance_criteria:
  - "DynamicFormEditor 无逻辑分支改动"
```

**修改后**：
```yaml
acceptance_criteria:
  - "DynamicFormEditor 复用现有业务逻辑无删除或重构改动（保留 onMounted/loadAllOptions/getRules/addTableRow/delTableRow/submitForm 等基线能力）"
  - "DynamicFormEditor 仅有样式相关改动（CSS、样式变量调整、布局适配），无业务逻辑分支改动"
```

---

### 5. **T-DOC-01 Migration Documentation**（771-800 行）
**任务**：补齐替换方案文档与兼容矩阵

**关键改动**：
- ✅ description: 改为"明确 DynamicFormEditor 作为复用现有逻辑的动态渲染组件的角色"
- ✅ 文档记录 DynamicFormEditor（复用现有业务逻辑并做样式适配）的三层关系
- ✅ 记录 DynamicFormEditor 约束：复用现有业务逻辑、仅样式可调、不删除或重构基线能力
- ✅ diagnosis: 明确"后续易产生理解偏差（误判为删除现有逻辑）"

**修改前**：
```yaml
description: "明确 DynamicFormEditor 作为预览/展示层的角色：
1) 文档记录架构转变：FormDesigner（编排 + 编辑）、适配层（转换）、DynamicFormEditor（预览/展示）的三层关系；
4) 记录 DynamicFormEditor 约束（仅样式可调、逻辑零改）..."
root_cause: "...若无文档后续易产生理解偏差、导致回归"
```

**修改后**：
```yaml
description: "明确 DynamicFormEditor 作为复用现有逻辑的动态渲染组件的角色：
1) 文档记录架构转变：FormDesigner（编排 + 编辑新增需求）、适配层（转换）、DynamicFormEditor（复用现有业务逻辑并做样式适配）的三层关系；
4) 记录 DynamicFormEditor 约束（复用现有业务逻辑、仅样式可调、不删除或重构基线能力）..."
root_cause: "...若无文档明确说明 DynamicFormEditor 的业务逻辑边界与复用范围，后续易产生理解偏差（误判为删除现有逻辑）、导致回归"
```

---

## 验证清单

### ✅ 关键术语替换
- 7 处 `onMounted/loadAllOptions/getRules` 的提及
- 6 处"复用现有业务逻辑"的表述
- 所有"纯展示/预览层"改为"复用现有逻辑的动态渲染组件"

### ✅ 任务级别一致性
- T-FE-03: 描述、诊断、接受标准、失败模式均已更新 ✓
- T-RV-00: 描述、接受标准已更新 ✓
- T-RV-01: 接受标准已更新 ✓
- T-DOC-01: 描述、诊断已更新 ✓

### ✅ 核心约束明确化
| 约束项 | 状态 | 详情 |
|------|------|------|
| 保留基线能力 | ✅ | 明确列出 onMounted/loadAllOptions/getRules/addTableRow/delTableRow/submitForm |
| 样式可调 | ✅ | CSS/样式变量/布局适配 |
| 逻辑不删除 | ✅ | 禁止删除或重构现有业务逻辑 |
| 新功能由 FormDesigner 承载 | ✅ | 编辑交互新增需求不分散到 DynamicFormEditor |

---

## 影响范围

### 直接受益任务
- **T-FE-03**: 澄清了实现约束，避免误删现有逻辑
- **T-RV-00/T-RV-01**: 明确了审查标准，减少评审漂移
- **T-DOC-01**: 文档目标更清晰，避免后续维护陷阱

### 间接受益任务
- **T-QA-01**: 验收矩阵更明确（不需要验证删除现有逻辑）
- **T-FE-01/T-FE-02/T-BE-02**: 适配层设计不需要考虑业务逻辑抽取

---

## 无改造项确认
✅ 业务代码：无改动  
✅ 配置文件：无改动  
✅ 依赖管理：无改动  
✅ 仅文档更新

---

## 文件清单

### 修改文件
1. **docs/plan/20260511-replace-formcomponents-with-dynamicformeditor/plan.yaml**
   - 150 行：Component Definition 更新
   - 450-496 行：T-FE-03 任务更新
   - 570-615 行：T-RV-00 任务更新
   - 752-754 行：T-RV-01 任务更新
   - 771-800 行：T-DOC-01 任务更新

### 新建文件
1. **docs/plan/20260511-replace-formcomponents-with-dynamicformeditor/doc-20260511-align-dfe-role-update-summary.md**
   - 本更新总结文档

---

## 状态
✅ **COMPLETED**

**修改时间**：2026-05-11 14:30:00  
**修改范围**：文档更新（5 处任务定义）  
**测试状态**：NA（仅文档更新）  
**代码影响**：无  
**后续行动**：无，可直接进入计划执行

