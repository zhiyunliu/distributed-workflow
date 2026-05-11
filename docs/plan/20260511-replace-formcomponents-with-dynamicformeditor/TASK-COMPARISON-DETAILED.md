# 任务变更详细对比

## 🔍 11 个任务概览（新增 1 个）

### Task Status Matrix

| 任务ID | 波次 | 状态 | 旧→新 | 优先级 | 工作量 | 变更类型 |
|---------|------|------|------|--------|--------|---------|
| T-BE-00 | W1 | 基础 | 同 | High | Small | 无改动 |
| T-BE-01 | W1 | 基础 | 同 | High | Medium | 无改动 |
| T-FE-01 | W1 | 基础 | 同 | High | Medium | 无改动 |
| T-FE-02 | W1 | 基础 | 同 | Medium | Small | 无改动 |
| **T-FE-04** | **W2** | **⭐NEW** | **N/A** | **High** | **Large** | **新增** |
| T-FE-03 | W2 | 修改 | 接入→接入+迁移 | High | Medium→Large | 职责扩大 |
| T-BE-02 | W2 | 基础 | 同 | High | Medium | 无改动 |
| T-RV-00 | W3 | 修改 | 样式检查→逻辑分离检查 | High | Small→Medium | 审查标准升级 |
| T-QA-01 | W4 | 修改 | 兼容性→兼容性+编辑交互 | High | Medium→Large | 验证场景扩大 |
| T-RV-01 | W4 | 修改 | 样式审查→逻辑分离审查 | Medium | Small→Medium | 审查重点升级 |
| T-DOC-01 | W5 | 修改 | 基础文档→含约束变更说明 | Medium | Small→Medium | 文档范围扩大 |

---

## 📋 新增任务详解：T-FE-04-dynamicformeditor-refactor

### 基本信息
```yaml
ID: T-FE-04-dynamicformeditor-refactor
Title: 【新增】DynamicFormEditor 逻辑分离重构（纯展示层）
Wave: 2
Agent: gem-implementer
Priority: High
Effort: Large (280 lines)
Dependencies: [T-FE-01-schema-adapter]
BlockedBy: [T-FE-03-designer-integration]
```

### 核心职责

#### 第一部分：删除所有编辑交互逻辑
```javascript
// ❌ 删除这些方法和逻辑

// 1. 表格行操作
function addTableRow() { ... }    // ❌ 删除
function delTableRow(index) { ... } // ❌ 删除

// 2. 表单提交
function submitForm() { ... }     // ❌ 删除

// 3. 验证相关
function verify() { ... }        // ❌ 删除
function validate() { ... }      // ❌ 删除

// 4. 编辑事件监听
@change="onFieldChange"          // ❌ 删除
@blur="onFieldBlur"              // ❌ 删除
@input="onFieldInput"            // ❌ 删除

// 5. 编辑状态管理
data() {
  return {
    editingData: {},             // ❌ 删除
    tempData: {},                // ❌ 删除
    formErrors: {},              // ❌ 删除
    // ... 其他编辑状态
  }
}
```

#### 第二部分：保留所有展示逻辑
```javascript
// ✅ 保留这些方法和逻辑

// 1. 初始化
function onMounted() { ... }     // ✅ 保留

// 2. 数据加载
function loadAllOptions() { ... } // ✅ 保留

// 3. 规则获取（用于显示验证提示）
function getRules() { ... }      // ✅ 保留

// 4. 字段渲染和值显示
<el-input v-model="formData.field" readonly /> // ✅ 保留

// v-if 条件渲染
<div v-if="form.type === 'text'"> ... </div> // ✅ 保留
```

#### 第三部分：新增通信机制
```javascript
// 💬 新增 Props 以支持与 FormDesigner 的交互

props: {
  // 从 FormDesigner 接收编辑结果，实时同步显示
  editingData: {
    type: Object,
    default: () => ({})
  },
  
  // 标记是否为只读模式（禁用用户输入）
  readonly: {
    type: Boolean,
    default: true  // 默认只读（展示模式）
  }
}

// 在 watch 中监听 editingData 变化，更新显示
watch: {
  editingData(newVal) {
    // 更新显示，但不执行编辑逻辑
    this.formData = { ...newVal };
  }
}
```

