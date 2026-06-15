# 20260615 执行阶段计划

| 字段 | 内容 |
| --- | --- |
| plan_id | 20260615-execution-phase-plan |
| task_id | TASK-006 |
| 文档类型 | 执行阶段计划 |
| 目标读者 | 开发人员 |
| 计划状态 | 可进入执行拆分 |
| 适用仓库 | distributed-workflow |

## 1. 计划背景与范围

本计划根据 `docs/architecture-design.md` 拆分统一流程编排平台的执行阶段，服务于后续研发排期、任务派发、验收评审和风险控制。它是执行阶段计划，不是代码实现，不创建数据库脚本、checkpoint yaml、tasks.json，也不替代后续接口详细设计、测试用例或代码评审。

本计划覆盖统一元模型、SQL Server 存储、发布与版本治理、规则链 MVP、工作流补齐、双引擎联动、监控审计与性能优化五个阶段。计划中的代码路径均为建议路径，因为当前仓库尚未包含后端或前端代码目录；后续实现时应按项目实际结构落位，并保持 `api`、`service`、`model`、`dao`、`engine`、`router`、`cache/log/monitor` 分层边界。

范围内：

| 范围 | 说明 |
| --- | --- |
| 执行阶段拆分 | 明确五阶段目标、任务边界、依赖、交付物和验收标准。 |
| 工程约束固化 | 固化 Go 1.24+、glue、SQL Server、Vue3/Vite/TypeScript/Element Plus、统一响应、JWT、API 方法与路径约束。 |
| 模块边界 | 拆分后端、前端、数据库、引擎、API、安全、观测、测试/验收等横向任务域。 |
| 后续任务清单 | 给出可执行的 Markdown checkbox 任务，使用 T001 起递增编号。 |
| 治理闭环 | 将发布版本、停用版本、复制新版本、导入导出、设计治理审计、权限拒绝审计纳入规则链和工作流执行前置条件。 |

范围外：

| 范围外事项 | 说明 |
| --- | --- |
| 代码实现 | 不创建 `api`、`service`、`model`、`dao`、`engine`、`frontend` 等代码文件。 |
| 配置修改 | 不修改 `go.mod`、`vite.config.ts`、数据库连接配置或其他核心配置。 |
| 数据库脚本落地 | 本计划描述数据库先行任务，但不创建迁移 SQL 文件。 |
| 详细 API 文档 | 本计划引用 API 边界和路径规范，不生成完整 OpenAPI 文档。 |

## 2. 来源文档与约束

| 来源 | 用途 | 本计划采用方式 |
| --- | --- | --- |
| `docs/architecture-design.md` | 主要架构来源，包含五层架构、双引擎隔离、元模型、数据库、API、安全、非功能和实施路线。 | 作为阶段拆分、模块边界、验收标准和风险控制的主依据。 |
| `docs/requirement.md` | 项目背景、目标、统一元模型、双引擎诉求和落地建议。 | 用于补充业务背景、双引擎目标和统一设计入口的必要性。 |
| `AGENTS.md` | 项目技术栈、Go/Vue 规范、统一响应、API 方法、安全和 AI 操作边界。 | 固化工程实现约束，确保后续执行不偏离项目规范。 |

### 2.1 技术栈约束

| 方向 | 约束 |
| --- | --- |
| 后端 | Go 1.24+，基于 `github.com/zhiyunliu/glue`，错误显式处理，结构体必须带 JSON 标签。 |
| 前端 | Vue 3、Vite、TypeScript、Element Plus，推荐使用 `<script setup lang="ts">`，组件大驼峰命名，页面放在 `views/`，公共组件放在 `components/`。 |
| 数据库 | SQL Server；核心字段优先使用 `varchar`、`datetime`；JSON 字段使用 `varchar(max)`；不建立物理外键，逻辑关联完整性由代码实现。 |
| 缓存/日志/监控 | 优先复用 glue Cache、glue 结构化日志、glue 监控能力，不重复引入同类基础设施。 |

### 2.2 API 与安全约束

| 方向 | 约束 |
| --- | --- |
| API 前缀 | 统一使用 `/api/basic-distributed`。 |
| 路径参数 | 使用 `:param` 格式，例如 `/flows/:definition_id/publish`。 |
| HTTP 方法 | 查询类接口使用 GET；创建、修改、删除、执行、发布类接口使用 POST。 |
| 统一响应 | 所有接口返回 `code`、`sub_code`、`message`、`data`；成功 `code=0` 且不输出 `sub_code`；错误必须输出 `sub_code`。 |
| 错误常量 | `code` 和 `sub_code` 必须分别在建议路径 `constants/respcode` 与 `constants/subcode` 中定义。 |
| 认证 | 使用 `Authorization: Bearer token`；敏感接口必须校验 JWT。 |
| 权限 | 设计、发布、执行、日志查看、跨引擎调用权限分离。 |
| 密钥 | 禁止硬编码密钥、密码、Token，必须通过环境变量或配置管理注入。 |
| SQL 安全 | `dao` 层禁止不安全 SQL 拼接，统一使用参数化查询或框架安全封装。 |

### 2.3 引擎与模型约束

| 方向 | 约束 |
| --- | --- |
| 统一元模型 | 流程定义必须遵循统一 JSON Schema，使用 `type` 区分 `WORKFLOW` 与 `RULE_CHAIN`，用 `nodes` 与 `connections` 显式表达节点和连线。 |
| 双引擎隔离 | 工作流和规则链不能共享内部上下文，不能直接跳转对方节点。 |
| 跨引擎联动 | 只能通过标准节点联动：工作流调用规则链节点、规则链启动工作流节点。 |
| 规则链 DAG | 规则链必须通过 DAG 校验，禁止循环。 |
| 循环与人工等待 | 循环、回退、人工等待归工作流处理，不放入规则链语义。 |
| trace_id | API、service、engine、日志、跨引擎调用必须透传或生成统一 `trace_id`。 |

## 3. 总体阶段路线

| 阶段 | 名称 | 主要目标 | 最小可用边界 |
| --- | --- | --- | --- |
| 第一阶段 | 基础元模型、数据库和发布治理先行 | 固化统一 JSON Schema、核心表、发布版本治理、治理审计、基础 API、统一响应、安全中间件边界和设计器骨架。 | 能保存、校验、发布、停用、复制新版本、导入导出并审计一个 `WORKFLOW` 和一个 `RULE_CHAIN` 定义。 |
| 第二阶段 | 规则链 MVP | 打通规则链模型加载、DAG 校验、节点执行、日志写入、限流超时熔断边界和前端调试预览。 | 一个已发布规则链可同步执行，环路模型被拒绝，执行与节点日志可追踪，高并发保护可验证。 |
| 第三阶段 | 工作流补齐 | 建设工作流实例、人工任务、状态恢复、节点日志和设计器工作流能力。 | 一个含人工节点的工作流可启动、等待、处理并完成。 |
| 第四阶段 | 双引擎联动 | 通过标准节点实现工作流调用规则链、规则链启动工作流，贯通权限审计和 trace_id。 | 双向跨引擎端到端用例通过，且不共享内部上下文。 |
| 第五阶段 | 监控审计与性能优化 | 完善 glue Cache 热更新、结构化日志、监控指标、容量基线、日志归档、慢节点分析、索引优化、安全审计和压测。 | 具备生产可观测、可审计、可回源验证和基础性能容量评估能力。 |

## 4. 横向任务域拆分

| 任务域 | 职责边界 | 主要建议路径 | 重点阶段 |
| --- | --- | --- | --- |
| 后端 | API 接入、service 编排、错误码、统一响应、中间件、权限边界。 | `api/`、`service/`、`router/`、`middleware/`、`constants/` | 全阶段，第一阶段建立边界。 |
| 前端 | 统一设计器、节点工具箱、属性面板、草稿保存、规则链调试、工作流人工任务界面、监控页面。 | `frontend/src/api/`、`frontend/src/views/`、`frontend/src/components/`、`frontend/src/types/` | 第一至第五阶段。 |
| 数据库 | 表结构、索引、归档策略、逻辑关联完整性、参数化查询支撑。 | `dao/`、`model/`、`database/migrations/` | 第一阶段先行，第五阶段优化。 |
| 引擎 | 规则链执行器、工作流执行器、节点处理器、上下文隔离、跨引擎标准节点。 | `engine/rulechain/`、`engine/workflow/`、`engine/common/` | 第二至第四阶段。 |
| API | 资源路径、GET/POST 边界、路径参数、统一响应、分页查询、执行入口。 | `api/`、`router/`、`model/dto/` | 全阶段。 |
| 安全 | JWT、CORS、权限校验、发布/执行隔离、审计、敏感字段脱敏、SQL 安全。 | `middleware/`、`service/auth/`、`service/audit/`、`dao/` | 第一阶段定边界，第四和第五阶段强化。 |
| 治理审计 | 草稿保存、发布、停用、复制新版本、导入、导出、权限拒绝、跨引擎治理操作审计和查询。 | `service/audit/`、`dao/audit/`、`log/`、`frontend/src/views/audit/` | 第一阶段建立闭环，第四和第五阶段扩展。 |
| 观测 | `trace_id`、结构化日志、执行总表、节点明细、监控指标、慢节点分析。 | `log/`、`monitor/`、`engine/*/log`、`dao/*log` | 第二阶段开始，第五阶段完善。 |
| 测试/验收 | 模型校验、DAO 参数化、规则链 DAG、工作流恢复、跨引擎联动、安全审计、压测。 | `tests/`、`api/*_test.go`、`engine/*_test.go`、`frontend/src/**/*.spec.ts` | 全阶段，按阶段验收。 |

## 5. 第一阶段：基础元模型、数据库和发布治理先行

### 阶段目标

