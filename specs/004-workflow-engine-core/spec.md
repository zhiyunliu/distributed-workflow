# Feature Specification: 工作流引擎核心

**Feature Branch**: `004-workflow-engine-core`  
**Created**: 2026-06-16  
**Status**: Draft  
**Input**: 根据 `docs/requirement.md` 拆分统一流程编排平台规格。

## User Scenarios & Testing *(mandatory)*

### User Story 1 - 启动并持久化工作流实例 (Priority: P1)

作为业务系统，我希望启动一个长周期工作流后，系统保存实例状态、变量和当前节点，使流程可以跨请求继续执行。

**Why this priority**: 实例持久化是工作流区别于规则链的核心能力。

**Independent Test**: 启动一个包含人工等待节点的工作流，系统创建实例并在等待点保存状态快照。

**Acceptance Scenarios**:

1. **Given** 一个已发布工作流定义，**When** 业务系统启动流程，**Then** 系统创建 `sch_workflow_instance` 记录。
2. **Given** 流程执行到人工任务，**When** 引擎进入等待状态，**Then** 系统保存当前节点、变量和状态快照。

---

### User Story 2 - 恢复等待中的工作流 (Priority: P1)

作为流程参与人，我希望提交人工任务处理结果后，系统从保存的等待点恢复流程并继续执行。

**Why this priority**: 没有恢复能力，长周期工作流无法覆盖审批、工单和订单生命周期场景。

**Independent Test**: 让流程停在人工任务，提交处理结果后验证实例状态推进到下一节点。

**Acceptance Scenarios**:

1. **Given** 一个等待中的实例，**When** 用户提交审批结果，**Then** 引擎加载状态快照并继续流转。
2. **Given** 审批结果触发条件分支，**When** 流程恢复执行，**Then** 系统按对应 `connections` 路由到后继节点。

---

### User Story 3 - 处理补偿和人工协作 (Priority: P2)

作为流程管理员，我希望工作流支持会签、或签、退回、跳转、超时和事务补偿，使复杂业务流程可控且可审计。

**Why this priority**: BPM 能力是工作流完整性的关键，但可在基础启动和恢复能力后迭代。

**Independent Test**: 配置包含会签或补偿点的流程，模拟失败或多人处理，系统按策略更新实例和节点日志。

**Acceptance Scenarios**:

1. **Given** 节点配置补偿策略，**When** 后续节点失败，**Then** 系统执行或记录补偿动作。
2. **Given** 人工任务超时，**When** 超时策略触发，**Then** 系统按配置转派、终止或升级处理。

### Edge Cases

- 实例状态快照损坏时不得盲目继续执行。
- 人工任务重复提交时必须通过幂等或状态校验拒绝第二次处理。
- 工作流允许循环和回退，但必须避免无界循环耗尽资源。
- 任务处理人无权限时必须拒绝并记录审计。
- 补偿失败时必须记录可人工介入的状态。
- 已停用定义不得启动新实例，但已有实例恢复策略需要明确。

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST execute only definitions with `flow_type=WORKFLOW` in the workflow engine.
- **FR-002**: System MUST create and update `sch_workflow_instance` for long-running workflow state.
- **FR-003**: System MUST persist variables, current node, status, and `state_snapshot` at durable wait points.
- **FR-004**: System MUST support manual task wait and resume operations.
- **FR-005**: System MUST support sequence flow, condition branch, parallel or gateway-like routing, rollback, jump, and terminal handling as planned workflow capabilities.
- **FR-006**: System MUST support compensation metadata and failure handling for long-running transactions.
- **FR-007**: System MUST write `sch_workflow_node_log` for every executed workflow node.
- **FR-008**: System MUST preserve workflow-specific runtime context inside the workflow engine boundary.
- **FR-009**: System MUST use glue Cache for hot definitions or instance data where caching is needed.

### Key Entities *(include if feature involves data)*

- **Workflow Definition**: Published flow definition with `flow_type=WORKFLOW`.
- **Workflow Instance**: Durable process instance with status, variables, current node, and snapshot.
- **Activity Handler**: Workflow node behavior interface.
- **Execution Context**: Workflow runtime state including variables, node state, user action, and trace id.
- **Persistence Service**: Component responsible for saving and loading workflow state.
- **Manual Task**: Human work item with assignee, decision, form data, timeout, and completion state.
- **Workflow Node Log**: Per-node trace for input, output, error, duration, and operator.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A workflow can stop at an artificial wait point and resume from persisted state.
- **SC-002**: Manual task duplicate submission is rejected deterministically.
- **SC-003**: Every node execution creates a queryable log record with `trace_id`.
- **SC-004**: Workflow engine execution never calls rule-chain internal node handlers directly.

## Assumptions

- Workflow support can be delivered after the rule-chain MVP but must remain compatible with the unified model.
- Manual task assignment and permission details may be refined during planning.
- State machine modeling is part of workflow capability but can be phased after core flowchart execution.
- Long-running instance retention policy will be finalized with observability and audit planning.

## Constitution Alignment *(mandatory)*

- **Flow Type**: WORKFLOW
- **Unified Model Impact**: Consumes explicit `connections` and maps workflow semantics to persisted edges.
- **Engine Boundary**: Workflow runtime state stays inside workflow engine; cross-engine behavior uses standard nodes only.
- **Data & Audit**: Writes `sch_workflow_instance` and `sch_workflow_node_log`; uses `trace_id`.
- **API & Response Contract**: Resume and manual task APIs use POST and unified response codes.
- **Verification Evidence**: Instance persistence tests, resume tests, manual task tests, duplicate submission tests, compensation failure checks.