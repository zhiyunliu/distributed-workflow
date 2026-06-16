<!--
Sync Impact Report
版本变更: 未正式采纳的模板 -> 1.0.0
修改的原则:
- 模板原则 1 -> I. 统一元模型与存储优先
- 模板原则 2 -> II. 双引擎隔离与标准联动
- 模板原则 3 -> III. glue 框架优先与依赖精简
- 模板原则 4 -> IV. SQL Server 企业级持久化与审计
- 模板原则 5 -> V. 可测试、可观测、可演进
新增章节:
- 技术与数据约束
- 开发流程与质量门禁
- Governance 修订与合规策略
移除章节:
- 模板占位注释和仅供示例的说明
需要同步的模板:
- ✅ .specify/templates/plan-template.md
- ✅ .specify/templates/spec-template.md
- ✅ .specify/templates/tasks-template.md
- ✅ .specify/templates/commands/*.md (directory not present; no files to update)
已检查的运行指导:
- ✅ README.md (未发现过时宪章引用)
- ✅ docs/requirement.md
- ✅ docs/architecture-design.md
后续 TODO: 无
-->

# distributed-workflow Constitution

## Core Principles

### I. 统一元模型与存储优先
所有流程定义 MUST 以统一 JSON Schema 表达，节点集合 `nodes` 与显式边集
`connections` 是唯一持久化标准。任何新功能在进入引擎实现前，MUST 先说明
对 `flow_type`、`flow_code`、版本、节点、连线、变量和配置字段的影响。公开 JSON
键名 MUST 使用小写下划线，例如 `from_id`、`to_id`、`relation_type`、`is_default`。
这样做可以保证设计器、版本审计、SQL Server 存储和双引擎加载拥有同一份事实来源。

### II. 双引擎隔离与标准联动
`WORKFLOW` 与 `RULE_CHAIN` 的运行时上下文、调度器、节点生命周期和持久化策略 MUST
保持隔离。跨引擎调用只能通过标准节点完成：工作流调用规则链节点、规则链启动工作流节点；
不得直接跳转对方节点或共享内部上下文。规则链 MUST 通过 DAG 校验并禁止循环；循环、回退、
人工等待和事务补偿 MUST 由工作流能力承载。该边界防止执行语义混乱，并让故障、扩容和审计
可以按引擎独立治理。

### III. glue 框架优先与依赖精简
后端服务、API 路由、依赖注入、中间件、缓存、日志、监控和服务治理 MUST 优先使用
`github.com/zhiyunliu/glue` 及其原生能力。规则链内核 MUST 使用 Go 自研实现，不得引入第三方
规则链引擎作为核心执行依赖。新增后端功能 MUST 遵循 glue 微服务分层实践，管理后台能力放在
`management` 目录，功能代码放在根目录对应领域包中。该原则保证平台自主可控，并减少重复建设
基础设施。

### IV. SQL Server 企业级持久化与审计
持久化设计 MUST 面向 SQL Server，业务表使用 `sch_` 前缀，时间字段使用 `datetime2(3)`，
JSON 内容使用 `nvarchar(max)` 并配合 `ISJSON` 或等价校验。常用查询维度 MUST 结构化列化，
不得依赖大 JSON 字段全量扫描。流程定义、工作流实例、规则链执行、节点明细、跨引擎调用和
权限相关操作 MUST 记录可追踪审计数据，并携带 `trace_id` 或等价链路标识。该原则保证长周期
业务流程可恢复、高频事件链路可追踪、企业审计可落地。

### V. 可测试、可观测、可演进
每个功能切片 MUST 明确独立验收方式，并覆盖其模型校验、路由分发、引擎边界、数据库读写和
错误响应。涉及统一模型、DAG、跨引擎联动、版本发布、权限、审计或公共 API 的变更 MUST 提供
对应测试或手工验证证据。运行路径 MUST 产生结构化日志、错误码、子码和链路信息；失败场景
MUST 有中文用户消息与稳定的 `code` / `sub_code` 常量。该原则使平台在扩展节点、升级模型和
调整执行策略时保持可验证、可诊断和可回滚。

## 技术与数据约束

- Go 后端 MUST 使用 Go 1.24+ 与 glue 框架；错误显式处理，禁止忽略 `error`。
- 后端分层 MUST 保持 `api`、`service`、`dao`、`model`、`engine`、`router` 等职责边界；
  `api` 不写业务编排和 SQL，`service` 不拼接 SQL，`engine` 不泄漏内部上下文到 API 层。
- HTTP 查询类接口 MUST 使用 GET；创建、修改、删除、发布、执行、调试类接口 MUST 使用 POST。
- API 响应 MUST 使用统一结构：成功返回 `code`、`message`、可选 `data`；错误返回
  `code`、`sub_code`、`message`。所有 `code` 与 `sub_code` MUST 在 `constants/respcode`
  和 `constants/subcode` 中定义。
- Vue 管理端 MUST 使用 Vue 3、Vite、TypeScript 严格模式、Element Plus、Pinia 与 Axios
  封装；页面放在 `views`，公共组件放在 `components`，接口封装放在 `src/api`。
- 前后端公开契约、数据库 JSON、导入导出模型和文档示例 MUST 使用小写下划线 JSON 键名。
- 禁止硬编码密钥、密码、Token；敏感配置 MUST 使用环境变量或配置文件。

## 开发流程与质量门禁

- 规格阶段 MUST 标明功能所属流程类型：`WORKFLOW`、`RULE_CHAIN`、跨引擎联动或纯管理能力。
- 计划阶段 MUST 通过宪章检查，说明统一元模型影响、引擎边界、glue 能力复用、SQL Server
  持久化、审计、测试和观测策略。
- 任务阶段 MUST 将模型、常量、数据库、服务、API、前端、测试、文档拆成可追踪任务，并映射到
  用户故事或质量门禁。
- 实现阶段 MUST 先完成模型和校验，再实现持久化、业务编排、引擎逻辑、API、前端交互和观测。
- 任何公共契约变更 MUST 同步更新规格、计划、任务、文档示例和验证证据。
- 禁止修改 `go.mod`、`vite.config.ts` 或核心配置文件，除非任务明确要求并记录原因。

## Governance

本宪章约束 distributed-workflow 的全部 SpecKit 产物和实现工作。当它与生成的规格、计划或任务
冲突时，以本宪章为准。`AGENTS.md` 与 `.github/copilot-instructions.md` 中更具体且不与本宪章
冲突的仓库规则仍然有效。

修订 MUST 包含变更原因、受影响的原则或章节、需要同步的模板和文档，以及 Sync Impact Report。
版本号遵循语义化版本：MAJOR 用于不向后兼容的治理或原则重定义，MINOR 用于新增或实质扩展原则，
PATCH 用于不改变强制行为的澄清。

规格、计划、任务生成、实现评审和发布准备阶段 MUST 审查宪章合规性。任何例外 MUST 在相关计划的
Complexity Tracking 中记录，并提供有边界的缓解或迁移路径。

**Version**: 1.0.0 | **Ratified**: 2026-06-16 | **Last Amended**: 2026-06-16
