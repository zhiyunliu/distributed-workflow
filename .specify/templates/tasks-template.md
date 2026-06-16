---

description: "Task list template for feature implementation"
---

# Tasks: [FEATURE NAME]

**Input**: Design documents from `/specs/[###-feature-name]/`
**Prerequisites**: plan.md (required), spec.md (required for user stories), research.md, data-model.md, contracts/

**Tests**: Include tests or validation tasks whenever the feature touches unified models, engine routing,
SQL Server persistence, public API contracts, response codes, frontend flows, permissions, audit logs,
or observability. If automated tests are not feasible, include explicit manual validation evidence.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Path Conventions

- **Go backend**: `api/`, `service/`, `dao/`, `model/`, `engine/`, `router/`, `constants/` at repository root
- **Rule-chain engine**: `engine/rulechain/` for DAG validation, scheduling, routing, and node handlers
- **Workflow engine**: `engine/workflow/` for long-running state, manual tasks, recovery, and compensation
- **Management UI**: `management/src/views/`, `management/src/components/`, `management/src/api/`
- **Documentation**: `docs/` and feature files under `specs/[###-feature-name]/`
- Paths shown below are samples - replace them with exact paths from plan.md

<!-- 
  ============================================================================
  IMPORTANT: The tasks below are SAMPLE TASKS for illustration purposes only.
  
  The /speckit.tasks command MUST replace these with actual tasks based on:
  - User stories from spec.md (with their priorities P1, P2, P3...)
  - Feature requirements from plan.md
  - Entities from data-model.md
  - Endpoints from contracts/
  
  Tasks MUST be organized by user story so each story can be:
  - Implemented independently
  - Tested independently
  - Delivered as an MVP increment
  
  DO NOT keep these sample tasks in the generated tasks.md file.
  ============================================================================
-->

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Project initialization and basic structure

- [ ] T001 Confirm feature scope, flow type, and affected directories from plan.md
- [ ] T002 Confirm glue module/API registration approach and existing project conventions
- [ ] T003 [P] Confirm SQL Server migration or schema update location
- [ ] T004 [P] Confirm management UI route/API type locations when frontend changes are included

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core infrastructure that MUST be complete before ANY user story can be implemented

**CRITICAL**: No user story work can begin until this phase is complete

Examples of foundational tasks (adjust based on your project):

- [ ] T005 Define or update unified model structs and snake_case JSON tags in model/
- [ ] T006 Define response codes in constants/respcode and sub-codes in constants/subcode
- [ ] T007 Create or update `sch_` SQL Server schema with structured query columns and JSON validation
- [ ] T008 [P] Implement model validation, including required fields and type-specific rules
- [ ] T009 [P] Implement authorization and audit hooks for affected operations
- [ ] T010 Setup glue API routing, middleware, logging, trace propagation, and error mapping

**Checkpoint**: Foundation ready - user story implementation can now begin in parallel

---

## Phase 3: User Story 1 - [Title] (Priority: P1) MVP

**Goal**: [Brief description of what this story delivers]

**Independent Test**: [How to verify this story works on its own]

### Tests for User Story 1

> **NOTE: Write these tests FIRST, ensure they FAIL before implementation**

- [ ] T011 [P] [US1] Model/API contract test or validation evidence for [contract]
- [ ] T012 [P] [US1] Integration test or manual validation for [user journey]

### Implementation for User Story 1

- [ ] T013 [P] [US1] Create or update model/ entities and DTOs with snake_case JSON tags
- [ ] T014 [P] [US1] Create or update dao/ SQL Server persistence and transactions
- [ ] T015 [US1] Implement service/ orchestration, permissions, and audit behavior
- [ ] T016 [US1] Implement engine/ or router/ behavior when this story touches execution
- [ ] T017 [US1] Implement api/ handler with glue binding and unified response mapping
- [ ] T018 [US1] Implement management UI view/API wrapper when this story includes frontend work
- [ ] T019 [US1] Add structured logging, trace id propagation, and error code mapping

**Checkpoint**: At this point, User Story 1 should be fully functional and testable independently

---

## Phase 4: User Story 2 - [Title] (Priority: P2)

**Goal**: [Brief description of what this story delivers]

**Independent Test**: [How to verify this story works on its own]

### Tests for User Story 2

- [ ] T020 [P] [US2] Contract test or validation evidence for [endpoint/model]
- [ ] T021 [P] [US2] Integration test or manual validation for [user journey]

