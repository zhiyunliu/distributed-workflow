# Wave1 架构设计（glue-microservice + xdb 预备层）

## 1. 目标与边界
- 目标：在不改变现有 API 接口行为与 Gin 路由逻辑的前提下，为后续 glue-microservice + xdb 渐进迁移建立可回滚的启动与目录骨架。
- 边界：本轮不替换现有 handler/service/dao 实现，不引入强制生效的 glue 启动流程，不变更 go.mod。

## 2. 目标架构图（文字描述）
- 当前（legacy）：`cmd/api/main.go -> handler(Server, Gin) -> service -> dao -> sqlserver`。
- 目标（未来）：`cmd/api/main.go -> bootstrap(App) -> app/modules(api/rpc/cron/mqc) -> repository/xdb -> sqlserver`。
- Wave1 状态：
  - 启动链路仍由 legacy Gin 路径负责。
  - 通过 `APP_MODE=legacy|glue` 预留模式切换点。
  - `glue` 模式当前只输出明确日志并回退到 `legacy`，不影响线上行为。

## 3. 目录映射
- 现有目录（保留）：
  - `internal/handler`：Gin 路由与 HTTP 适配层。
  - `internal/service`：业务服务层。
  - `internal/dao`：数据库访问层。
- Wave1 新增骨架：
  - `internal/app/`：未来 glue 应用模块聚合入口（API/RPC/CRON/MQC）。
  - `internal/bootstrap/`：未来启动装配、依赖注入与生命周期管理。
  - `internal/repository/xdb/`：未来基于 xdb 的仓储实现与适配层。

## 4. 依赖注入策略
- 短期（Wave1）：沿用 `main.go` 手工注入方式，确保 legacy 行为零变化。
- 中期（Wave2+）：将连接、仓储、服务、鉴权配置迁移至 `bootstrap`，由启动装配统一创建并注入到 `app` 模块。
- 长期：
  - 抽象仓储接口，支持 `dao` 与 `repository/xdb` 双实现并行。
  - 通过模式开关控制注入图，逐步替换而非一次性迁移。

## 5. 配置策略
- 统一配置目标：支持「环境变量优先 + 文件配置兜底 + 合理默认值」。
- Wave1 提供样例：`config-management/backend/configs/glue.example.yaml`。
- 建议优先级：
  1. 环境变量（最高优先级，便于容器化与多环境）
  2. glue 配置文件
  3. 内置默认值（仅非敏感字段）
- 敏感字段策略：如 `jwt.secret`、`xdb.sqlserver.dsn` 不提供真实默认值，示例中仅占位。

## 6. 分阶段切换策略
- Phase 0（当前 Wave1）：
  - 仅增加目录与配置骨架。
  - 引入 `APP_MODE` 启动开关，默认 `legacy`。
  - `glue` 模式不生效业务，仅日志提示并回退。
- Phase 1：在 `bootstrap` 中接入 glue 启动最小闭环（健康检查、基础模块装配）。
- Phase 2：将部分读接口或低风险模块迁移至 glue/xdb，保留 legacy 回退。
- Phase 3：核心链路迁移与双栈验证，稳定后再清理 legacy 过渡代码。

## 7. 风险清单
- 风险1：双启动链路并存导致配置分叉。
  - 缓解：明确配置优先级，集中到 `bootstrap` 统一解析。
- 风险2：数据库访问层并行（dao 与 xdb）导致行为不一致。
  - 缓解：先迁移只读路径并增加回归测试，保持接口契约不变。
- 风险3：模式开关误配置导致运行预期偏差。
  - 缓解：默认 `legacy`，并对非法值打印告警且回退 `legacy`。
- 风险4：迁移期排障复杂度上升。
  - 缓解：统一日志字段（模式、配置来源、组件初始化状态）。