固化统一 JSON Schema、核心表、状态/版本模型、发布治理、治理审计、统一响应、错误码常量、JWT/CORS 中间件边界和基础设计器框架，为规则链和工作流后续执行提供稳定的模型、存储和版本发布底座。

### 范围内任务

| 任务 | 说明 |
| --- | --- |
| 统一 JSON Schema | 定义 `flowCode`、`type`、`version`、`variables`、`settings`、`nodes`、`connections` 等核心结构。 |
| 数据库表设计 | 落地 `sch_flow_definition`、`sch_workflow_instance`、`sch_workflow_node_log`、`sch_rulechain_exec_log`、`sch_rulechain_node_log` 的模型和迁移计划。 |
| 基础 API | 模型校验、保存草稿、查询草稿/定义列表、导入导出边界。 |
| 发布与版本治理 | 支持发布版本、停用版本、复制新版本、默认最新版本、显式版本查询和状态流转校验。 |
| 发布权限校验 | 发布、停用、复制新版本、导入、导出等治理操作必须校验 JWT 与权限码，权限拒绝返回稳定 `sub_code`。 |
| 治理审计 | 保存草稿、发布、停用、复制新版本、导入、导出、权限拒绝写入 glue 结构化日志，并可选落到建议表 `sch_flow_design_audit_log`。 |
| 响应与错误 | 定义统一响应封装、`constants/respcode`、`constants/subcode`。 |
| 安全边界 | 明确 JWT、CORS、设计/发布/查看权限边界。 |
| 基础设计器 | 建立 Vue3/Vite/TypeScript/Element Plus 的设计器页面骨架、类型和 API 封装。 |

### 范围外任务

| 范围外任务 | 原因 |
| --- | --- |
| 规则链真实执行 | 放到第二阶段，第一阶段只保证模型可保存、校验、发布和停用。 |
| 工作流实例流转 | 放到第三阶段，第一阶段只建表、发布治理和基本模型。 |
| 跨引擎调用 | 依赖规则链和工作流两个内核，放到第四阶段。 |
| 性能压测与归档自动化 | 依赖足量执行日志，放到第五阶段。 |

### 涉及模块

| 模块 | 说明 |
| --- | --- |
| `model` | 统一元模型、数据库模型、DTO、状态枚举。 |
| `dao` | SQL Server 表访问、草稿查询、分页查询、参数化写入。 |
| `service` | 模型校验、版本状态、草稿保存、权限检查编排。 |
| `api` | HTTP 接入、参数绑定、统一响应。 |
| `router` | `/api/basic-distributed` 路由注册。 |
| `middleware` | JWT、CORS、`trace_id` 边界。 |
| `service/audit` | 设计治理操作审计、发布审计、权限拒绝审计和审计查询编排。 |
| `frontend/src` | 设计器骨架、API 封装、TS 类型。 |

### 前置依赖

| 依赖 | 说明 |
| --- | --- |
| 架构设计确认 | 统一元模型、核心表和 API 边界已在架构设计中明确；建议审计表与后续建议待办表仅作为执行计划扩展，不修改架构设计文档。 |
| 技术栈确认 | Go 1.24+、glue、SQL Server、Vue3/Vite/TypeScript/Element Plus 已由项目规范确认。 |
| 数据库约定确认 | 使用 `varchar`、`datetime`、无物理外键、代码实现关联完整性。 |

### 关键交付物

| 交付物 | 建议路径 |
| --- | --- |
| 统一元模型结构 | `model/flow_model.go`、`model/dto/flow_definition.go` |
| 核心表迁移计划 | `database/migrations/` |
| 错误码常量 | `constants/respcode/`、`constants/subcode/` |
| 统一响应封装 | `api/response.go` 或 `pkg/response/` |
| 基础 API | `api/flow_definition.go`、`service/flow_definition_service.go`、`dao/flow_definition_dao.go` |
| 发布治理服务 | `service/publish_service.go`、`dao/flow_definition_dao.go` |
| 治理审计 | `service/audit_service.go`、`dao/design_audit_dao.go`、建议表 `sch_flow_design_audit_log` |
| 路由注册 | `router/basic_distributed.go` |
| 前端设计器骨架 | `frontend/src/views/designer/`、`frontend/src/api/flowDefinition.ts`、`frontend/src/types/flow.ts` |
| 前端版本治理操作 | `frontend/src/views/designer/VersionPanel.vue`、`frontend/src/views/audit/DesignAuditLog.vue` |

### 最小验收标准

| 验收项 | 标准 |
| --- | --- |
| 模型校验 | 能校验一个 `WORKFLOW` 草稿和一个 `RULE_CHAIN` 草稿，并返回结构化 errors/warnings。 |
| 保存查询 | 能保存并查询流程草稿，响应格式符合 `code/sub_code/message/data`。 |
| 发布版本 | `POST /api/basic-distributed/flows/:definition_id/publish` 可将 `status` 从 `DRAFT` 或 `VALIDATED` 变为 `PUBLISHED`，返回 `definition_id`、`flow_code`、`version`、`status`、`is_latest`、`published_at`。 |
| 停用版本 | `POST /api/basic-distributed/flows/:definition_id/deactivate` 可将 `PUBLISHED` 版本变为 `DEACTIVATED`，显式旧版本执行后续必须按状态拒绝或按策略处理。 |
| 复制新版本 | `POST /api/basic-distributed/flows/:definition_id/copy` 生成 `DRAFT` 新版本，表记录保留 `source_definition_id`、`flow_code`、递增 `version`、`is_latest=false`。 |
| 状态流转 | 版本状态枚举至少包含 `DRAFT`、`VALIDATED`、`PUBLISHED`、`DEACTIVATED`、`ARCHIVED`，非法流转返回稳定 `sub_code`。 |
| 发布审计 | 保存草稿、发布、停用、复制新版本、导入、导出、权限拒绝均记录 `operation`、`actor_id`、`permission_code`、`flow_code`、`version`、`status_before`、`status_after`、`sub_code`、`trace_id`、`created_at`。 |
| 数据库 | 核心表和建议治理审计表迁移设计可评审，字段类型、索引、逻辑关联和 JSON 策略明确。 |
| 安全边界 | JWT、CORS、权限边界在路由和中间件层明确，不在业务代码分散处理。 |
| 前端骨架 | 设计器能加载基础页面，具备流程类型选择、模型编辑入口和保存按钮骨架。 |

### 测试/验证方式

| 验证方式 | 说明 |
| --- | --- |
| 单元测试 | 校验 JSON Schema、错误码映射、响应封装、DAO 参数化查询。 |
| API 测试 | 覆盖 `POST /api/basic-distributed/flows/model/validate`、`POST /api/basic-distributed/flows/drafts/save`、GET 查询接口。 |
| 发布治理测试 | 覆盖发布、停用、复制新版本、导入、导出、非法状态流转、无权限发布拒绝和审计记录落点。 |
| 审计查询测试 | 按 `actor_id`、`operation`、`flow_code`、`version`、`status_after`、`sub_code`、`trace_id`、时间范围查询治理审计记录。 |
| 数据库评审 | 检查表字段、索引、JSON 字段、无物理外键和代码关联完整性策略。 |
| 前端静态检查 | TypeScript 类型、组件命名、API 封装路径、版本治理操作和 Element Plus 用法符合规范。 |

### 风险与灰区

| 风险/灰区 | 处理方式 |
| --- | --- |
| 需求文档中出现 `nvarchar`、`datetime2`，执行约束要求 `varchar`、`datetime` | 本计划以架构设计后续章节和当前执行约束为准，统一采用 `varchar`、`datetime`。 |
| JSON Schema 过早复杂化 | 第一阶段只固化双引擎共同字段和必要校验，扩展节点配置按注册表演进。 |
| 无物理外键带来脏数据风险 | 在 `service` / `dao` 事务中做存在性校验、停用限制和逻辑级联处理。 |
| 设计器功能范围膨胀 | 第一阶段只做骨架、基础类型、保存/查询和模型校验入口。 |
| 发布治理绕过执行入口 | 第二、三阶段执行入口必须只加载 `PUBLISHED` 版本，且显式旧版本和默认最新版本策略都经 `router` 与 `service` 校验。 |

## 6. 第二阶段：规则链 MVP

### 阶段目标

优先打通高并发、低延迟、无状态事件链路。实现规则链模型加载、DAG 校验、ConnectionRouter、节点处理器接口、节点工厂、基础过滤/转换/日志节点、执行 API、执行入口限流、超时、熔断、协程池/并发保护、执行日志写入和前端调试预览。

### 范围内任务

| 任务 | 说明 |
| --- | --- |
| 模型加载 | 从已发布 `RULE_CHAIN` 定义加载统一元模型，构建邻接表和入度表。 |
| DAG 校验 | 使用拓扑排序校验环路，发布或执行前拒绝循环模型。 |
| ConnectionRouter | 按 `relationType`、条件表达式、默认连线选择后继节点。 |
| 节点 SPI | 定义 `IRuleNodeHandler`，通过节点工厂注册基础节点。 |
| 基础节点 | 实现过滤、转换、日志节点，作为规则链 MVP 能力闭环。 |
| 执行 API | 提供 `POST /api/basic-distributed/rulechains/execute`。 |
| 高并发治理 | 执行入口支持限流、超时、熔断、协程池/并发保护，错误映射到稳定限流和超时 `sub_code`。 |
| 执行日志 | 写入 `sch_rulechain_exec_log` 与 `sch_rulechain_node_log`。 |
| 前端调试 | 提供规则链模拟输入、路径预览和节点输出展示。 |

### 范围外任务

| 范围外任务 | 原因 |
| --- | --- |
| 人工等待、回退、循环 | 这些语义属于工作流，规则链 MVP 禁止实现。 |
| 工作流启动节点 | 属于第四阶段跨引擎联动。 |
| 大规模归档 | 属于第五阶段性能优化。 |
| 复杂外部集成节点 | MVP 只提供基础过滤、转换、日志节点。 |

