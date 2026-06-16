# Feature Specification: 统一可视化流程设计器

**Feature Branch**: `006-visual-flow-designer`  
**Created**: 2026-06-16  
**Status**: Draft  
**Input**: 根据 `docs/requirement.md` 拆分统一流程编排平台规格。

## User Scenarios & Testing *(mandatory)*

### User Story 1 - 双模式流程建模 (Priority: P1)

作为流程设计人员，我希望在同一设计器中切换工作流和规则链模式，并使用一致的拖拽、连线、属性配置体验完成建模。

**Why this priority**: 统一设计入口是项目降低学习和运维成本的核心目标。

**Independent Test**: 在设计器中分别创建工作流和规则链草稿，保存后后端收到同一统一模型结构。

**Acceptance Scenarios**:

1. **Given** 用户选择 `RULE_CHAIN` 模式，**When** 打开节点工具箱，**Then** 只显示过滤、转换、动作、集成、日志和启动工作流等规则链节点。
2. **Given** 用户选择 `WORKFLOW` 模式，**When** 打开节点工具箱，**Then** 只显示人工任务、网关、事件、子流程、补偿和规则链调用等工作流节点。

---

### User Story 2 - 实时校验和保存版本 (Priority: P1)

作为设计人员，我希望设计器在保存前实时提示节点缺失、连线非法、模式混用和规则链环路，避免反复提交后端才发现错误。

**Why this priority**: 可视化建模必须与统一模型校验紧密结合，才能降低错误成本。

**Independent Test**: 在规则链模式创建循环连线或添加人工任务节点，设计器即时提示并阻止保存或发布。

**Acceptance Scenarios**:

1. **Given** 规则链画布中出现环，**When** 用户尝试保存，**Then** 设计器展示 DAG 错误并调用后端二次校验。
2. **Given** 用户保存合法草稿，**When** 后端返回版本信息，**Then** 设计器显示当前 `flow_code`、`version` 和状态。

---

### User Story 3 - 预览调试和导入导出 (Priority: P2)

作为设计人员，我希望能输入模拟数据预览执行路径、查看节点输出，并导入导出统一 JSON 模型用于评审、备份和迁移。

**Why this priority**: 调试和迁移能力提升设计效率，但可以在基础建模保存后增强。

**Independent Test**: 使用一个草稿模型执行调试预览，设计器展示路径、节点输出、错误和警告；导出的 JSON 可再次导入并保持一致。

**Acceptance Scenarios**:

1. **Given** 一个规则链草稿，**When** 用户输入模拟载荷并点击调试，**Then** 设计器显示路由路径和节点输出。
2. **Given** 一个导出的模型文件，**When** 用户导入，**Then** 设计器恢复节点、连线、位置和配置。

### Edge Cases

- 模式切换时不得保留另一模式专属节点工具箱。
- 规则链模式不得允许循环连线或人工任务节点。
- 工作流模式允许回退和循环，但仍需校验节点和连线引用合法。
- 导入 JSON 缺少必要字段或使用驼峰键名时必须提示错误。
- 后端校验结果与前端校验不一致时以前端显示后端错误为准。
- 调试预览超时时必须展示可理解的错误信息。

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST provide one visual canvas that supports both `WORKFLOW` and `RULE_CHAIN` modes.
- **FR-002**: System MUST isolate node toolbox contents by selected `flow_type`.
- **FR-003**: System MUST render and edit `nodes` and `connections` as the primary graph model.
- **FR-004**: System MUST store node positions in `position` using snake_case JSON payloads where public fields are exposed.
- **FR-005**: System MUST provide property panels for node name, node type, conditions, permissions, timeout, retry, and input/output mappings as applicable.
- **FR-006**: System MUST perform frontend validation before save and call backend model validation before publish.
- **FR-007**: System MUST support draft save, version display, publish request, debug preview, import, and export through unified APIs.
- **FR-008**: System MUST use Vue 3, Vite, TypeScript strict mode, Element Plus, Pinia, and Axios wrappers under `management`.

### Key Entities *(include if feature involves data)*

- **Designer Flow Model**: Frontend representation of the unified model.
- **Canvas Node**: Visual node with type, position, label, config, and validation state.
- **Canvas Connection**: Visual edge with source, target, label, relation type, condition, and default marker.
- **Node Toolbox**: Mode-specific list of available node types.
- **Property Panel**: Form used to edit node and connection configuration.
- **Debug Preview Result**: Route path, node outputs, warnings, errors, and trace id.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Users can create and save a simple rule chain without manually editing JSON.
- **SC-002**: Users can create and save a simple workflow without manually editing JSON.
- **SC-003**: Mode-specific invalid nodes are blocked or flagged before publish.
- **SC-004**: Exported JSON can be imported back and render the same graph structure.

## Assumptions

- The visual graph library will be AntV X6 or LogicFlow; final selection may be confirmed during planning.
- The first usable designer can support core node types and expand advanced nodes later.
- Management UI lives under `management` and uses project-standard Axios API wrappers.
- Backend remains the authoritative validator even when frontend validation exists.

## Constitution Alignment *(mandatory)*

- **Flow Type**: MANAGEMENT
- **Unified Model Impact**: Reads and writes `nodes`, `connections`, positions, settings, and node config.
- **Engine Boundary**: Designer configures engine type but never calls engine internals directly.
- **Data & Audit**: Save, publish, import, export, and debug actions are audited through backend APIs.
- **API & Response Contract**: Uses snake_case API payloads and unified response handling.
- **Verification Evidence**: UI validation checks, import/export round-trip checks, API wrapper checks, and debug preview checks.