### 验收标准

- [ ] 删除了所有 addTableRow/delTableRow/submitForm/verify 方法
- [ ] 删除了所有编辑相关的 @change/@blur/@input 事件
- [ ] 保留了 onMounted/loadAllOptions/getRules 方法
- [ ] 保留了所有 v-if/v-model 展示分支
- [ ] 新增 props:editingData 和 props:readonly
- [ ] 无编译错误，pnpm build 通过
- [ ] 无 linting 错误，pnpm lint 通过
- [ ] 输出「删除清单」供 T-FE-03 参考
- [ ] 输出「迁移清单」供 T-FE-03 实现

### 输出物

#### 1. 删除清单
```
编辑方法和逻辑已从 DynamicFormEditor.vue 删除：
- ✅ addTableRow() 方法已删除（行号：旧的 XXX）
- ✅ delTableRow() 方法已删除（行号：旧的 XXX）
- ✅ submitForm() 方法已删除（行号：旧的 XXX）
- ✅ verify() 方法已删除（行号：旧的 XXX）
- ✅ @change/@blur/@input 事件已删除（行号：旧的 XXX）
- ✅ editingData/tempData/formErrors 状态已删除（行号：旧的 XXX）
```

#### 2. 迁移清单
```
以下编辑方法需要迁移至 FormDesigner.vue：
1. addTableRow(tableFieldKey, newRow)
   - 功能：向指定表格字段添加新行
   - 参数：tableFieldKey (字段标识), newRow (新行数据)
   - 返回值：boolean (是否成功)
   - 原实现位置：DynamicFormEditor.vue:XXX-YYY 行

2. delTableRow(tableFieldKey, rowIndex)
   - 功能：删除指定表格字段的指定行
   - 参数：tableFieldKey (字段标识), rowIndex (行索引)
   - 返回值：boolean (是否成功)
   - 原实现位置：DynamicFormEditor.vue:XXX-YYY 行

3. submitForm()
   - 功能：提交表单（验证 + API 调用）
   - 依赖方法：verify()
   - API 调用：PUT /api/forms/:id
   - 原实现位置：DynamicFormEditor.vue:XXX-YYY 行

4. verify()
   - 功能：验证表单数据完整性和正确性
   - 返回值：{ valid: boolean, errors: {...} }
   - 原实现位置：DynamicFormEditor.vue:XXX-YYY 行
```

---

## 🔗 Task 依赖调整（新的关键链路）

### 旧链路（T-FE-03 直接接入 DynamicFormEditor）
```
T-FE-01-schema-adapter ─┐
                       ├─→ T-FE-03 ─→ T-RV-00 ─→ T-QA-01
T-FE-02-route-compat ──┘
```
**问题**: DynamicFormEditor 仍包含编辑逻辑，导致职责混淆。

### 新链路（T-FE-04 先重构，T-FE-03 再迁移）
```
T-FE-01-schema-adapter ──┐
                        ├─→ T-FE-04 (删除编辑逻辑，输出迁移清单)
                        │        │
                        │        ↓
T-FE-02-route-compat ───┤   T-FE-03 (接收迁移清单，完整实现编辑逻辑)
                        │        │
                        └────────┴─→ T-RV-00 (审查逻辑分离)
                                     │
                                     ↓
                                   T-QA-01 (验证编辑交互完整性)
```
**优势**:
- 职责清晰（展示 vs 编辑）
- 迁移可追踪（清单完整）
- 风险可控（分阶段验收）

---

## 📊 影响范围矩阵

### Task 修改影响

| Task | 类型 | 变更点 | 影响程度 |
|------|------|-------|---------|
| T-FE-03 | 依赖新增 | +T-FE-04 | 🔴 High |
| T-FE-03 | 职责扩大 | +迁移编辑逻辑 | 🔴 High |
| T-RV-00 | 审查标准更新 | "仅样式" → "逻辑分离" | 🟡 Medium |
| T-QA-01 | 验证场景新增 | +编辑交互验证 | 🟡 Medium |
| T-RV-01 | 审查重点升级 | 新增逻辑分离检查 | 🟡 Medium |
| T-DOC-01 | 文档范围扩大 | +约束变更说明 | 🟢 Low |

