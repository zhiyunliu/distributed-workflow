# 约束变更重新规划总结
**计划ID**: 20260511-replace-formcomponents-with-dynamicformeditor  
**重新规划日期**: 2026-05-11 16:00:00+08:00  
**规划者**: gem-planner  
**新计划位置**: `plan-v2-constraint-relaxation.yaml`

---

## 📋 变更背景

### 原始约束
```
"DynamicFormEditor 业务逻辑保持不变，仅允许样式相关代码调整（CSS/样式变量/布局适配）；
禁止修改字段渲染、值处理、验证等逻辑。"
```

### 当前阻塞
- **任务**: T-RV-00-pre-qa-boundary-gate
- **原因**: DynamicFormEditor 包含编辑逻辑（addTableRow/delTableRow/submitForm），违反"仅样式可改"约束
- **后续影响**: T-QA-01 无法进行

### 用户授权变更
```
"允许我重构 DynamicFormEditor"
=> 新约束：允许完全重构以分离显示层与编辑层逻辑
```

---

## 🎯 核心规划决策

### 1️⃣ 新的分层架构

#### 旧架构（问题）
```
DynamicFormEditor（职责混淆）
├─ 展示逻辑 ✓
├─ 编辑逻辑 ✗（违反约束，但无处可放）
└─ 状态管理（编辑状态）
```

#### 新架构（解决）
```
编排层（FormDesigner）
├─ 编辑交互逻辑 ← addTableRow/delTableRow/submitForm/verify（迁移）
├─ 状态管理（编辑状态）
└─ API 调用

     ↓ schema via adapter ↓

展示层（DynamicFormEditor）
├─ 字段渲染（v-if 分支）✓
├─ 值显示（v-model 绑定）✓
├─ 初始化（onMounted）✓
├─ 选项加载（loadAllOptions）✓
└─ 验证规则显示（getRules）✓
   （删除）✗ addTableRow/delTableRow/submitForm/verify
```

### 2️⃣ 删除与保留清单

#### DynamicFormEditor 中要删除的
| 方法/逻辑 | 原位置 | 新位置 | 原因 |
|----------|------|------|------|
| `addTableRow` | DynamicFormEditor | FormDesigner | 编辑交互逻辑 |
| `delTableRow` | DynamicFormEditor | FormDesigner | 编辑交互逻辑 |
| `submitForm` | DynamicFormEditor | FormDesigner | 编辑交互逻辑 |
| `verify/validate` | DynamicFormEditor | FormDesigner | 验证逻辑 |
| `@change/@blur/@input` 编辑事件 | DynamicFormEditor | FormDesigner | 编辑事件处理 |
| 编辑状态管理 | DynamicFormEditor.data | FormDesigner.data | 编辑状态 |

#### DynamicFormEditor 中要保留的
| 方法/逻辑 | 保留原因 |
|----------|--------|
| `onMounted` | 初始化显示数据 |
| `loadAllOptions` | 填充下拉选项用于展示 |
| `getRules` | 获取验证规则供显示提示使用 |
| 字段渲染 `v-if` 分支 | 字段展示 |
| 值绑定 `v-model` | 值显示与同步 |

#### DynamicFormEditor 中要新增的
| Props | 目的 |
|------|-----|
| `editingData?` | 接收 FormDesigner 的编辑结果，实时显示编辑后的值 |
| `readonly` | 标记只读模式，禁用用户输入 |

---

## 🔄 Task 调整

### 新增任务
#### **T-FE-04-dynamicformeditor-refactor** ⭐ (新)
| 属性 | 值 |
|-----|-----|
| Wave | 2 |
| 优先级 | High |
| 工作量 | Large (280 lines) |
| 目标 | 删除所有编辑交互逻辑，实现纯展示层 |
| 产物 | DynamicFormEditor（纯展示）+ 删除清单 + 迁移清单 |
| 依赖 | T-FE-01-schema-adapter |
| 阻塞 | T-FE-03-designer-integration |

**关键验收**:
- ✅ 删除 addTableRow/delTableRow/submitForm/verify
- ✅ 保留 onMounted/loadAllOptions/getRules
- ✅ 仅 v-if/v-model 展示分支，无编辑事件
- ✅ 产出「删除清单 + 迁移清单」供 T-FE-03 使用

### 修改任务
#### **T-FE-03-designer-integration** (修改)
| 变更项 | 旧值 | 新值 |
|-------|------|------|
| 依赖 | T-FE-01, T-FE-02 | T-FE-01, T-FE-02, **T-FE-04** |
| 职责 | 接入 DynamicFormEditor 预览 | 接入 + **接收迁移的编辑逻辑** |
| 工作量 | Medium (260L) | **Large (300L)** |
| 新职责清单 | 无 | addTableRow/delTableRow/submitForm/verify 在此实现 |