### Implementation for User Story 2

- [ ] T022 [P] [US2] Create or update model/ and dao/ artifacts
- [ ] T023 [US2] Implement service/ behavior and engine/router integration if needed
- [ ] T024 [US2] Implement api/ handler and response code mapping
- [ ] T025 [US2] Integrate with prior story components without breaking independent validation

**Checkpoint**: At this point, User Stories 1 AND 2 should both work independently

---

## Phase 5: User Story 3 - [Title] (Priority: P3)

**Goal**: [Brief description of what this story delivers]

**Independent Test**: [How to verify this story works on its own]

### Tests for User Story 3

- [ ] T026 [P] [US3] Contract test or validation evidence for [endpoint/model]
- [ ] T027 [P] [US3] Integration test or manual validation for [user journey]

### Implementation for User Story 3

- [ ] T028 [P] [US3] Create or update model/ and dao/ artifacts
- [ ] T029 [US3] Implement service/ behavior and engine/router integration if needed
- [ ] T030 [US3] Implement api/ handler and management UI changes if needed

**Checkpoint**: All user stories should now be independently functional

---

[Add more user story phases as needed, following the same pattern]

---

## Phase N: Polish & Cross-Cutting Concerns

**Purpose**: Improvements that affect multiple user stories

- [ ] TXXX [P] Documentation updates in docs/
- [ ] TXXX Code cleanup and refactoring
- [ ] TXXX Performance optimization across all stories
- [ ] TXXX [P] Additional Go unit tests or frontend checks required by plan.md
- [ ] TXXX Security hardening
- [ ] TXXX Run quickstart.md validation
- [ ] TXXX Verify constitution gates: model, engine boundary, glue reuse, SQL Server audit, API contract, observability

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user stories
- **User Stories (Phase 3+)**: All depend on Foundational phase completion
  - User stories can then proceed in parallel (if staffed)
  - Or sequentially in priority order (P1 → P2 → P3)
- **Polish (Final Phase)**: Depends on all desired user stories being complete

### User Story Dependencies

- **User Story 1 (P1)**: Can start after Foundational (Phase 2) - No dependencies on other stories
- **User Story 2 (P2)**: Can start after Foundational (Phase 2) - May integrate with US1 but should be independently testable
- **User Story 3 (P3)**: Can start after Foundational (Phase 2) - May integrate with US1/US2 but should be independently testable

### Within Each User Story

- Tests (if included) MUST be written and FAIL before implementation
- Models before services
- Services before endpoints
- Core implementation before integration
- Story complete before moving to next priority

### Parallel Opportunities

- All Setup tasks marked [P] can run in parallel
- All Foundational tasks marked [P] can run in parallel (within Phase 2)
- Once Foundational phase completes, all user stories can start in parallel (if team capacity allows)
- All tests for a user story marked [P] can run in parallel
- Models within a story marked [P] can run in parallel
- Different user stories can be worked on in parallel by different team members

---

## Parallel Example: User Story 1

```bash
# Launch all tests for User Story 1 together (if tests requested):
Task: "Contract test for [endpoint] in api/[feature]_test.go"
Task: "Integration validation for [user journey] using specs/[###-feature-name]/quickstart.md"

# Launch all models for User Story 1 together:
Task: "Create [Entity1] model in model/[entity1].go"
Task: "Create [Entity2] DAO in dao/[entity2].go"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational (CRITICAL - blocks all stories)
3. Complete Phase 3: User Story 1
4. **STOP and VALIDATE**: Test User Story 1 independently
5. Deploy/demo if ready

### Incremental Delivery

1. Complete Setup + Foundational → Foundation ready
2. Add User Story 1 → Test independently → Deploy/Demo (MVP!)
3. Add User Story 2 → Test independently → Deploy/Demo
4. Add User Story 3 → Test independently → Deploy/Demo
5. Each story adds value without breaking previous stories

### Parallel Team Strategy

With multiple developers:

1. Team completes Setup + Foundational together
2. Once Foundational is done:
   - Developer A: User Story 1
   - Developer B: User Story 2
   - Developer C: User Story 3
3. Stories complete and integrate independently

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story for traceability
- Each user story should be independently completable and testable
- Verify tests fail before implementing
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently
- Avoid: vague tasks, same file conflicts, cross-story dependencies that break independence