### 工作流影响

| 阶段 | 旧流程 | 新流程 | 变化 |
|------|--------|--------|-----|
| 实现 | T-FE-03 直接修改 DynamicFormEditor | T-FE-04 → T-FE-03 (串行) | 额外 1 个任务 |
| 代码审查 | 确认仅样式改动 | ✓ 逻辑分离<br/>✓ 方法迁移<br/>✓ 职责边界 | 新增 3 个检查项 |
| QA 验证 | 兼容性 + 路由 | + 编辑交互完整性 | 新增 4 个场景 |
| 文档 | 基础架构说明 | + 约束变更背景<br/>+ 迁移清单 | 新增 2 个章节 |

---

## 🎯 Wave 2 执行策略

### 执行时序图

```
Day 1-2: T-FE-04 执行（删除与重构）
  ├─ 第一步：删除所有编辑逻辑
  │   └─ 产出：纯展示的 DynamicFormEditor.vue
  │
  ├─ 第二步：新增通信 Props
  │   └─ props:editingData, props:readonly
  │
  ├─ 第三步：输出清单
  │   └─ 删除清单 + 迁移清单
  │
  └─ 第四步：验收
      └─ lint + build + code review

Day 3-5: T-FE-03 执行（迁移与接线）
  ├─ 第一步：接收 T-FE-04 的迁移清单
  │   └─ 明确各方法的新位置和职责
  │
  ├─ 第二步：实现迁移的编辑方法
  │   ├─ addTableRow() → FormDesigner
  │   ├─ delTableRow() → FormDesigner
  │   ├─ submitForm() → FormDesigner
  │   └─ verify() → FormDesigner
  │
  ├─ 第三步：建立通信机制
  │   └─ 通过 props:editingData 同步编辑结果
  │
  └─ 第四步：验收
      └─ lint + build + 手工验证编辑链路

Wave 2 后续：
  Day 6-7: T-BE-02 并行执行（后端兼容适配）
  Day 8: 团队对齐 + T-RV-00 preparation
```

### 质量关键点

**T-FE-04 交接 T-FE-03 的 DoD**
```
✅ DynamicFormEditor.vue 编译通过
✅ 无 linting 错误
✅ 所有编辑方法已删除（可通过 grep 验证）
✅ 所有展示逻辑已保留（可通过对比验证）
✅ 新增 props:editingData 和 props:readonly
✅ 输出的「迁移清单」明确列出每个方法的签名、功能、参数、返回值
✅ Code review 通过（重点：确认删除和保留的正确性）
```

**T-FE-03 接收 T-FE-04 的检查**
```
✅ 收到并理解「迁移清单」
✅ 逐一实现每个迁移的方法（无缺失）
✅ 实现的方法功能与原版本等同
✅ 通过 props:editingData 正确传递编辑结果给 DynamicFormEditor
✅ 编辑链路完整可测（表格行增删、字段修改、验证、提交）
✅ Code review 通过（重点：功能等同性、通信机制、编辑流程）
```

---

## 🚦 Pre-QA Gate (T-RV-00) 新审查标准

### 旧标准 ❌
```
□ DynamicFormEditor 中仅有样式相关改动
  └─ 问题：无法验证逻辑的删除和迁移是否完整
```

### 新标准 ✅
```
☑ 逻辑分离检查
  □ DynamicFormEditor 中 addTableRow/delTableRow/submitForm/verify 已删除
  □ DynamicFormEditor 中无编辑相关事件（@change/@blur/@input）
  □ DynamicFormEditor 仅包含 v-if/v-model 展示分支
  □ FormDesigner 中包含所有迁移的编辑方法
  └─ 验证方式：grep + diff + code review

☑ 冲突优先级检查（保持原有）
  □ FE/BE 都使用 schema > formSchema（仅缺失时回退）
  └─ 验证方式：单元测试断言

☑ Repository 零改动检查（保持原有）
  □ form_repository.go 无变动
  □ SQL schema 无变动
  └─ 验证方式：git diff

☑ 输出 gate-report.yaml
  ├─ 逻辑分离检查：PASS/FAIL
  ├─ 冲突优先级检查：PASS/FAIL
  ├─ Repository 零改动检查：PASS/FAIL
  └─ 阻断项清单（如有失败）
```