### 涉及模块

| 模块 | 说明 |
| --- | --- |
| `engine/rulechain` | 规则链模型加载、执行器、上下文、DAG 校验、ConnectionRouter。 |
| `engine/rulechain/node` | `IRuleNodeHandler`、节点工厂、基础节点。 |
| `service/rulechain` | 执行编排、权限校验、定义版本加载、限流熔断、日志落库编排。 |
| `engine/common` | 可复用超时、并发保护、协程池和执行保护接口，不持有具体引擎内部类型。 |
| `dao/rulechain` | 执行总表和节点日志表写入查询。 |
| `api/rulechain` | 执行接口、调试接口。 |
| `frontend/src/views/rulechain` | 调试预览、执行结果、路径展示。 |

### 前置依赖

| 依赖 | 说明 |
| --- | --- |
| 第一阶段模型 | 统一元模型和 `sch_flow_definition` 已可保存、查询、发布。 |
| 第一阶段日志表 | `sch_rulechain_exec_log` 与 `sch_rulechain_node_log` 结构已明确。 |
| 安全边界 | 执行权限、JWT、统一响应和错误码常量已建立。 |
| 发布治理闭环 | 规则链执行只允许加载 `PUBLISHED` 版本；默认最新版本与显式版本都必须通过状态和权限校验。 |

### 关键交付物

| 交付物 | 建议路径 |
| --- | --- |
| 规则链模型加载器 | `engine/rulechain/loader.go` |
| DAG 校验器 | `engine/rulechain/dag_validator.go` |
| ConnectionRouter | `engine/rulechain/connection_router.go` |
| 节点接口和工厂 | `engine/rulechain/node/handler.go`、`engine/rulechain/node/factory.go` |
| 基础节点 | `engine/rulechain/node/filter.go`、`transform.go`、`log.go` |
| 执行 API | `api/rulechain.go`、`service/rulechain_service.go` |
| 执行保护 | `service/rulechain_guard_service.go`、`engine/common/execution_guard.go` |
| 日志 DAO | `dao/rulechain_log_dao.go` |
| 前端调试预览 | `frontend/src/views/rulechain/DebugPreview.vue`、`frontend/src/api/rulechain.ts` |

### 最小验收标准

| 验收项 | 标准 |
| --- | --- |
| 规则链执行 | 一个已发布规则链可通过 `POST /api/basic-distributed/rulechains/execute` 同步执行成功。 |
| 环路拒绝 | 含循环的规则链模型在发布或执行前被拒绝，并返回稳定 `sub_code`。 |
| 限流拒绝 | 超过规则链执行入口阈值时返回统一响应，错误包含限流 `sub_code`，不创建不完整节点明细。 |
| 超时熔断 | 单次执行超过配置超时返回超时 `sub_code`；连续失败达到阈值进入熔断窗口，恢复后可半开探测。 |
| 并发保护 | 并发执行不超过配置的协程池或工作队列容量，排队、拒绝和取消路径都有稳定日志字段。 |
| 节点日志 | 总表和节点明细写入 `trace_id`、`status`、`error_code`、`duration_ms`。 |
| 上下文隔离 | 每次规则链执行创建独立上下文，执行结束只保留日志和结果。 |
| 前端预览 | 前端能展示模拟输入、路由路径、节点输出和错误信息。 |

### 测试/验证方式

| 验证方式 | 说明 |
| --- | --- |
| 单元测试 | DAG 校验、ConnectionRouter、节点工厂、过滤/转换/日志节点。 |
| API 测试 | 规则链执行成功、模型不存在、未发布、环路、节点失败、超时响应。 |
| 限流测试 | 构造超过入口阈值的并发请求，断言响应 `code != 0`、`sub_code` 为限流常量、总表状态为 `REJECTED` 或无总表写入策略明确。 |
| 熔断测试 | 构造连续节点失败，断言熔断窗口内请求被拒绝，窗口后探测请求可进入执行。 |
| 日志验证 | 查询 `trace_id` 能串起执行总表和节点明细。 |
| 并发冒烟 | 小规模并发执行验证上下文隔离、协程池容量、取消路径和日志写入稳定性。 |

### 风险与灰区

| 风险/灰区 | 处理方式 |
| --- | --- |
| 条件表达式能力边界不清 | MVP 先支持明确的基础表达式或配置化判断，复杂表达式作为后续扩展。 |
| 日志写入影响执行耗时 | MVP 可先同步写入保证可追踪，第五阶段再做异步批量优化。 |
| 节点错误映射不稳定 | 所有节点错误必须映射到 `constants/subcode` 中稳定标识。 |
| 规则链被误用于长周期流程 | 通过模型校验和前端工具箱隔离禁止人工等待、循环和回退节点。 |
| 限流阈值早期缺少业务 SLA | 第二阶段先给技术保护阈值和可配置项，第五阶段通过压测报告沉淀容量基线。 |

## 7. 第三阶段：工作流补齐

### 阶段目标

建设长周期、人工参与、状态持久化的工作流能力。实现 `WorkflowInstance`、`ActivityHandler`、`ExecutionContext`、`PersistenceService`、人工任务存储模型、会签或签最小策略、补偿策略字段、状态恢复、工作流启动/查询/处理 API、工作流设计器节点与属性面板、节点日志。

### 范围内任务

| 任务 | 说明 |
| --- | --- |
| 工作流实例 | 管理实例生命周期、状态、当前节点、变量和业务键。 |
| ActivityHandler | 定义工作流节点处理器接口和基础活动处理流程。 |
| ExecutionContext | 承载工作流变量、节点状态、运行时数据和 `trace_id`。 |
| PersistenceService | 保存和恢复实例状态、变量、当前节点和快照。 |
| 人工任务 | 支持创建待办、处理待办、等待状态、审批结果写回，建议表 `sch_workflow_task` 形成待办存储闭环。 |
| 会签或签 | 第三阶段至少支持会签全员完成和或签任一通过两种最小策略，复杂加权策略后续扩展。 |
| 补偿策略 | 在实例、节点日志或建议表中记录补偿策略字段、失败补偿路径和人工介入策略。 |
| 状态恢复 | 根据 `instance_id` 加载快照并从等待节点恢复执行。 |
| API | 启动工作流、查询实例、查询详情、处理人工任务。 |
| 前端设计器 | 工作流节点工具箱、人工任务属性面板、实例查询和待办处理页面。 |
| 节点日志 | 写入 `sch_workflow_node_log`，支持按 `instance_id` 和 `trace_id` 追踪。 |

### 范围外任务

| 范围外任务 | 原因 |
| --- | --- |
| 工作流调用规则链节点 | 放到第四阶段跨引擎联动。 |
| 规则链启动工作流节点 | 放到第四阶段跨引擎联动。 |
| 会签/或签的复杂策略全量实现 | 第三阶段可保留接口和最小策略，复杂策略后续迭代。 |
| 大规模待办统计和监控大盘 | 放到第五阶段观测优化。 |

### 涉及模块

| 模块 | 说明 |
| --- | --- |
| `engine/workflow` | 实例、执行上下文、活动处理器、状态恢复和流转。 |
| `service/workflow` | 启动、查询、人工任务处理、权限与事务编排。 |
| `dao/workflow` | 实例表、节点日志表、人工任务相关查询写入。 |
| `api/workflow` | 工作流启动、实例查询、任务处理 API。 |
| `frontend/src/views/workflow` | 工作流设计器节点、属性面板、实例详情、待办处理。 |

### 前置依赖

| 依赖 | 说明 |
| --- | --- |
| 第一阶段模型和表 | `WORKFLOW` 草稿保存、发布、实例表、节点日志表已明确。 |
| 第一阶段安全 | JWT、权限、统一响应和错误码常量已建立。 |
| 设计器基础 | 已有统一画布和模型保存基础。 |

### 关键交付物

| 交付物 | 建议路径 |
| --- | --- |
| 工作流实例对象 | `engine/workflow/instance.go` |
| 活动处理器接口 | `engine/workflow/activity_handler.go` |
| 执行上下文 | `engine/workflow/execution_context.go` |
| 持久化服务 | `engine/workflow/persistence_service.go` |
| 人工任务服务 | `service/workflow_task_service.go`、`dao/workflow_task_dao.go` |
| 工作流 API | `api/workflow.go` |
| 工作流日志 DAO | `dao/workflow_log_dao.go` |
| 人工任务建议表 | 建议路径 `database/migrations/`，建议表名 `sch_workflow_task` |
| 前端工作流界面 | `frontend/src/views/workflow/Designer.vue`、`InstanceDetail.vue`、`TaskHandle.vue` |

### 最小验收标准

| 验收项 | 标准 |
| --- | --- |
| 启动工作流 | 一个已发布工作流可通过 `POST /api/basic-distributed/workflows/start` 创建实例。 |
| 人工等待 | 含人工节点的工作流可进入等待状态，并生成可查询待办。 |
| 人工任务存储 | 建议表 `sch_workflow_task` 至少包含 `task_id`、`instance_id`、`node_id`、`assignee_id`、`candidate_group`、`permission_code`、`claim_status`、`task_status`、`approval_policy`、`approval_result`、`due_at`、`claimed_at`、`completed_at`、`trace_id`、`created_at`、`updated_at`。 |
| 人工任务索引 | 待办查询必须命中 `assignee_id/task_status/due_at`、`candidate_group/task_status`、`instance_id/node_id`、`trace_id` 维度索引或等价组合索引。 |
| 人工任务状态流转 | 待办状态至少包含 `CREATED`、`CLAIMED`、`COMPLETED`、`CANCELLED`、`EXPIRED`；非法重复处理返回稳定 `sub_code`。 |
| 人工处理 | 待办处理后流程继续流转并最终完成。 |
| 会签或签 | 会签在所有必需处理人完成后继续；或签在任一有权限处理人通过后继续并取消其他待办。 |
| 补偿路径 | 节点失败后按 `compensation_policy` 选择重试、执行补偿节点、转人工或失败终止，并在节点日志记录 `compensation_status`。 |
| 状态恢复 | 服务重启或重新加载后能根据 `instance_id` 恢复当前节点和变量。 |
| 节点日志 | `sch_workflow_node_log` 可按 `instance_id` 和 `trace_id` 回溯节点轨迹。 |

