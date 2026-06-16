# Feature Specification: 自研规则链引擎核心

**Feature Branch**: `003-rulechain-engine-core`  
**Created**: 2026-06-16  
**Status**: Draft  
**Input**: 根据 `docs/requirement.md` 拆分统一流程编排平台规格。

## User Scenarios & Testing *(mandatory)*

### User Story 1 - 执行无状态规则链 (Priority: P1)

作为业务系统，我希望提交一条事件消息后，规则链引擎按模型中的节点和连线完成过滤、转换、动作和日志处理。

**Why this priority**: 规则链是第一阶段 MVP 的核心执行能力。

**Independent Test**: 创建包含过滤、转换和日志节点的规则链定义，提交消息后验证节点按连线结果执行并产生日志。

**Acceptance Scenarios**:

1. **Given** 一个已发布规则链，**When** 事件消息进入执行入口，**Then** 引擎创建独立规则上下文并从入口节点开始执行。
2. **Given** 节点返回成功结果，**When** 连线路由匹配 `relation_type=success`，**Then** 引擎推进到对应后继节点。

---

### User Story 2 - 拒绝非法 DAG (Priority: P1)

作为平台维护人员，我希望规则链发布或加载时执行 DAG 校验，避免循环导致运行时死循环或资源耗尽。

**Why this priority**: DAG 是规则链与工作流的关键边界。

**Independent Test**: 提交包含环的规则链模型，发布或预加载失败，并返回明确错误。

**Acceptance Scenarios**:

1. **Given** `node_a -> node_b -> node_a` 的规则链，**When** 系统发布，**Then** 发布失败并返回 DAG 错误。
2. **Given** 一个合法 DAG，**When** 系统预加载，**Then** 构建邻接表和入度信息。

---

### User Story 3 - 扩展节点处理器 (Priority: P2)

作为开发人员，我希望通过统一节点处理器接口注册过滤、转换、HTTP、日志等节点，而无需修改执行内核。

**Why this priority**: 扩展机制决定后续节点生态，但可在基础执行后逐步完善。

**Independent Test**: 注册一个自定义节点处理器，在规则链执行时通过节点类型找到并调用该处理器。

**Acceptance Scenarios**:

1. **Given** 一个已注册节点类型，**When** 执行器加载节点，**Then** 节点工厂返回对应处理器。
2. **Given** 一个未注册节点类型，**When** 执行器加载节点，**Then** 执行失败并返回节点处理器不存在错误。

### Edge Cases

- 规则链没有入口节点时必须拒绝发布或执行。
- 节点处理器不存在时必须终止当前链路并记录错误。
- 条件表达式无法解析时必须按错误路径或默认错误处理策略执行。
- 执行超时、并发过高或节点 panic 时必须捕获并记录失败结果。
- 同一次执行不得复用其他消息的运行时上下文。
- 规则链内不得配置人工任务、回退或循环节点。

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST implement a pure Go rule-chain runtime without third-party rule-chain engine dependency.
- **FR-002**: System MUST parse the unified flow model and execute only definitions with `flow_type=RULE_CHAIN`.
- **FR-003**: System MUST validate DAG topology using a deterministic algorithm such as Kahn topological sorting.
- **FR-004**: System MUST build an adjacency structure from `connections` before execution.
- **FR-005**: System MUST create an isolated `rule_context` for every message execution.
- **FR-006**: System MUST route next nodes by `relation_type`, `condition`, and `is_default`.
- **FR-007**: System MUST expose a node handler interface and node factory for extensibility.
- **FR-008**: System MUST support at least filter, transform, HTTP action, and log node categories as built-in or planned built-in nodes.
- **FR-009**: System MUST record execution summary and node details for every rule-chain execution.
- **FR-010**: System MUST support timeout and concurrency controls for high-throughput scenarios.

### Key Entities *(include if feature involves data)*

- **Rule Chain Definition**: Published flow definition with `flow_type=RULE_CHAIN`.
- **Rule Context**: Single-message runtime data including payload, variables, trace id, node outputs, and errors.
- **Rule Node Handler**: Interface implemented by every rule-chain node type.
- **Connection Router**: Component selecting next nodes based on node result and connection metadata.
- **Execution Log**: Summary record for one rule-chain execution.
- **Node Log**: Per-node execution detail with input, output, status, duration, and error.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Legal DAG definitions execute from entry node to terminal path without manual engine selection.
- **SC-002**: Cyclic rule-chain definitions are rejected before execution.
- **SC-003**: Missing node handlers produce stable error responses and node logs.
- **SC-004**: Each execution writes a summary log and at least one node detail log.

## Assumptions

- Cache abstraction is provided by glue Cache and stores hot published definitions or compiled adjacency data.
- Rule-chain execution is stateless beyond persisted logs and outputs.
- Initial implementation may run synchronously; async execution and worker pools can be added behind the same executor contract.
- Condition language and expression sandbox details will be refined during planning.

## Constitution Alignment *(mandatory)*

- **Flow Type**: RULE_CHAIN
- **Unified Model Impact**: Consumes `nodes`, `connections`, `relation_type`, `condition`, and `is_default`.
- **Engine Boundary**: Rejects workflow-only constructs and never calls workflow internals directly.
- **Data & Audit**: Writes `sch_rulechain_exec_log` and `sch_rulechain_node_log`; carries `trace_id`.
- **API & Response Contract**: Engine errors are mapped by the router/API layer to stable response constants.
- **Verification Evidence**: DAG validation tests, node routing tests, missing handler tests, timeout tests, and log persistence checks.