---

## 📝 Code Review Checklist (新)

### For T-FE-04
```
□ 编辑方法删除完整性
  ├─ [ ] addTableRow 已删除
  ├─ [ ] delTableRow 已删除
  ├─ [ ] submitForm 已删除
  ├─ [ ] verify 已删除
  └─ [ ] 其他编辑相关方法已删除

□ 展示逻辑保留完整性
  ├─ [ ] onMounted 已保留
  ├─ [ ] loadAllOptions 已保留
  ├─ [ ] getRules 已保留
  └─ [ ] v-if/v-model 分支已保留

□ 新增通信机制
  ├─ [ ] props:editingData 已定义
  ├─ [ ] props:readonly 已定义
  ├─ [ ] watch:editingData 已实现
  └─ [ ] readonly 模式逻辑已实现

□ 输出物质量
  ├─ [ ] 删除清单明确列出所有删除项
  └─ [ ] 迁移清单包含方法签名、功能、参数、返回值

□ 代码质量
  ├─ [ ] lint 无错误
  ├─ [ ] build 成功
  └─ [ ] 无遗留的编辑相关变量或函数
```

### For T-FE-03
```
□ 迁移完整性
  ├─ [ ] addTableRow 已实现
  ├─ [ ] delTableRow 已实现
  ├─ [ ] submitForm 已实现
  ├─ [ ] verify 已实现
  └─ [ ] 功能与原版本等同

□ 通信机制
  ├─ [ ] 正确使用 props:editingData
  ├─ [ ] 正确使用 props:readonly
  └─ [ ] 编辑结果同步显示

□ 编辑流程
  ├─ [ ] 表格行增删能正常工作
  ├─ [ ] 字段修改能正常工作
  ├─ [ ] 验证能正常工作
  └─ [ ] 提交能正常工作

□ 代码质量
  ├─ [ ] lint 无错误
  ├─ [ ] build 成功
  └─ [ ] 编辑链路无新增错误
```

---

## 📊 Task 工作量估算更新

### T-FE-04 (新增)
```
分解工作：
  1. 方法删除分析        : 30 分钟
  2. 编辑逻辑识别和删除   : 2 小时
  3. 新增通信 Props      : 30 分钟
  4. watch 实现          : 30 分钟
  5. 清单输出            : 1 小时
  6. Code Review 迭代   : 1 小时
  
总计 : ~6 小时 (280 lines)
```

### T-FE-03 (修改)
```
增量工作（在原有基础上）：
  1. 迁移清单理解        : 1 小时
  2. 编辑方法实现        : 3 小时
  3. 通信机制接线        : 1 小时
  4. 测试 & Code Review : 2 小时
  
增量总计 : ~7 小时 (+120 lines)
原有工作 : ~8 小时 (+180 lines)
修改后总计 : ~15 小时 (+300 lines)
```

---

## ✅ 核心质量指标

| 指标 | 旧值 | 新值 | 验收标准 |
|------|------|------|--------|
| DynamicFormEditor 的职责 | 混淆（展示+编辑） | 明确（仅展示） | grep 检查 0 条编辑逻辑 |
| 编辑逻辑的位置 | 分散 | 集中（FormDesigner） | grep 检查 FormDesigner 包含所有方法 |
| 方法迁移的追踪 | 无 | 有清单 | 迁移清单覆盖 100% 的方法 |
| 审查标准 | 样式改动 | 逻辑分离 | T-RV-00 gate report PASS |
| 验证场景 | 兼容性 | +编辑交互 | T-QA-01 覆盖表格行增删等 |

---

这份详细对比应该能帮助团队清晰理解约束变更的影响范围和执行方式。