### 测试/验证方式

| 验证方式 | 说明 |
| --- | --- |
| 单元测试 | `ExecutionContext`、状态转换、持久化服务、人工任务状态机。 |
| API 测试 | 启动、查询实例、处理人工任务、重复处理、无权限、实例不存在。 |
| 人工任务断言 | 断言待办表记录、状态枚举变化、权限字段、审批结果、取消待办和审计日志与 API 返回一致。 |
| 补偿测试 | 覆盖节点失败重试、补偿节点执行、补偿失败转人工和失败终止四类最小路径。 |
| 恢复测试 | 模拟实例等待后重新加载，确认变量、当前节点和待办状态一致。 |
| 前端验证 | 工作流节点工具箱、属性面板、实例详情和待办处理交互可用。 |

### 风险与灰区

| 风险/灰区 | 处理方式 |
| --- | --- |
| 工作流状态机复杂度上升 | 第三阶段优先完成启动、等待、处理、完成、失败等核心状态。 |
| 人工任务权限粒度不清 | 先实现处理权限边界和审计字段，复杂组织权限后续扩展。 |
| 事务补偿范围过大 | 第三阶段保留补偿策略字段和基础错误处理，不强行覆盖所有补偿场景。 |
| 实例快照 JSON 结构演进 | 使用版本字段和向后兼容加载策略，避免历史实例不可恢复。 |

## 8. 第四阶段：双引擎联动

### 阶段目标

通过标准节点完成双引擎能力互补。实现工作流调用规则链节点、规则链启动工作流节点、`trace_id` 透传、权限审计和跨引擎端到端测试，同时保持工作流与规则链内部上下文完全隔离。跨引擎必须经 `service/cross_engine_service` 和 `router/engine_router`，禁止 `engine/workflow` 与 `engine/rulechain` 互相直接依赖内部类型。

### 范围内任务

| 任务 | 说明 |
| --- | --- |
| 工作流调用规则链节点 | 在工作流中配置目标规则链、版本策略、输入映射、输出映射和失败策略。 |
| 规则链启动工作流节点 | 在规则链中配置目标工作流、业务键、变量映射和启动模式。 |
| 标准输入输出 | 调用方只传标准 payload，接收标准 result/error，不访问对方内部上下文。 |
| trace_id 透传 | 跨引擎调用必须沿用调用方 `trace_id`，禁止生成孤立链路号。 |
| 权限审计 | 跨引擎调用经过路由层权限校验，并写入双方执行日志、跨引擎审计和 glue 结构化日志。 |
| 跨引擎审计字段 | 审计必须记录 `direction`、`caller_engine`、`callee_engine`、`caller_instance_id` 或 `caller_exec_id`、`callee_instance_id` 或 `callee_exec_id`、`standard_node_id`、`actor_id`、`permission_code`、`idempotency_key`、`duration_ms`、`sub_code`、`trace_id`。 |
| 端到端测试 | 工作流调用规则链、规则链启动工作流各提供一条可重复验证用例。 |

### 范围外任务

| 范围外任务 | 原因 |
| --- | --- |
| 引擎内部上下文共享 | 明确禁止，违反双引擎隔离原则。 |
| 跨引擎直接节点跳转 | 明确禁止，只能通过标准节点。 |
| 复杂分布式事务 | 第四阶段通过幂等、超时、重试和补偿策略控制，不引入跨引擎强事务。 |
| 全量性能优化 | 放到第五阶段。 |

### 涉及模块

| 模块 | 说明 |
| --- | --- |
| `engine/workflow/node` | 规则链调用节点。 |
| `engine/rulechain/node` | 启动工作流节点。 |
| `router` | 根据 `flow_type`、版本、状态和权限分发跨引擎请求，唯一入口建议路径 `router/engine_router.go`。 |
| `service` | 标准节点调用编排、权限审计、幂等和错误处理，唯一编排建议路径 `service/cross_engine_service.go`。 |
| `dao` | 双方日志写入和查询串联。 |
| `frontend/src` | 跨引擎节点属性面板和联动调试结果展示。 |

### 前置依赖

| 依赖 | 说明 |
| --- | --- |
| 第二阶段规则链 MVP | 规则链已可发布、执行、写日志。 |
| 第三阶段工作流能力 | 工作流已可启动、等待、处理、恢复和写日志。 |
| trace_id 机制 | API、service、engine 和日志已支持 `trace_id`。 |
| 权限模型 | 发布、执行、跨引擎调用权限边界已可用。 |

### 关键交付物

| 交付物 | 建议路径 |
| --- | --- |
| 工作流规则链调用节点 | `engine/workflow/node/rulechain_call.go` |
| 规则链启动工作流节点 | `engine/rulechain/node/start_workflow.go` |
| 跨引擎服务编排 | `service/cross_engine_service.go` |
| 路由扩展 | `router/engine_router.go` |
| 跨引擎审计日志 | `service/audit_service.go`、`log/cross_engine.go` |
| 前端节点配置 | `frontend/src/components/designer/nodes/RulechainCallNode.vue`、`StartWorkflowNode.vue` |
| 端到端测试 | `tests/e2e/cross_engine/` |

### 最小验收标准

| 验收项 | 标准 |
| --- | --- |
| 工作流调用规则链 | 工作流执行到规则链调用节点后能触发规则链，结果写回工作流变量并继续流转。 |
| 规则链启动工作流 | 规则链满足条件后能启动工作流实例，并返回 `instance_id` 和状态。 |
| trace_id 串联 | 调用方节点日志、被调用方执行日志和 glue 日志可用同一 `trace_id` 关联。 |
| 隔离验证 | 跨引擎调用不读取或写入对方内部上下文，只使用标准输入输出。 |
| 权限审计 | 无权限跨引擎调用被拒绝并产生稳定错误码和审计日志。 |
| 跨引擎字段 | 跨引擎审计记录包含 `direction`、`caller_engine`、`callee_engine`、`caller_instance_id/caller_exec_id`、`callee_instance_id/callee_exec_id`、`standard_node_id`、`actor_id`、`permission_code`、`idempotency_key`、`duration_ms`、`sub_code`、`trace_id`。 |
| 前端隔离 | 工作流工具箱只展示工作流节点和规则链调用标准节点；规则链工具箱只展示规则链节点和启动工作流标准节点。 |
| 导入拒绝 | 导入 JSON 中跨模式混用内部节点类型时，后端发布校验拒绝并返回稳定 `sub_code`。 |

### 测试/验证方式

| 验证方式 | 说明 |
| --- | --- |
| 端到端测试 | 覆盖工作流调用规则链、规则链启动工作流两条主链路。 |
| 权限测试 | 覆盖执行权限不足、发布权限不足、日志查看权限不足。 |
| 失败策略测试 | 覆盖被调用方超时、节点失败、定义未发布、版本不存在。 |
| 双引擎隔离测试 | 覆盖前端工具箱模式隔离、导入 JSON 混用拒绝、发布校验拒绝跨模式内部节点、代码依赖扫描禁止 `engine/workflow` 与 `engine/rulechain` 互相 import 内部类型。 |
| 日志追踪验证 | 通过 `trace_id` 查询双方日志和结构化日志，确认链路完整。 |

### 风险与灰区

| 风险/灰区 | 处理方式 |
| --- | --- |
| 跨引擎调用引入隐式耦合 | 标准节点只接受配置化输入输出映射，不暴露对方内部 context。 |
| 重复调用造成重复实例或重复执行 | 使用 `business_key`、`message_id`、幂等键和 trace 组合控制。 |
| 超时策略影响业务结果 | 节点配置必须声明超时、重试、失败后终止或转人工处理策略。 |
| 权限边界复杂 | 发布权限、执行权限和跨引擎调用权限分开校验，并记录审计。 |

## 9. 第五阶段：监控审计与性能优化

### 阶段目标

提升生产可运维能力。完善 glue Cache 热更新、结构化日志、监控指标、容量基线、日志归档、慢节点分析、索引优化、安全审计和性能压测，让平台具备可观测、可审计、可调优的运行闭环。

### 范围内任务

| 任务 | 说明 |
| --- | --- |
| glue Cache 热更新 | 发布版本后刷新流程定义、规则链邻接表、节点处理器元数据缓存。 |
| 结构化日志 | 统一记录 `trace_id`、用户、接口、flow_code、version、节点、耗时、sub_code。 |
| 监控指标 | 规则链 QPS、P95/P99、节点失败率；工作流待办量、超时量、恢复失败量。 |
| 容量基线 | 建立技术基准表，覆盖规则链执行总表、规则链节点明细、工作流实例、工作流节点日志、归档数据量、并发数、分页大小、查询时间阈值、P95/P99 采样窗口。 |
| 日志归档 | 按月或按时间范围归档执行日志和节点日志，保留查询键。 |
| 热冷查询策略 | 明确热表与归档表查询边界，同一 API 是否跨热/冷数据、分页排序规则和索引命中要求。 |
| 慢节点分析 | 按 `flow_code`、`node_type`、`duration_ms` 定位慢节点。 |
| 索引优化 | 验证核心表组合索引、时间范围查询和 trace 查询性能。 |
| 安全审计 | 校验 JWT、权限、CORS、敏感字段脱敏、SQL 参数化、硬编码密钥扫描。 |
| 性能压测 | 覆盖规则链高并发、工作流恢复和查询场景。 |

