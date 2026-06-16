# Feature Specification: 引擎路由与统一 API

**Feature Branch**: `002-engine-router-api`  
**Created**: 2026-06-16  
**Status**: Draft  
**Input**: 根据 `docs/requirement.md` 拆分统一流程编排平台规格。

## User Scenarios & Testing *(mandatory)*

### User Story 1 - 统一管理流程定义 (Priority: P1)

作为流程管理员，我希望通过统一 API 创建、保存、发布、停用、查询和导出流程定义，而不需要关心底层属于工作流还是规则链。

**Why this priority**: API 和路由层是设计器、引擎执行和后续管理功能的入口。

**Independent Test**: 使用同一组 API 分别保存、发布、查询、导出 `WORKFLOW` 与 `RULE_CHAIN` 定义。

**Acceptance Scenarios**:

1. **Given** 一个流程草稿，**When** 用户调用保存接口，**Then** 系统保存版本化定义并返回统一响应。
2. **Given** 一个已发布定义，**When** 用户调用导出接口，**Then** 系统返回包含 `flow_code`、`version` 和 `model_content` 的 JSON。

---

### User Story 2 - 按流程类型路由执行请求 (Priority: P1)

作为业务系统，我希望向统一执行入口提交 `flow_code`、版本策略和输入数据，由平台自动路由到正确引擎执行。

**Why this priority**: 双引擎统一入口能降低业务接入成本，同时保持内核隔离。

**Independent Test**: 分别调用 `WORKFLOW` 与 `RULE_CHAIN` 的执行入口，确认路由层选择对应引擎且不暴露内部上下文。

**Acceptance Scenarios**:

1. **Given** 已发布的规则链定义，**When** 业务请求提交事件载荷，**Then** 路由层调用规则链执行器。
2. **Given** 已发布的工作流定义，**When** 业务请求启动流程，**Then** 路由层创建或恢复工作流实例。

---

### User Story 3 - 调试和模型校验入口 (Priority: P2)

作为设计人员，我希望在发布前通过统一 API 校验和调试模型，提前发现连线、节点、权限或 DAG 问题。

**Why this priority**: 调试能力提升设计效率，但可在保存发布基础能力后完善。

**Independent Test**: 对草稿模型调用校验和调试预览接口，系统返回路径、节点输出、错误和警告。

**Acceptance Scenarios**:

1. **Given** 一个未发布草稿，**When** 用户请求调试预览，**Then** 系统在不改变发布状态的情况下返回模拟结果。
2. **Given** 一个模型存在错误，**When** 用户请求校验，**Then** 系统返回 `valid=false` 和错误列表。

### Edge Cases

- 请求的 `flow_code` 不存在时返回定义不存在错误。
- 指定版本已停用时不得执行。
- 多个版本存在时必须按明确版本策略选择，不得随机选择。
- 无权限用户不得保存、发布、停用、调试或执行敏感流程。
- 发布期间发生版本冲突时必须返回冲突错误。
- 执行入口不得允许调用方指定引擎内部当前节点。

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST provide unified flow definition APIs for save draft, validate model, publish, disable, query, detail, export, import, debug preview, and execute.
- **FR-002**: System MUST route execution by `flow_type`, `flow_code`, version, status, permission, and publish state.
- **FR-003**: System MUST keep HTTP query operations as GET and create/update/delete/publish/execute/debug operations as POST.
- **FR-004**: System MUST return unified response shapes with `code`, `message`, optional `data`, and `sub_code` on errors.
- **FR-005**: System MUST define all response codes and sub-codes in `constants/respcode` and `constants/subcode`.
- **FR-006**: System MUST use glue APIs for routing, binding, middleware, logging, and dependency composition.
- **FR-007**: System MUST enforce permission checks before publish, disable, execute, debug, import, and export operations.
- **FR-008**: System MUST pass `trace_id` through execution and debug calls.

### Key Entities *(include if feature involves data)*

- **Flow Request**: API payload with `flow_code`, `flow_type`, `version`, `model_content`, or execution input.
- **Version Policy**: Rule for selecting latest published, specified version, or draft model.
- **Engine Route**: Decision record that maps a request to workflow, rule-chain, or validation-only handling.
- **API Response**: Standard `code`, `message`, optional `data`, and error `sub_code`.
- **Permission Context**: Caller identity, roles, operation, and flow scope.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: All public APIs use snake_case request and response fields.
- **SC-002**: Routing tests prove `WORKFLOW` and `RULE_CHAIN` requests reach different engine adapters.
- **SC-003**: Unauthorized publish or execute attempts are rejected with stable error responses.
- **SC-004**: A request can be traced from API entry to route decision and engine invocation through `trace_id`.

## Assumptions

- Authentication is provided by existing or planned glue-compatible middleware.
- Permission model can start with operation-level checks and later expand to tenant or organization scopes.
- Import/export use the unified model from spec 001.
- Execution adapters call engine interfaces but do not own engine internals.

## Constitution Alignment *(mandatory)*

- **Flow Type**: INFRASTRUCTURE
- **Unified Model Impact**: Reads and writes versioned flow definitions; does not redefine schema.
- **Engine Boundary**: Routes by adapter; never shares engine runtime context.
- **Data & Audit**: Records publish, disable, execute, debug, import, and export audit events.
- **API & Response Contract**: Enforces GET/POST rules, snake_case payloads, and response constants.
- **Verification Evidence**: API contract tests, router decision tests, permission tests, and trace propagation checks.