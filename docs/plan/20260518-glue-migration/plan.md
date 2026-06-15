# glue-microservice 一次性迁移详细计划（2026-05-18）

## Wave 1：架构与基础设施准备
- 明确 glue-microservice 架构落地方案，完成 glue 依赖、xdb 配置、基础目录结构调整，主流程可编译、可启动。
- 任务：方案设计与评审、glue 依赖引入、xdb 配置、目录结构重组。
- 回归点：go build 通过，glue 服务可正常启动。

## Wave 2：DAO/sysrepo → glue xdb 重构
- 所有数据访问层迁移为 glue xdb Repo，保证数据模型、SQL、事务、错误处理兼容。
- 任务：sysmodel 结构体梳理、DAO/sysrepo 迁移为 glue Repo、xdb 配置与依赖注入、单元/集成测试。
- 回归点：Repo 层接口功能与原有一致，SQL/事务/错误处理无回归，单元测试通过。

## Wave 3：Service/Handler 层 glue 化与接口兼容
- Service/Handler 全部 glue 化，接口标准化，保证前端接口结构/字段/状态码兼容。
- 任务：Service glue Service 重构、Handler glue Handler 重构、标准返回/错误/认证 glue 化、路由注册 glue 化、自动化接口测试/前后端联调。
- 回归点：接口结构/字段/状态码与前端兼容，权限/认证/错误处理无回归，前后端联调通过。

## Wave 4：安全合规、最佳实践、回归验证
- 全面安全合规检查、glue 最佳实践落地，回归测试、文档补全、交付。
- 任务：安全合规检查、glue 最佳实践自查、自动化/回归测试、技术/接口/迁移文档补全。
- 回归点：安全合规无高危问题，glue 关键用法全部落地，自动化测试全通过，文档交付完整。

## 关键说明与风险提示
- 每阶段需产出可验证交付物，接口兼容点需与前端同步，接口变更需联调。
- glue 最佳实践：依赖注入、标准返回、错误优先、JWT 认证、Repo/Service/Handler 分层、禁止硬编码敏感信息。
- 主要风险：SQL/事务遗漏、路由/依赖注入遗漏、错误处理/返回格式不统一、安全合规未覆盖、自动化测试覆盖不足。

## Wave2 进展记录（2026-05-18）
- 主路径已切换到 `internal/repository/xdb`：当前 API 启动注入使用 `xdb.NewDB(db)`，系统管理 Repo 来源为 xdb 实现。
- 旧 `internal/dao` 实现已清理：`config-management/backend/internal/dao/repository.go` 已删除。
- 兼容性说明：`sysrepo` 接口契约保持不变，Service/Handler 调用路径无需额外改动。

## 执行结果（2026-05-18）

### 1) Wave1~Wave4 已完成项（简明）
- Wave1：完成 glue 架构落地、依赖与目录调整，后端主流程可编译、可启动。
- Wave2：完成 DAO/sysrepo 到 xdb Repo 的主路径迁移，旧 dao 主实现已清理，接口契约保持兼容。
- Wave3：完成 Service/Handler glue 化改造与路由链路打通，保持前端调用路径可用。
- Wave4：完成关键安全项修复与构建/测试回归验证，当前无阻断发布问题。

### 2) 关键安全修复
- 登录验密修复：统一使用 bcrypt 校验口令，不再接受明文直比。
- 初始化管理员密码修复：统一按 bcrypt 哈希入库，避免明文或不一致哈希策略。
- 配置强校验修复：DB_DSN、JWT_SECRET 设为必填，缺失时启动失败并给出明确错误。

### 3) 当前已知非阻断项
- 分页 total 返回结构存在历史兼容差异（字段组织形式与旧返回不完全一致），当前不阻断主流程；建议在后续小版本统一返回契约并补充前后端对齐说明。

### 4) 验证结果摘要
- go build ./config-management/backend/cmd/api：通过。
- go test ./config-management/backend/...：通过。
