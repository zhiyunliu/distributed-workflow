# Feature Specification: 统一元模型与存储

**Feature Branch**: `001-unified-model-storage`  
**Created**: 2026-06-16  
**Status**: Draft  
**Input**: 根据 `docs/requirement.md` 拆分统一流程编排平台规格。

## User Scenarios & Testing *(mandatory)*

### User Story 1 - 保存标准流程定义 (Priority: P1)

作为流程设计人员，我希望系统使用同一套流程定义模型保存工作流和规则链，使设计器、引擎和审计都读取同一份结构化事实。

**Why this priority**: 统一元模型是所有后续引擎、设计器和 API 的基础，没有它无法可靠规划其他功能。

**Independent Test**: 提交包含 `nodes` 与 `connections` 的 `WORKFLOW` 和 `RULE_CHAIN` 模型，系统能够保存为版本化流程定义，并返回稳定的定义标识、版本和状态。

**Acceptance Scenarios**:

1. **Given** 一个合法 `RULE_CHAIN` 模型，**When** 用户保存草稿，**Then** 系统创建 `sch_flow_definition` 记录并保留完整 `model_content`。
2. **Given** 一个合法 `WORKFLOW` 模型，**When** 用户保存新版本，**Then** 系统在同一 `flow_code` 下递增 `version`。

---

### User Story 2 - 校验模型结构 (Priority: P1)

作为平台维护人员，我希望系统在保存和发布前校验模型结构，避免不完整、跨模式混用或字段命名不合规的定义进入引擎。

**Why this priority**: 模型错误进入存储会放大到路由、引擎执行、审计和设计器回显。

**Independent Test**: 提交缺失节点、缺失连线目标、驼峰 JSON 键名或规则链循环结构，系统返回明确的错误码、子码和中文消息。

**Acceptance Scenarios**:

1. **Given** `connections` 中存在不存在的 `to_id`，**When** 用户保存模型，**Then** 系统拒绝保存并返回模型校验错误。
2. **Given** `RULE_CHAIN` 中存在环，**When** 用户发布模型，**Then** 系统拒绝发布并指出规则链必须为 DAG。

---

### User Story 3 - 映射工作流隐式连线 (Priority: P2)

作为工作流引擎开发人员，我希望工作流的隐式后继关系能够无损映射为统一显式边集，方便持久化、版本 diff 和设计器渲染。

**Why this priority**: 工作流能力可以晚于规则链落地，但模型需要提前兼容。

**Independent Test**: 输入包含普通步骤、条件分支、多分支默认路径的工作流模型，系统能够生成或解析等价的 `connections`。

**Acceptance Scenarios**:

1. **Given** 一个条件分支节点，**When** 系统转换模型，**Then** 生成 `relation_type=condition` 的 True/False 连线。
2. **Given** 一个默认分支，**When** 系统转换模型，**Then** 对应连线包含 `is_default=true`。

### Edge Cases

- 模型缺少 `flow_code`、`flow_type`、`nodes` 或 `connections` 时必须拒绝保存。
- `flow_type` 不是 `WORKFLOW` 或 `RULE_CHAIN` 时必须拒绝保存。
- 同一版本内节点 ID 或连线 ID 重复时必须拒绝保存。
- `connections` 引用不存在的 `from_id` 或 `to_id` 时必须拒绝保存。
- 公开 JSON 出现 `fromId`、`toId`、`relationType`、`isDefault` 等驼峰键名时必须拒绝或规范化并记录警告。
- `model_content` 不是合法 JSON 时必须拒绝入库。

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST define a unified flow definition model containing `id`, `flow_code`, `name`, `flow_type`, `version`, `variables`, `settings`, `nodes`, and `connections`.
- **FR-002**: System MUST store every flow definition in `sch_flow_definition` with `flow_code`, `flow_type`, `version`, `model_content`, `status`, `create_time`, and `update_time`.
- **FR-003**: System MUST use lowercase snake_case for all public JSON keys and persisted model examples.
- **FR-004**: System MUST treat `connections` as the only persisted representation of flow edges.
- **FR-005**: System MUST validate node IDs, connection IDs, edge references, flow type, version rules, JSON validity, and mode-specific constraints before save or publish.
- **FR-006**: System MUST support version increments under the same `flow_code` and prevent accidental overwrite of published versions.
- **FR-007**: System MUST preserve a full `model_content` JSON snapshot for import, export, replay, audit, and design-time rendering.
- **FR-008**: System MUST expose validation errors through stable `code`, `sub_code`, and Chinese `message` values.

### Key Entities *(include if feature involves data)*

- **Flow Definition**: Versioned metadata and full model JSON for a workflow or rule chain.
- **Flow Node**: Model node with `id`, `name`, `type`, `position`, and `config`.
- **Flow Connection**: Explicit edge with `id`, `from_id`, `to_id`, `label`, `relation_type`, `condition`, and `is_default`.
- **Flow Variable**: Named input or runtime variable with `name`, `type`, and `default`.
- **Validation Result**: Structured result with `valid`, `errors`, `warnings`, and optional `normalized_model`.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 100% of saved definitions contain `nodes` and `connections` as explicit collections.
- **SC-002**: Invalid model submissions return deterministic error responses with `sub_code`.
- **SC-003**: The same `flow_code` can maintain at least 10 historical versions without overwriting published definitions.
- **SC-004**: Design-time rendering can load a saved model without deriving edges from node-specific hidden fields.

## Assumptions

- SQL Server is the authoritative persistent store for flow definitions.
- `status=0` means disabled or draft-like non-executable state, and `status=1` means enabled or published executable state unless later specifications refine status values.
- `model_content` stores the full JSON while common query fields remain structured columns.
- Model validation is shared by backend save, publish, import, and debug flows.

## Constitution Alignment *(mandatory)*

- **Flow Type**: INFRASTRUCTURE
- **Unified Model Impact**: Defines `nodes`, `connections`, variables, settings, versioning, and model validation.
- **Engine Boundary**: Stores a shared model but does not share runtime context between engines.
- **Data & Audit**: Creates or updates `sch_flow_definition`; stores JSON in `nvarchar(max)`; keeps structured version and status columns.
- **API & Response Contract**: Uses snake_case payloads and stable response code/sub-code constants.
- **Verification Evidence**: Model validation tests, versioning tests, SQL Server schema checks, and import/export model round-trip checks.