### 范围外任务

| 范围外任务 | 原因 |
| --- | --- |
| 引入额外监控平台 | 优先使用 glue 原生监控能力，外部平台作为后续可选集成。 |
| 重构前四阶段核心设计 | 第五阶段以优化和加固为主，非必要不重做模型和引擎。 |
| 合规报表系统 | 可保留 `sch_audit_log` 扩展方向，但第五阶段先复用结构化日志和执行日志。 |

### 涉及模块

| 模块 | 说明 |
| --- | --- |
| `cache` | 流程定义缓存、规则链拓扑缓存、热更新策略。 |
| `log` | 结构化日志字段、脱敏、跨引擎链路。 |
| `monitor` | 指标采集、慢节点统计、健康检查。 |
| `dao` | 索引友好查询、日志归档、分页查询优化。 |
| `service` | 发布热更新、审计、安全检查、慢节点分析服务。 |
| `frontend/src/views/monitor` | 监控审计查询和慢节点分析页面。 |
| `tests/performance` | 压测脚本、容量报告和验收数据。 |

### 前置依赖

| 依赖 | 说明 |
| --- | --- |
| 第二阶段执行日志 | 规则链总表和节点明细已有真实写入。 |
| 第三阶段工作流日志 | 工作流实例和节点日志已有真实写入。 |
| 第四阶段 trace_id | 跨引擎链路可通过统一 `trace_id` 关联。 |
| 基础权限 | 设计、发布、执行、查看日志、跨引擎权限已分离。 |

### 关键交付物

| 交付物 | 建议路径 |
| --- | --- |
| Cache 热更新服务 | `cache/flow_definition_cache.go`、`service/publish_service.go` |
| 结构化日志封装 | `log/fields.go`、`middleware/trace.go` |
| 监控指标采集 | `monitor/rulechain_metrics.go`、`monitor/workflow_metrics.go` |
| 日志归档任务 | `service/log_archive_service.go`、`dao/log_archive_dao.go` |
| 慢节点分析 | `service/slow_node_service.go`、`api/monitor.go` |
| 索引优化报告 | `docs/performance/index-review.md` |
| 安全审计报告 | `docs/security/audit-review.md` |
| 压测报告 | `docs/performance/load-test-report.md` |

### 容量基线表

没有明确业务 SLA 前，第五阶段先以技术基准作为上线前验收门槛；后续可按真实业务 SLA 调整阈值。

| 对象 | 热数据技术基准 | 归档数据基准 | 并发数 | 分页大小 | 查询时间阈值 | P95/P99 采样窗口 |
| --- | --- | --- | --- | --- | --- | --- |
| 规则链执行总表 `sch_rulechain_exec_log` | 1,000,000 行以内热查询 | 月度归档，单月 5,000,000 行以内 | 200 并发执行 | 20/50/100 | 常规列表 P95 <= 500ms，trace 精确查询 <= 200ms | 连续 30 分钟压测，按 1 分钟聚合 |
| 规则链节点明细 `sch_rulechain_node_log` | 10,000,000 行以内热查询 | 月度归档，单月 50,000,000 行以内 | 200 并发写入 | 20/50/100 | 按 `exec_id` 或 `trace_id` 查询 P95 <= 800ms | 连续 30 分钟压测，按 1 分钟聚合 |
| 工作流实例 `sch_workflow_instance` | 500,000 个运行中或近期待办实例 | 完结实例按季度归档 | 100 并发启动/恢复 | 20/50/100 | 实例列表 P95 <= 500ms，`instance_id` 精确查询 <= 200ms | 连续 30 分钟压测，按 1 分钟聚合 |
| 工作流节点日志 `sch_workflow_node_log` | 5,000,000 行以内热查询 | 季度归档，单季 20,000,000 行以内 | 100 并发写入 | 20/50/100 | 按 `instance_id` 或 `trace_id` 查询 P95 <= 800ms | 连续 30 分钟压测，按 1 分钟聚合 |
| 归档日志查询 | 默认不跨冷数据 | 单次查询限定月份或季度分区 | 20 并发查询 | 20/50 | 归档列表 P95 <= 2000ms | 连续 15 分钟压测，按 1 分钟聚合 |

### 最小验收标准

| 验收项 | 标准 |
| --- | --- |
| 链路查询 | 可按 `flow_code`、状态、时间范围、`trace_id` 查询执行链路。 |
| 监控指标 | 规则链 QPS、P95/P99、节点失败率和工作流待办/超时/恢复失败量可观测。 |
| 缓存热更新 | 发布新版本后默认最新版本执行命中新版本；停用版本后默认执行拒绝停用版本；显式旧版本执行按状态策略可执行或稳定拒绝；规则链邻接表和节点处理器元数据刷新成功。 |
| 回源失败验证 | 缓存未命中回源失败时返回稳定 `sub_code`，不使用过期未确认定义执行，并记录 `trace_id`、`flow_code`、`version`、`cache_key`、`sub_code`。 |
| 日志归档 | 归档策略验证可用，归档后仍可按 `trace_id`、`flow_code`、`exec_id`、`instance_id` 追踪。 |
| 热冷查询 | 明确同一 API 默认只查热表；需要跨热/冷数据时必须显式传入归档时间范围，排序固定为 `created_at desc, id desc` 或等价稳定游标。 |
| 索引命中 | `trace_id`、`exec_id`、`instance_id` 精确查询命中索引；慢查询超过 1000ms 必须记录 SQL 标识、参数摘要、耗时、执行计划摘要和 `trace_id`。 |
| 慢节点分析 | 可按节点类型、流程编码和耗时排序定位慢节点。 |
| 安全审计 | 无硬编码密钥，无不安全 SQL 拼接，JWT/CORS/权限/脱敏策略通过检查。 |
| 压测 | 规则链执行和工作流恢复在目标测试数据量下有容量结论和瓶颈记录。 |

### 测试/验证方式

| 验证方式 | 说明 |
| --- | --- |
| 压测 | 模拟规则链高并发执行、工作流实例恢复、日志查询分页。 |
| 容量基线验证 | 使用容量基线表中的数据量、并发数、分页大小和采样窗口执行压测，报告 P95/P99、错误率、慢查询和瓶颈。 |
| 指标核对 | 对比执行日志与监控指标，确认计数、耗时、失败率一致。 |
| 归档演练 | 构造历史日志并执行归档，验证热查询、归档查询、跨热冷查询开关、分页排序和查询键保留。 |
| 缓存演练 | 依次执行发布新版本、停用版本、默认最新版本执行、显式旧版本执行、规则链邻接表刷新、节点处理器元数据刷新和回源失败验证。 |
| 安全扫描 | 检查硬编码密钥、SQL 拼接、敏感日志、权限绕过。 |
| 索引评审 | 对核心查询执行计划和慢查询结果做评审。 |

### 风险与灰区

| 风险/灰区 | 处理方式 |
| --- | --- |
| 日志量快速膨胀 | 明细热数据短保留，总表长保留，按月归档或分区。 |
| 指标与日志口径不一致 | 统一指标字段和日志字段，验收时做交叉核对。 |
| 压测目标不明确 | 先给出基准容量和瓶颈点，不阻塞后续按业务 SLA 细化。 |
| 安全审计发现高风险项 | 高风险项必须阻断上线，低风险项记录整改优先级。 |

## 10. 阶段依赖矩阵

| 阶段 | 依赖第一阶段 | 依赖第二阶段 | 依赖第三阶段 | 依赖第四阶段 | 可并行事项 |
| --- | --- | --- | --- | --- | --- |
| 第一阶段：基础元模型、数据库和发布治理先行 | 无 | 无 | 无 | 无 | 前端设计器骨架、版本治理面板、审计查询页面可与后端模型设计并行。 |
| 第二阶段：规则链 MVP | 必须依赖统一模型、定义表、规则链日志表、统一响应、安全边界、发布状态流转、发布权限和治理审计。 | 无 | 无 | 无 | 基础节点、DAG 校验、限流保护和前端调试预览可并行。 |
| 第三阶段：工作流补齐 | 必须依赖统一模型、定义表、工作流实例表、节点日志表、统一响应、安全边界、发布治理闭环和治理审计。 | 不强依赖，但可复用日志、trace、发布版本和限流错误治理经验。 | 无 | 无 | 工作流引擎、人工任务存储、会签或签、补偿策略和前端属性面板可并行。 |
| 第四阶段：双引擎联动 | 必须依赖统一路由、安全、trace、发布治理和审计字段。 | 必须依赖规则链可执行、可记录日志、可限流和可拒绝未发布版本。 | 必须依赖工作流可启动、可恢复、可记录日志、人工任务闭环和补偿字段。 | 无 | 两类标准节点、前端模式隔离和混用拒绝测试可并行，最终做端到端联调。 |
| 第五阶段：监控审计与性能优化 | 依赖基础表、统一响应、安全边界、发布审计和设计治理审计。 | 依赖规则链执行、限流保护和日志。 | 依赖工作流实例、人工任务和日志。 | 依赖跨引擎 trace 与审计字段贯通。 | 指标、容量基线、索引、安全审计、缓存演练和压测准备可分流推进。 |

## 11. 验收标准矩阵

