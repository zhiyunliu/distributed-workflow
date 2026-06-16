# Feature Specification: 跨引擎标准联动

**Feature Branch**: `005-cross-engine-linkage`  
**Created**: 2026-06-16  
**Status**: Draft  
**Input**: 根据 `docs/requirement.md` 拆分统一流程编排平台规格。

## User Scenarios & Testing *(mandatory)*

### User Story 1 - 工作流调用规则链 (Priority: P1)

作为流程设计人员，我希望在工作流中配置规则链调用节点，用审批上下文同步调用规则链完成校验或计算，然后将结果写回流程变量。

**Why this priority**: 这是两类引擎能力互补的主要业务路径之一。

**Independent Test**: 工作流执行到规则链调用节点时，系统通过路由层调用规则链并把返回结果写入工作流变量。

**Acceptance Scenarios**:

1. **Given** 工作流包含规则链调用节点，**When** 执行到该节点，**Then** 系统提交 `flow_code`、版本策略、输入变量和 `trace_id` 到规则链。
2. **Given** 规则链返回成功结果，**When** 调用完成，**Then** 工作流将结果写回变量并继续流转。

---

### User Story 2 - 规则链启动工作流 (Priority: P1)

作为事件处理人员，我希望规则链在满足条件时通过启动工作流节点创建工单、审批或订单流程实例。

**Why this priority**: 这是事件驱动场景进入长周期业务流程的关键路径。

**Independent Test**: 规则链匹配条件后调用标准节点，系统创建或恢复对应工作流实例并返回实例信息。

**Acceptance Scenarios**:

1. **Given** 规则链包含启动工作流节点，**When** 消息满足条件，**Then** 系统通过路由层创建工作流实例。
2. **Given** 启动请求包含业务键，**When** 重复消息到达，**Then** 系统按幂等策略避免重复创建实例。

---

### User Story 3 - 跨引擎错误治理 (Priority: P2)

作为运维人员，我希望跨引擎调用失败、超时或重复执行时有清晰的重试、补偿、终止和审计策略。

**Why this priority**: 联动失败会跨越两个引擎，需要标准化治理避免问题难以追踪。

**Independent Test**: 模拟目标引擎超时、目标定义不存在、权限不足和重复调用，系统按节点配置处理并记录审计。

**Acceptance Scenarios**:

1. **Given** 目标规则链超时，**When** 工作流调用节点失败，**Then** 系统按节点配置重试、补偿、终止或转人工。
2. **Given** 规则链重复触发相同业务键，**When** 启动工作流，**Then** 系统返回已有实例或拒绝重复启动。

### Edge Cases

- 跨引擎调用不得共享内部上下文或直接设置对方当前节点。
- 目标流程定义不存在、未发布或已停用时必须失败并记录审计。
- 版本策略不明确时不得调用随机版本。
- 调用链必须传递同一个 `trace_id` 或派生链路标识。
- 重试可能导致重复执行，必须支持业务键或幂等键。
- 调用方权限和目标流程执行权限都必须校验。

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST implement cross-engine linkage only through standard nodes.
- **FR-002**: System MUST provide a workflow standard node that invokes a rule chain through the engine router.
- **FR-003**: System MUST provide a rule-chain standard node that starts or requests a workflow instance through the engine router.
- **FR-004**: System MUST pass `flow_code`, version policy, payload or variables, `trace_id`, and optional idempotency key in cross-engine calls.
- **FR-005**: System MUST write standard node input, output, status, duration, and error details to node logs.
- **FR-006**: System MUST support configurable handling for timeout, retry, compensation, termination, and manual intervention.
- **FR-007**: System MUST enforce permission checks on both caller operation and target flow operation.
- **FR-008**: System MUST prevent direct cross-engine node jumps and runtime context sharing.

### Key Entities *(include if feature involves data)*

- **Rule-chain Call Node**: Workflow node that invokes a rule-chain definition.
- **Workflow Start Node**: Rule-chain node that starts or requests a workflow instance.
- **Cross-engine Request**: Standard payload carrying target flow, version policy, variables, trace id, and idempotency key.
- **Cross-engine Result**: Standard output containing success flag, result data, target instance or execution id, and error data.
- **Idempotency Key**: Business key used to prevent duplicate workflow starts or repeated side effects.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A workflow can call a rule chain and use its result in a subsequent branch.
- **SC-002**: A rule chain can start a workflow instance without direct access to workflow internals.
- **SC-003**: Cross-engine calls preserve traceability across both engines.
- **SC-004**: Duplicate cross-engine requests are detected by business key or idempotency key.

## Assumptions

- Both target engines have stable adapter interfaces before this specification is implemented.
- Initial linkage may support synchronous calls first, with asynchronous behavior added behind the same standard node contract.
- Idempotency storage can use existing SQL Server tables or a dedicated table decided during planning.
- Failure strategy is configured per standard node.

## Constitution Alignment *(mandatory)*

- **Flow Type**: CROSS_ENGINE
- **Unified Model Impact**: Adds or standardizes cross-engine node `config` fields and result mappings.
- **Engine Boundary**: Cross-engine interactions only pass through route/API/service adapters.
- **Data & Audit**: Records caller node log, target execution or instance log, and shared `trace_id`.
- **API & Response Contract**: Uses snake_case standard node payloads and stable sub-codes for cross-engine failures.
- **Verification Evidence**: Workflow-to-rule-chain tests, rule-chain-to-workflow tests, timeout tests, idempotency tests, and trace audit checks.