# Feature Specification: 运行观测与审计治理

**Feature Branch**: `007-observability-audit`  
**Created**: 2026-06-16  
**Status**: Draft  
**Input**: 根据 `docs/requirement.md` 拆分统一流程编排平台规格。

## User Scenarios & Testing *(mandatory)*

### User Story 1 - 查询执行链路 (Priority: P1)

作为运维人员，我希望通过 `trace_id` 查询一次流程定义保存、发布、执行和节点处理的完整链路，快速定位问题。

**Why this priority**: 双引擎平台如果缺少统一追踪，跨引擎和高频规则链问题会难以排查。

**Independent Test**: 执行一次规则链和一次工作流，使用 `trace_id` 查询执行总表、节点日志和 API 日志。

**Acceptance Scenarios**:

1. **Given** 一次规则链执行，**When** 运维人员按 `trace_id` 查询，**Then** 系统返回执行总览和节点明细。
2. **Given** 一次工作流人工任务恢复，**When** 运维人员按实例查询，**Then** 系统返回实例状态和节点执行历史。

---

### User Story 2 - 审计关键操作 (Priority: P1)

作为安全或平台管理员，我希望保存、发布、停用、导入导出、调试、执行和跨引擎调用都有可审计记录。

**Why this priority**: 企业级平台必须满足流程治理、权限追踪和问题追责。

**Independent Test**: 对流程执行保存、发布、停用、导出和执行操作后，审计记录包含操作人、时间、对象、结果和 trace id。

**Acceptance Scenarios**:

1. **Given** 用户发布流程定义，**When** 操作成功，**Then** 系统记录发布审计事件。
2. **Given** 用户无权限执行流程，**When** 操作失败，**Then** 系统记录失败审计事件和错误子码。

---

### User Story 3 - 治理日志增长和保留 (Priority: P2)

作为平台管理员，我希望高频规则链日志支持归档或保留策略，避免日志表无限增长影响查询和存储成本。

**Why this priority**: 高频事件链路会快速产生节点日志，必须在第三阶段治理。

**Independent Test**: 配置日志保留窗口后，系统可以区分热数据查询和历史归档策略。

**Acceptance Scenarios**:

1. **Given** 规则链节点日志超过保留窗口，**When** 归档任务运行，**Then** 系统按策略归档或标记历史数据。
2. **Given** 运维查询热数据，**When** 使用常用过滤条件，**Then** 查询走结构化列而非大 JSON 全量扫描。

### Edge Cases

- 执行失败也必须写入可诊断日志，不得只记录成功路径。
- 跨引擎调用必须能关联调用方节点和目标执行或实例。
- 高频规则链日志写入失败时不得导致主链路无限阻塞。
- 审计记录不得泄露敏感载荷、密钥或 Token。
- 日志 JSON 字段必须合法，常用查询维度必须列化。
- trace id 缺失时系统必须生成或返回明确错误，策略需在计划阶段确认。

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST use glue logging, monitoring, and tracing capabilities as the primary observability foundation.
- **FR-002**: System MUST record rule-chain execution summaries in `sch_rulechain_exec_log`.
- **FR-003**: System MUST record rule-chain node details in `sch_rulechain_node_log`.
- **FR-004**: System MUST record workflow node details in `sch_workflow_node_log`.
- **FR-005**: System MUST carry `trace_id` through API, router, engines, standard nodes, and logs.
- **FR-006**: System MUST audit save, publish, disable, import, export, debug, execute, permission failure, and cross-engine operations.
- **FR-007**: System MUST store common query dimensions as structured columns and full payloads or snapshots as `nvarchar(max)` JSON where needed.
- **FR-008**: System MUST support log retention, archiving, or partitioning strategy for high-frequency rule-chain logs.
- **FR-009**: System MUST avoid storing secrets, passwords, or tokens in logs and audit payloads.

### Key Entities *(include if feature involves data)*

- **Trace Context**: Request and execution correlation data including `trace_id`.
- **Audit Event**: Operation record containing actor, action, target, result, timestamp, and error sub-code.
- **Rule-chain Execution Log**: Summary of one rule-chain run.
- **Rule-chain Node Log**: Detail of one node execution inside a rule-chain run.
- **Workflow Node Log**: Detail of one workflow node execution.
- **Retention Policy**: Rule for keeping, archiving, or partitioning hot and historical logs.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Every execution path has a queryable `trace_id`.
- **SC-002**: Failed operations produce audit events with stable error sub-codes.
- **SC-003**: Operators can query one rule-chain execution and see ordered node details.
- **SC-004**: Log queries use structured columns for `flow_code`, `flow_type`, `trace_id`, status, and time range.

## Assumptions

- glue observability capabilities are available and should be reused before introducing external monitoring dependencies.
- Detailed dashboard requirements can be planned after raw logs and audit events are reliable.
- Retention windows may differ between execution detail logs and audit records.
- Sensitive payload masking rules will be finalized during planning.

## Constitution Alignment *(mandatory)*

- **Flow Type**: INFRASTRUCTURE
- **Unified Model Impact**: Uses flow metadata for log dimensions but does not redefine the model.
- **Engine Boundary**: Observes both engines without sharing runtime context.
- **Data & Audit**: Defines logging, audit, retention, structured columns, and trace behavior.
- **API & Response Contract**: Query APIs use GET; audit and maintenance actions use POST where they mutate state.
- **Verification Evidence**: Trace propagation checks, audit event checks, log persistence checks, sensitive data masking checks, retention strategy validation.