| 阶段 | 功能验收 | 数据验收 | API 验收 | 安全验收 | 观测验收 | 测试验收 |
| --- | --- | --- | --- | --- | --- | --- |
| 第一阶段 | 草稿保存、模型校验、发布、停用、复制新版本、导入导出可用。 | 核心表和建议审计表字段、索引、JSON 策略、状态枚举、无物理外键约定明确。 | GET/POST 边界、`/api/basic-distributed` 前缀和 `:param` 路径格式正确；发布类接口返回版本状态字段。 | JWT/CORS/发布权限/设计权限明确，无硬编码密钥；权限拒绝返回稳定 `sub_code`。 | `trace_id` 边界、发布审计和设计治理审计字段明确。 | 模型校验、响应封装、DAO 参数化、发布状态流转、权限拒绝和审计查询测试通过。 |
| 第二阶段 | 已发布规则链同步执行成功，环路拒绝，限流、超时、熔断和并发保护可验证。 | 规则链执行总表和节点日志写入完整，拒绝和超时策略有明确落库或不落库规则。 | `POST /api/basic-distributed/rulechains/execute` 符合统一响应，限流错误返回稳定 `sub_code`。 | 执行权限、发布状态、显式版本和入口限流边界可验证。 | `trace_id` 串起执行总表、节点明细、限流拒绝和超时日志。 | DAG、路由、节点、执行 API、限流、熔断、并发保护和压测冒烟通过。 |
| 第三阶段 | 工作流启动、等待、会签、或签、人工处理、补偿、完成和恢复可用。 | 实例表、建议待办任务表、节点日志支持恢复、权限过滤和状态追踪。 | 工作流启动、查询、人工任务领取/处理接口符合 GET/POST 边界并返回状态枚举变化。 | 实例查看、任务领取、任务处理和补偿转人工权限可验证。 | `instance_id`、`task_id` 与 `trace_id` 可回溯节点轨迹、待办状态和补偿路径。 | 状态恢复、人工任务、会签或签、补偿失败路径、重启恢复和 API 测试通过。 |
| 第四阶段 | 工作流调用规则链、规则链启动工作流可用，跨模式节点混用被拒绝。 | 双方日志和跨引擎审计记录通过同一 `trace_id` 关联，审计字段完整。 | 跨引擎调用只通过 `service/cross_engine_service` 和 `router/engine_router`，不暴露内部上下文。 | 跨引擎权限、幂等键、发布状态和无权限拒绝审计通过。 | 调用方、被调用方和跨引擎审计链路完整。 | 双向端到端、失败策略、权限、前端工具箱隔离、导入 JSON 混用拒绝和依赖扫描测试通过。 |
| 第五阶段 | 监控、缓存热更新、归档、热冷查询、慢节点分析、容量基线和压测报告可用。 | 索引、归档、热冷查询和容量基线有验证结果。 | 监控审计查询接口符合统一响应和权限要求，分页排序稳定。 | 安全审计通过，无高风险遗留；慢查询和权限拒绝都有审计记录。 | 指标、结构化日志、trace 查询、缓存回源失败和慢查询记录形成闭环。 | 压测、容量基线、缓存演练、归档演练、安全扫描和索引评审完成。 |

## 12. 里程碑和建议执行顺序

| 里程碑 | 建议顺序 | 完成标志 |
| --- | --- | --- |
| M1：模型、存储与发布治理冻结 | 第一阶段第 1 批 | JSON Schema、核心表、状态流转、发布/停用/复制新版本、响应和错误码边界评审通过。 |
| M2：基础 API、设计器与审计闭环 | 第一阶段第 2 批 | 草稿校验、保存、查询、导入导出、发布治理接口、前端基础页面和治理审计查询可联通。 |
| M3：规则链 MVP 可执行且受保护 | 第二阶段 | 已发布规则链可执行，环路拒绝，限流/超时/熔断/协程池保护和日志追踪可验证。 |
| M4：工作流人工链路可闭环 | 第三阶段 | 含人工节点工作流可启动、等待、会签或签、处理、补偿、完成和重启恢复。 |
| M5：双引擎互调可审计且隔离 | 第四阶段 | 双向跨引擎用例通过，`trace_id` 串联双方日志，跨引擎审计字段完整，混用内部节点被拒绝。 |
| M6：生产可运维基线 | 第五阶段 | 缓存热更新、监控指标、容量基线、日志归档、热冷查询、慢节点分析、安全审计和压测报告完成。 |

建议执行顺序：

1. 先完成统一 JSON Schema、核心表、状态枚举、发布治理、治理审计和错误码边界，避免上层功能反复返工。
2. 在基础 API 可用后，优先实现规则链 MVP 和执行保护，快速验证统一元模型对 DAG 执行与高并发治理的适配性。
3. 规则链稳定后补齐工作流长周期能力，复用已有模型、日志、trace、发布版本和审计机制。
4. 两个引擎均具备独立闭环后，再实现跨引擎标准节点，避免内部上下文耦合并补齐混用拒绝测试。
5. 最后做缓存热更新、观测、安全、归档、索引和性能优化，把验证数据沉淀为上线基线。

## 13. 风险、灰区与非阻塞决策

| 类型 | 内容 | 建议决策 | 是否阻塞当前阶段 |
| --- | --- | --- | --- |
| 风险 | 统一元模型过度抽象，影响引擎语义清晰。 | 元模型只统一存储和设计表达，执行语义由各引擎解释。 | 阻塞第一阶段模型冻结前评审。 |
| 风险 | 发布治理不完整导致第二、三阶段执行未发布或停用版本。 | 第一阶段必须完成发布、停用、复制新版本、状态流转、权限校验和审计闭环。 | 阻塞第二、三阶段执行入口验收。 |
| 风险 | 规则链日志增长快，影响查询和存储。 | 总表和明细表分层保留，按月归档或分区，第五阶段验证。 | 不阻塞第二阶段 MVP。 |
| 风险 | 高并发入口缺少限流和熔断导致资源耗尽。 | 第二阶段建立执行保护，限流和超时错误映射到稳定 `sub_code`，第五阶段压测校准。 | 阻塞第二阶段上线验收。 |
| 风险 | 跨引擎调用失败链路复杂。 | 强制 `trace_id`，标准节点记录输入输出，配置超时、重试、补偿或转人工策略。 | 阻塞第四阶段验收。 |
| 风险 | 无物理外键导致关联完整性依赖代码。 | `service` / `dao` 在事务内做存在性、状态和删除限制校验。 | 阻塞第一阶段 DAO 评审。 |
| 风险 | 人工任务状态和权限字段不足导致待办不可恢复或越权处理。 | 第三阶段使用建议表 `sch_workflow_task` 明确字段、索引、状态流转和权限校验断言。 | 阻塞第三阶段人工任务验收。 |
| 灰区 | 条件表达式引擎选择。 | MVP 使用可控表达式能力，复杂表达式后续扩展。 | 不阻塞第二阶段。 |
| 灰区 | 会签/或签完整策略。 | 第三阶段实现会签全员完成和或签任一通过最小策略，复杂策略按节点扩展。 | 不阻塞第三阶段最小验收。 |
| 灰区 | 独立审计表是否需要。 | 当前建议使用 `sch_flow_design_audit_log` 记录设计治理；跨引擎审计可先复用结构化日志，合规报表需要时再扩展统一 `sch_audit_log`。 | 不阻塞第五阶段。 |
| 灰区 | 外部监控平台集成。 | 当前优先 glue 原生监控；外部平台作为后续增强。 | 不阻塞第五阶段。 |

## 14. 交付物清单

| 阶段 | 交付物 | 说明 |
| --- | --- | --- |
| 第一阶段 | 统一 JSON Schema、核心表、统一响应、错误码常量、模型校验 API、草稿保存/查询 API、发布/停用/复制新版本 API、导入导出、发布审计、设计治理审计、基础设计器框架和版本治理前端操作。 | 建立模型、存储、API 和发布治理底座。 |
| 第二阶段 | 规则链加载器、DAG 校验器、ConnectionRouter、`IRuleNodeHandler`、节点工厂、过滤/转换/日志节点、规则链执行 API、限流/超时/熔断/协程池保护、规则链日志、调试预览。 | 打通受保护的规则链最小执行闭环。 |
| 第三阶段 | `WorkflowInstance`、`ActivityHandler`、`ExecutionContext`、`PersistenceService`、建议待办任务表、人工任务服务、会签或签最小策略、补偿策略字段、状态恢复、工作流 API、工作流设计器节点、工作流节点日志。 | 打通工作流长周期、人工协作和补偿闭环。 |
| 第四阶段 | 工作流调用规则链节点、规则链启动工作流节点、`service/cross_engine_service`、`router/engine_router`、trace_id 透传、跨引擎审计字段、前端工具箱隔离、导入 JSON 混用拒绝、端到端测试。 | 建立隔离且可审计的双引擎联动闭环。 |
| 第五阶段 | Cache 热更新、结构化日志、监控指标、容量基线、热冷查询策略、日志归档、慢节点分析、索引优化、安全审计、缓存回源失败验证、性能压测报告。 | 建立生产运维基线。 |

## 15. 阶段任务清单

以下任务使用建议路径表达后续实现落点；当前仓库尚未有代码目录，执行时应先按项目脚手架和 glue 规范创建对应目录。

### 第一阶段任务

