# Implementation Plan: [FEATURE]

**Branch**: `[###-feature-name]` | **Date**: [DATE] | **Spec**: [link]
**Input**: Feature specification from `/specs/[###-feature-name]/spec.md`

**Note**: This template is filled in by the `/speckit.plan` command. See `.specify/templates/plan-template.md` for the execution workflow.

## Summary

[Extract from feature spec: primary requirement + technical approach from research]

## Technical Context

<!--
  ACTION REQUIRED: Replace the content in this section with the technical details
  for the project. The structure here is presented in advisory capacity to guide
  the iteration process.
-->

**Language/Version**: Go 1.24+ for backend; Vue 3 + TypeScript strict mode for management UI  
**Primary Dependencies**: `github.com/zhiyunliu/glue`, SQL Server driver, Vue 3, Vite, Element Plus, Pinia, Axios  
**Storage**: SQL Server 2016+; JSON fields use `nvarchar(max)` with structured query columns  
**Testing**: Go standard `testing` package; frontend test tooling only when already present or explicitly planned  
**Target Platform**: glue-based backend API service plus `management` frontend application
**Project Type**: Go microservice + Vue management frontend  
**Performance Goals**: Define p95 latency, rule-chain throughput, workflow recovery time, and audit query targets per feature  
**Constraints**: unified explicit-edge model; isolated workflow/rule-chain engines; glue-first infrastructure; snake_case JSON keys  
**Scale/Scope**: Define affected flow types, node count, version volume, execution volume, and retention window per feature

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

1. **Unified Model Gate**: Plan identifies impact on `flow_type`, `flow_code`, versioning, `nodes`,
  `connections`, variables, settings, and JSON schema. Public JSON keys use lowercase snake_case.
2. **Engine Boundary Gate**: Plan classifies work as `WORKFLOW`, `RULE_CHAIN`, cross-engine standard
  node, or management-only; no direct runtime context sharing or cross-engine node jumping is introduced.
3. **glue Reuse Gate**: Plan uses glue for API routing, dependency composition, middleware, cache,
  logging, monitoring, and service governance unless an exception is documented.
4. **SQL Server & Audit Gate**: Plan defines `sch_` tables or changes, structured query columns,
  `nvarchar(max)` JSON fields, `datetime2(3)` timestamps, transaction boundaries, and audit records.
5. **API Contract Gate**: Plan follows GET for query operations and POST for create/update/delete,
  publish, execute, and debug operations; response codes and sub-codes are mapped to constants.
6. **Verification Gate**: Plan lists model validation, DAG/routing checks, persistence tests,
  API/contract checks, frontend checks where applicable, and observability evidence.

## Project Structure

### Documentation (this feature)

```text
specs/[###-feature]/
├── plan.md              # This file (/speckit.plan command output)
├── research.md          # Phase 0 output (/speckit.plan command)
├── data-model.md        # Phase 1 output (/speckit.plan command)
├── quickstart.md        # Phase 1 output (/speckit.plan command)
├── contracts/           # Phase 1 output (/speckit.plan command)
└── tasks.md             # Phase 2 output (/speckit.tasks command - NOT created by /speckit.plan)
```

### Source Code (repository root)
<!--
  ACTION REQUIRED: Replace the placeholder tree below with the concrete layout
  for this feature. Delete unused options and expand the chosen structure with
  real paths (e.g., apps/admin, packages/something). The delivered plan must
  not include Option labels.
-->

```text
api/                    # HTTP handlers, binding, auth, unified responses
service/                # Business orchestration, permissions, version management
dao/                    # SQL Server access, transactions, pagination
model/                  # Database models, DTOs, unified flow model structs
engine/
├── workflow/           # Long-running workflow runtime
└── rulechain/          # Pure Go rule-chain runtime and DAG scheduler
router/                 # Flow-type and version based engine routing
constants/
├── respcode/           # Stable numeric response codes
└── subcode/            # Stable business sub-codes
management/             # Vue 3 management UI and admin-only capabilities
docs/                   # Architecture, requirement, quickstart, API notes
```

**Structure Decision**: [Document the selected structure and reference the real
directories captured above]

## Complexity Tracking

> **Fill ONLY if Constitution Check has violations that must be justified**

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| [e.g., 4th project] | [current need] | [why 3 projects insufficient] |
| [e.g., Repository pattern] | [specific problem] | [why direct DB access insufficient] |