**关键变更**:
- 接收 T-FE-04 输出的「迁移清单」
- 完整实现所有迁移的编辑方法（无功能缺失）
- 通过 `props:editingData` 与 DynamicFormEditor 同步值
- 保持编辑能力完整

#### **T-RV-00-pre-qa-boundary-gate** (修改)
| 变更项 | 旧值 | 新值 |
|-------|------|------|
| 审查标准 | "仅样式改动" | **"纯展示 + 编辑逻辑完整迁移"** |
| 依赖 | T-FE-03, T-BE-02 | T-FE-03, **T-FE-04**, T-BE-02 |
| 新检查项 | 无 | ✓ DynamicFormEditor 无编辑方法<br/>✓ FormDesigner 包含迁移方法<br/>✓ 编辑事件完全移出 |

#### **T-QA-01-compat-regression** (修改)
| 变更项 | 旧值 | 新值 |
|-------|------|------|
| 新增验证场景 | 无 | **表格行增删、编辑值显示同步** |
| 依赖 | 4 | **5 (新增 T-FE-04)** |
| 验证重点 | 兼容性 | **兼容性 + 编辑交互完整性** |

#### **T-RV-01-contract-review** (修改)
| 变更项 | 旧值 | 新值 |
|-------|------|------|
| 审查重点 | 样式改动 | **逻辑分离 + 方法迁移** |
| 新检查项 | 无 | ✓ 逻辑分离完整性<br/>✓ 编辑方法迁移验证 |

#### **T-DOC-01-migration-doc** (修改)
| 变更项 | 旧值 | 新值 |
|-------|------|------|
| 文档范围 | 架构说明 | **+ 约束变更说明<br/>+ 逻辑分离清单<br/>+ 编辑方法迁移位置<br/>+ 职责边界明确化** |
| 覆盖矩阵 | 6 项 | **10 项** |

---

## 📊 DAG 变更

### 新的依赖图
```
Wave 1:
  T-BE-00 ─┐
           ├─→ T-BE-02
  T-BE-01 ─┘
  
  T-FE-01 ─┐
           ├─→ T-FE-04 (NEW)
  T-FE-02 ─┘

Wave 2:
  T-FE-04 (NEW) ───┐
                   ├─→ T-FE-03
  T-FE-01 ─────────┘
  T-FE-02 ─────────┘
  
  T-BE-00 ─┐
           ├─→ T-BE-02
  T-BE-01 ─┘

Wave 3:
  T-FE-03 ──┐
            ├─→ T-RV-00
  T-FE-04 ──┤
            │
  T-BE-02 ──┘

Wave 4:
  T-FE-03 ──┐
  T-FE-04 ──┤
  T-BE-00 ──┤
  T-BE-02 ──┤
            ├─→ T-QA-01 ─→ T-RV-01
  T-RV-00 ──┘

Wave 5:
  T-RV-01 ─→ T-DOC-01
```

---

## 🎨 Contracts 调整

### 新增 Contracts
```yaml
from_task: T-FE-04-dynamicformeditor-refactor
to_task: T-FE-03-designer-integration
interface: "DynamicFormEditor 已删除所有编辑逻辑，仅支持 v-model:schema 展示与 props:editingData 值同步"
format: "Vue component interface"
```

### 修改 Contracts
```yaml
# T-FE-03 与 DynamicFormEditor 的交互
from_task: T-FE-03-designer-integration
to_task: (implicit) DynamicFormEditor component
interface: "FormDesigner 通过 props:editingData 同步编辑结果，通过 props:readonly 标记只读模式"
format: "Vue component props interface"
```

---

## ⚠️ 风险管理（新增风险）

### 新增高风险
| ID | 风险场景 | 可能性 | 影响 | Mitigation |
|-------|---------|--------|------|-----------|
| R-01 | DynamicFormEditor 重构时遗漏关键显示逻辑 | Medium | Critical | 明确保留白名单 (onMounted/loadAllOptions/getRules) + code review |
| R-02 | 编辑逻辑迁移至 FormDesigner 时遗漏验证或错误处理 | Medium | High | T-FE-04 输出「迁移清单」，T-FE-03 强制校验完整性 |
| R-03 | 编辑交互逻辑分散，导致职责边界不清 | High | High | T-RV-00 与 T-RV-01 的边界审查 + T-DOC-01 明确文档 |
| R-04 | 删除的编辑方法未完整迁移导致功能缺失 | High | High | T-FE-03 依赖 T-FE-04，逐一校验迁移结果 |

### 原有风险（更新）
| ID | 原风险 | 新 Mitigation |
|-------|-------|---------------|
| 越界改动 | 原为"不改逻辑" | **新为"逻辑完全迁移"**，T-RV-00 校验迁移完整性 |