- [ ] T001 在建议路径 `model/flow_model.go` 定义统一 JSON Schema 结构，包含 `flowCode`、`type`、`version`、`variables`、`settings`、`nodes`、`connections` 和 JSON 标签。
- [ ] T002 在建议路径 `model/dto/flow_definition.go` 定义流程草稿保存、模型校验、定义查询、发布、停用、复制新版本、导入导出 DTO，避免使用 `any`。
- [ ] T003 在建议路径 `database/migrations/` 编写 `sch_flow_definition` 迁移脚本，字段使用 `varchar`、`datetime`、`varchar(max)`，保留 `flow_code/version/status/is_latest/source_definition_id/published_at/deactivated_at` 字段和索引设计。
- [ ] T004 在建议路径 `database/migrations/` 编写 `sch_workflow_instance` 迁移脚本，包含 `instance_id`、`business_key`、`current_node_id`、`variables`、`state_snapshot`、`compensation_policy` 和时间字段。
- [ ] T005 在建议路径 `database/migrations/` 编写 `sch_workflow_node_log` 迁移脚本，包含 `instance_id`、`node_id`、`trace_id`、`error_code`、`duration_ms`、`compensation_status` 和查询索引。
- [ ] T006 在建议路径 `database/migrations/` 编写 `sch_rulechain_exec_log` 迁移脚本，包含 `exec_id`、`message_id`、`source`、`payload`、`result_data`、`trace_id`、`status`、`sub_code` 和状态索引。
- [ ] T007 在建议路径 `database/migrations/` 编写 `sch_rulechain_node_log` 迁移脚本，包含 `exec_id`、`relation_type`、`from_node_id`、`to_node_ids`、`trace_id`、`sub_code`、`duration_ms` 和慢节点索引。
- [ ] T008 在建议路径 `database/migrations/` 设计建议表 `sch_flow_design_audit_log`，记录保存草稿、发布、停用、复制新版本、导入、导出、权限拒绝等治理操作审计字段和查询索引。
- [ ] T009 在建议路径 `constants/respcode/` 和 `constants/subcode/` 定义基础响应码、模型校验错误、认证错误、权限错误、发布治理错误、限流错误、跨引擎错误、数据库错误和引擎错误常量。
- [ ] T010 在建议路径 `api/response.go` 或 `pkg/response/` 实现统一响应封装，保证成功不输出 `sub_code`，错误必须输出 `sub_code`。
- [ ] T011 在建议路径 `middleware/jwt.go` 明确 JWT `Authorization: Bearer token` 校验边界，并将草稿保存、发布、停用、导入、导出、执行、任务处理等敏感接口纳入认证要求。
- [ ] T012 在建议路径 `middleware/cors.go` 明确 CORS 统一处理边界，避免业务 API 分散配置跨域。
- [ ] T013 在建议路径 `service/model_validator_service.go` 实现模型校验服务，覆盖 JSON 格式、节点引用、连线引用、类型混用、前端导入 JSON 混用拒绝和规则链 DAG 前置校验入口。
- [ ] T014 在建议路径 `api/flow_definition.go` 和 `service/flow_definition_service.go` 实现 `POST /api/basic-distributed/flows/model/validate` 模型校验接口。
- [ ] T015 在建议路径 `api/flow_definition.go`、`service/flow_definition_service.go`、`dao/flow_definition_dao.go` 实现 `POST /api/basic-distributed/flows/drafts/save` 保存草稿接口，并写入设计治理审计。
- [ ] T016 在建议路径 `api/flow_definition.go` 和 `service/publish_service.go` 实现 `POST /api/basic-distributed/flows/:definition_id/publish` 发布接口，校验发布权限、状态流转、模型合法性和返回字段 `definition_id/flow_code/version/status/is_latest/published_at`。
- [ ] T017 在建议路径 `api/flow_definition.go` 和 `service/publish_service.go` 实现 `POST /api/basic-distributed/flows/:definition_id/deactivate` 停用接口，校验停用权限、状态流转和审计记录。
- [ ] T018 在建议路径 `api/flow_definition.go` 和 `service/flow_definition_service.go` 实现 `POST /api/basic-distributed/flows/:definition_id/copy` 复制新版本接口，生成 `DRAFT` 新版本并保留 `source_definition_id`。
- [ ] T019 在建议路径 `api/flow_definition.go` 和 `service/flow_definition_service.go` 实现导入导出接口，导入使用 POST，导出和详情查询使用 GET，路径参数使用 `:definition_id`。
- [ ] T020 在建议路径 `api/flow_definition.go` 和 `dao/flow_definition_dao.go` 实现流程定义列表、详情、版本列表等 GET 查询接口，支持默认最新版本和显式版本查询。
- [ ] T021 在建议路径 `router/basic_distributed.go` 注册 `/api/basic-distributed` API 前缀和 GET/POST 方法边界。
- [ ] T022 在建议路径 `service/audit_service.go` 和 `dao/design_audit_dao.go` 实现发布审计与设计治理审计写入，字段包含 `operation`、`actor_id`、`permission_code`、`flow_code`、`version`、`status_before`、`status_after`、`sub_code`、`trace_id`、`created_at`。
- [ ] T023 在建议路径 `api/audit.go` 和 `dao/design_audit_dao.go` 实现治理审计 GET 查询接口，支持按 `actor_id`、`operation`、`flow_code`、`version`、`status_after`、`sub_code`、`trace_id`、时间范围过滤。
- [ ] T024 在建议路径 `frontend/src/types/flow.ts` 定义统一元模型、版本状态、审计记录 TypeScript 类型，与后端 DTO 字段保持一致。
- [ ] T025 在建议路径 `frontend/src/api/flowDefinition.ts` 封装模型校验、保存草稿、发布、停用、复制新版本、导入、导出、查询定义 API，统一处理 `code/sub_code/message/data`。
- [ ] T026 在建议路径 `frontend/src/views/designer/FlowDesigner.vue` 建立基础设计器页面骨架，包含流程类型选择、模型编辑入口、保存、校验、导入和导出按钮。
- [ ] T027 在建议路径 `frontend/src/views/designer/VersionPanel.vue` 实现前端发布版本、停用版本、复制新版本和默认最新版本操作入口。
- [ ] T028 在建议路径 `frontend/src/views/audit/DesignAuditLog.vue` 实现设计治理审计查询页面，展示操作类型、操作者、权限码、状态变化、`sub_code` 和 `trace_id`。
- [ ] T029 在建议路径 `tests/`、`api/*_test.go`、`dao/*_test.go` 增加第一阶段基础验收测试，覆盖统一响应、模型校验、DAO 参数化查询和无硬编码密钥扫描。
- [ ] T030 在建议路径 `tests/`、`api/*_test.go` 增加发布治理验收测试，断言接口请求、返回字段、表记录、状态枚举变化、日志字段、权限拒绝 `sub_code` 和审计查询结果。

### 第二阶段任务

- [ ] T031 在建议路径 `engine/rulechain/loader.go` 实现规则链模型加载器，从已发布 `RULE_CHAIN` 定义构建节点表、连线表和邻接表。
- [ ] T032 在建议路径 `engine/rulechain/dag_validator.go` 实现 Kahn 拓扑排序 DAG 校验，确保规则链禁止循环。
- [ ] T033 在建议路径 `engine/rulechain/connection_router.go` 实现 ConnectionRouter，按 `relationType`、条件和默认连线选择后继节点。
- [ ] T034 在建议路径 `engine/rulechain/context.go` 定义单次执行独立上下文，包含 payload、变量、节点输出、`trace_id` 和执行状态。
- [ ] T035 在建议路径 `engine/rulechain/node/handler.go` 定义 `IRuleNodeHandler` 接口，约束节点输入、输出、错误和 `sub_code` 映射。
- [ ] T036 在建议路径 `engine/rulechain/node/factory.go` 实现节点工厂和注册表，支持按节点类型获取处理器。
- [ ] T037 在建议路径 `engine/rulechain/node/filter.go` 实现基础过滤节点，支持基于配置条件返回 SUCCESS/FAILURE。
- [ ] T038 在建议路径 `engine/rulechain/node/transform.go` 实现基础转换节点，支持输入字段映射和输出数据写回上下文。
- [ ] T039 在建议路径 `engine/rulechain/node/log.go` 实现基础日志节点，按脱敏规则输出结构化日志。
- [ ] T040 在建议路径 `engine/rulechain/executor.go` 实现 RuleChainExecutor，同步执行已发布规则链并返回结果。
- [ ] T041 在建议路径 `service/rulechain_guard_service.go` 和 `engine/common/execution_guard.go` 设计执行入口限流、超时、熔断、协程池/并发保护策略和配置项。
- [ ] T042 在建议路径 `api/rulechain.go` 和 `service/rulechain_service.go` 实现 `POST /api/basic-distributed/rulechains/execute` 执行接口，强制校验 `PUBLISHED` 状态、默认最新版本、显式版本和执行权限。
- [ ] T043 在建议路径 `dao/rulechain_log_dao.go` 实现 `sch_rulechain_exec_log` 与 `sch_rulechain_node_log` 写入，字段包含 `trace_id`、`status`、`error_code`、`sub_code`、`duration_ms`。
- [ ] T044 在建议路径 `frontend/src/views/rulechain/DebugPreview.vue` 实现规则链调试预览页面，展示模拟输入、路由路径、节点输出、错误信息、限流和超时提示。
- [ ] T045 在建议路径 `frontend/src/api/rulechain.ts` 封装规则链执行和调试预览 API。
- [ ] T046 在建议路径 `engine/rulechain/*_test.go` 和 `api/rulechain_test.go` 增加 DAG 环路拒绝、ConnectionRouter、基础节点、版本状态和执行 API 测试。
- [ ] T047 在建议路径 `tests/performance/` 和 `api/rulechain_test.go` 增加限流、超时、熔断、协程池容量、并发保护、限流 `sub_code` 和小规模压测验证任务。

### 第三阶段任务

- [ ] T048 在建议路径 `engine/workflow/instance.go` 定义 `WorkflowInstance`，管理实例 ID、流程编码、版本、业务键、状态、当前节点、变量和补偿策略摘要。
- [ ] T049 在建议路径 `engine/workflow/activity_handler.go` 定义 `ActivityHandler` 接口，统一工作流节点处理入口和错误映射。
- [ ] T050 在建议路径 `engine/workflow/execution_context.go` 定义 `ExecutionContext`，承载变量、状态快照、当前节点、操作者、权限字段和 `trace_id`。
- [ ] T051 在建议路径 `engine/workflow/persistence_service.go` 实现 `PersistenceService`，保存和加载实例状态、变量、当前节点、待办引用和快照。
- [ ] T052 在建议路径 `service/workflow_service.go` 实现工作流启动编排，创建实例并写入初始节点日志。
- [ ] T053 在建议路径 `dao/workflow_instance_dao.go` 实现 `sch_workflow_instance` 参数化写入、按 `instance_id` 查询、按 `business_key` 查询和分页查询。
- [ ] T054 在建议路径 `dao/workflow_log_dao.go` 实现 `sch_workflow_node_log` 写入和按 `instance_id`、`trace_id` 查询。
- [ ] T055 在建议路径 `database/migrations/` 设计建议表 `sch_workflow_task`，字段包含 `task_id`、`instance_id`、`node_id`、`assignee_id`、`candidate_group`、`permission_code`、`claim_status`、`task_status`、`approval_policy`、`approval_result`、`due_at`、`claimed_at`、`completed_at`、`trace_id`、`created_at`、`updated_at`，索引覆盖待办、候选组、实例节点和 `trace_id` 查询。
- [ ] T056 在建议路径 `service/workflow_task_service.go` 和 `dao/workflow_task_dao.go` 实现人工任务创建、查询、领取、处理、取消、过期和状态更新。
- [ ] T057 在建议路径 `api/workflow_task.go` 实现待办列表和详情 GET 接口，按 `assignee_id`、`candidate_group`、`permission_code`、`task_status` 和时间范围过滤。
- [ ] T058 在建议路径 `api/workflow_task.go` 实现人工任务处理 POST 接口，确保重复处理、无权限处理、过期待办处理返回稳定 `sub_code`。
- [ ] T059 在建议路径 `engine/workflow/node/approval.go` 或 `service/workflow_task_service.go` 实现会签全员完成和或签任一通过的最小策略。
- [ ] T060 在建议路径 `engine/workflow/compensation.go` 和 `service/workflow_service.go` 设计补偿策略字段、失败补偿路径、补偿失败转人工和失败终止策略。
- [ ] T061 在建议路径 `api/workflow.go` 实现 `POST /api/basic-distributed/workflows/start` 工作流启动接口，强制校验 `PUBLISHED` 状态和启动权限。
- [ ] T062 在建议路径 `api/workflow.go` 实现 `GET /api/basic-distributed/workflows/instances` 和 `GET /api/basic-distributed/workflows/instances/:instance_id` 查询接口。
- [ ] T063 在建议路径 `frontend/src/views/workflow/Designer.vue` 增加工作流节点工具箱，包含开始、结束、人工任务、网关、会签、或签和补偿类节点入口。
- [ ] T064 在建议路径 `frontend/src/components/workflow/PropertyPanel.vue` 实现工作流属性面板，支持权限、表单、超时、输入输出映射、审批策略和补偿策略配置。
- [ ] T065 在建议路径 `frontend/src/views/workflow/InstanceDetail.vue` 和 `TaskHandle.vue` 实现实例详情和人工任务处理页面。
- [ ] T066 在建议路径 `engine/workflow/*_test.go`、`api/workflow_test.go` 增加启动、等待、人工处理、待办表记录、权限字段、状态恢复和节点日志测试。
- [ ] T067 在建议路径 `engine/workflow/*_test.go`、`api/workflow_task_test.go` 增加会签、或签、补偿路径、重复处理、权限拒绝 `sub_code` 和服务重启恢复步骤测试。

### 第四阶段任务

- [ ] T068 在建议路径 `engine/workflow/node/rulechain_call.go` 实现工作流规则链调用标准节点，支持目标 `flow_code`、版本策略、输入映射、输出映射和失败策略，但只能调用 `service/cross_engine_service`，禁止依赖 `engine/rulechain` 内部类型。
- [ ] T069 在建议路径 `engine/rulechain/node/start_workflow.go` 实现规则链启动工作流标准节点，支持目标工作流、`business_key`、变量映射和启动模式，但只能调用 `service/cross_engine_service`，禁止依赖 `engine/workflow` 内部类型。
- [ ] T070 在建议路径 `service/cross_engine_service.go` 实现跨引擎标准调用编排，禁止共享内部上下文，只传标准 payload/result/error。
- [ ] T071 在建议路径 `router/engine_router.go` 扩展引擎路由，按 `flow_type`、版本、状态、权限和发布状态选择执行入口。
- [ ] T072 在建议路径 `middleware/trace.go` 和 `log/cross_engine.go` 确保跨引擎调用透传调用方 `trace_id`，禁止生成孤立链路号。
- [ ] T073 在建议路径 `service/audit_service.go` 实现跨引擎权限审计，字段包含 `direction`、`caller_engine`、`callee_engine`、`caller_instance_id` 或 `caller_exec_id`、`callee_instance_id` 或 `callee_exec_id`、`standard_node_id`、`actor_id`、`permission_code`、`idempotency_key`、`duration_ms`、`sub_code`、`trace_id`。
- [ ] T074 在建议路径 `frontend/src/components/designer/nodes/RulechainCallNode.vue` 和 `StartWorkflowNode.vue` 增加跨引擎标准节点配置 UI。
- [ ] T075 在建议路径 `frontend/src/components/designer/NodeToolbox.vue` 或等价组件实现前端工具箱模式隔离，工作流和规则链只展示本模式节点及允许的标准跨引擎节点。
- [ ] T076 在建议路径 `tests/e2e/cross_engine/` 增加工作流调用规则链端到端用例，验证结果写回变量并继续流转。
- [ ] T077 在建议路径 `tests/e2e/cross_engine/` 增加规则链启动工作流端到端用例，验证返回 `instance_id` 并写入双方日志。
- [ ] T078 在建议路径 `tests/e2e/cross_engine/` 增加跨引擎无权限、超时、定义未发布、版本不存在、幂等键重复和权限拒绝 `sub_code` 测试。
- [ ] T079 在建议路径 `tests/`、`api/flow_definition_test.go` 增加导入 JSON 跨模式节点混用拒绝和后端发布校验拒绝测试。
- [ ] T080 在建议路径 `tests/` 或静态检查脚本中增加依赖扫描，确认 `engine/workflow` 与 `engine/rulechain` 不互相直接 import 内部类型，跨引擎只经 `service/cross_engine_service` 和 `router/engine_router`。

### 第五阶段任务

- [ ] T081 在建议路径 `cache/flow_definition_cache.go` 和 `service/publish_service.go` 实现流程定义、规则链邻接表和节点处理器元数据的 glue Cache 热更新。
- [ ] T082 在建议路径 `log/fields.go` 统一结构化日志字段，包含 `trace_id`、用户、接口、`flow_code`、版本、节点、耗时、`sub_code`、发布审计字段和跨引擎审计字段。
- [ ] T083 在建议路径 `middleware/trace.go` 完善 API 入口 trace 生成和透传，确保执行链路、发布治理、权限拒绝和跨引擎调用全程可关联。
- [ ] T084 在建议路径 `monitor/rulechain_metrics.go` 采集规则链 QPS、P95/P99、节点失败率、限流数量、熔断数量和超时数量。
- [ ] T085 在建议路径 `monitor/workflow_metrics.go` 采集工作流待办量、超时量、运行中实例数、补偿失败量和恢复失败量。
- [ ] T086 在建议路径 `docs/performance/capacity-baseline.md` 记录容量基线，覆盖规则链执行总表、规则链节点明细、工作流实例、工作流节点日志、归档数据量、并发数、分页大小、查询时间阈值和 P95/P99 采样窗口。
- [ ] T087 在建议路径 `service/log_archive_service.go` 和 `dao/log_archive_dao.go` 实现日志归档服务，明确热表与归档表查询策略、同一 API 是否跨热/冷数据、分页排序规则和保留 `trace_id`、`flow_code`、`exec_id`、`instance_id` 查询键。
- [ ] T088 在建议路径 `service/slow_node_service.go` 和 `api/monitor.go` 实现慢节点分析接口，支持按 `flow_code`、`node_type`、`duration_ms` 排序查询。
- [ ] T089 在建议路径 `dao/` 审核并优化核心表索引查询，重点覆盖 `trace_id`、`flow_code/status/time`、`instance_id`、`exec_id` 查询路径，并记录超过 1000ms 的慢查询阈值、SQL 标识、参数摘要和 `trace_id`。
- [ ] T090 在建议路径 `service/security_audit_service.go` 或安全检查脚本中检查硬编码密钥、Token、密码、不安全 SQL、敏感日志输出、JWT/CORS/权限绕过和参数化 SQL。
- [ ] T091 在建议路径 `frontend/src/views/monitor/ExecutionLog.vue` 实现执行链路查询页面，支持按 `trace_id`、状态、时间、流程编码、`exec_id`、`instance_id` 过滤，并标识热表或归档来源。
- [ ] T092 在建议路径 `frontend/src/views/monitor/SlowNodeAnalysis.vue` 实现慢节点分析页面，展示节点耗时、失败率、限流数量、超时数量和趋势。
- [ ] T093 在建议路径 `tests/performance/` 增加规则链并发执行、入口限流、工作流恢复、日志查询分页、热冷查询和容量基线压测脚本。
- [ ] T094 在建议路径 `tests/` 或 `tests/performance/` 增加缓存热更新验收，覆盖发布新版本、停用版本、默认最新版本、显式旧版本执行、规则链邻接表刷新、节点处理器元数据刷新和回源失败验证。
- [ ] T095 在建议路径 `tests/` 或 `tests/performance/` 增加归档与索引验收，断言热表查询、归档表查询、跨热冷查询开关、稳定分页排序、`trace_id/exec_id/instance_id` 查询索引命中和慢查询阈值记录。
- [ ] T096 在建议路径 `docs/performance/load-test-report.md` 记录压测结果、容量结论、P95/P99、错误率、限流率、瓶颈和优化建议。
- [ ] T097 在建议路径 `docs/security/audit-review.md` 记录安全审计结果、风险等级、整改结论、权限拒绝 `sub_code` 验收和无硬编码密钥结论。