---

## 📋 Minimum DoD Checklist 变更

### 新增项
```yaml
- "【约束变更】DynamicFormEditor.vue 已完全重构为纯显示层，
  不包含任何编辑交互逻辑，仅包含 v-if/v-model 展示分支
  （删除 addTableRow/delTableRow/submitForm 等编辑方法）。"

- "编辑交互逻辑（字段增删改、表格行增删、值绑定、验证、表单提交）
  全部由 FormDesigner 在编排层维护，不分散到 DynamicFormEditor。"

- "schema-adapter.ts 中的双向映射保持不变（已验证）"
```

### 修改项
```yaml
# 旧
- "DynamicFormEditor 业务逻辑保持不变，仅允许样式相关代码调整..."

# 新
- "【约束变更】DynamicFormEditor 已完全重构为纯展示层..."
```

---

## 📈 工作量对比

| 任务 | 旧工作量 | 新工作量 | 变化 | 原因 |
|------|---------|---------|-----|------|
| T-FE-03 | Medium (260L) | Large (300L) | +40L | 接收编辑逻辑迁移 |
| T-FE-04 | N/A (不存在) | Large (280L) | NEW | 新增逻辑分离重构 |
| T-RV-00 | Small (80L) | Medium (150L) | +70L | 新增逻辑分离审查 |
| T-QA-01 | Medium (120L) | Large (180L) | +60L | 新增编辑交互验证场景 |
| T-RV-01 | Small (80L) | Medium (120L) | +40L | 新增逻辑分离验证 |
| T-DOC-01 | Small (150L) | Medium (200L) | +50L | 新增约束变更和逻辑分离说明 |

**总计新增**: ~490 行代码 & 测试，涉及 6 个任务

---

## 🚀 执行路径

### 推荐执行顺序
1. **确认 open questions** (5-10 mins)
   - DynamicFormEditor 中编辑逻辑的具体迁移方案
   - 是否需要支持暂存编辑状态

2. **执行 Wave 1-2** (并行)
   - Wave 1 照旧（4 个后端前端基础任务）
   - Wave 2 新流程：**T-FE-04 → T-FE-03** (串行，确保迁移完整)

3. **执行 Wave 3 (gate)**
   - T-RV-00 按新标准执行（逻辑分离验证）

4. **执行 Wave 4** (并行)
   - T-QA-01 新增编辑交互验证场景
   - T-RV-01 新增逻辑分离验证

5. **执行 Wave 5**
   - T-DOC-01 产出完整的架构与迁移文档

---

## ✅ 验收预检清单

在进入实现前，确认以下事项：

- [ ] 理解新的分层架构（编排 + 适配 + 展示）
- [ ] 同意删除清单（addTableRow/delTableRow/submitForm/verify）
- [ ] 同意保留清单（onMounted/loadAllOptions/getRules + v-if/v-model）
- [ ] 同意新的约束变更（允许完全重构）
- [ ] 确认 T-FE-04 的输出格式（删除清单 + 迁移清单）
- [ ] 确认 T-FE-03 的接收方式（如何使用迁移清单）
- [ ] 确认 T-RV-00 的新审查标准（逻辑分离 vs 样式改动）
- [ ] 确认 T-QA-01 的新验证场景（编辑交互）
- [ ] 确认项目方支持额外工作量（+490 行代码）

---

## 📁 输出物

**新计划文件**: `plan-v2-constraint-relaxation.yaml`
- 11 个任务（新增 T-FE-04）
- 5 个 Wave
- 更新的 contracts、risks、mitigations
- 更新的 minimum_dod_checklist
- 更新的验收标准

**参考文件**:
- 原计划: `plan.yaml` (保留备份用途)
- 研究报告: `research_findings_*.yaml` (保持不变)

---

## 🎯 后续步骤

### 立即行动
1. **审批新计划** → 确认所有变更点
2. **澄清 open questions** → 确定编辑逻辑迁移的具体方案
3. **启动 T-FE-04 规划会** → 明确删除和保留的范围

### 准备阶段
1. 准备测试环境（表格行增删、字段验证等编辑场景）
2. 准备 code review checklist（重点: 逻辑分离完整性）
3. 准备文档模板（新增: 约束变更说明、迁移清单）

---

## 📝 Notes

- 本重新规划基于用户的明确授权变更约束
- 所有变更点已在新的 plan-v2-constraint-relaxation.yaml 中明确记录
- 建议在启动实现前再做一次**风险评估和团队对齐**
- 新增工作量约 490 行代码，建议预留 1.5-2 倍的时间用于 code review 和测试

---

**规划完成时间**: 2026-05-11 16:30:00+08:00  
**规划版本**: v2 (约束变更)
