# distributed-workflow

分布式工作流引擎 —— 基于 Go 语言实现的轻量级、可扩展的 DAG 工作流调度系统。

## 架构概览

```
┌─────────────────────────────────────────────────────┐
│                  HTTP API (Gin)                      │
│           endpoint/http  :8080                       │
└──────────────────────┬──────────────────────────────┘
                       │
┌──────────────────────▼──────────────────────────────┐
│                   Engine (调度中心)                   │
│  WorkflowService │ SchedulerService │ WorkerManager  │
│  InstanceStateService │ gRPC Server :9090            │
└──────────┬──────────────────────────────────────────┘
           │  gRPC (AssignTask / ReportTaskResult)
┌──────────▼──────────────────────────────────────────┐
│              NodeWorker (执行节点，可多实例)           │
│  TaskExecutionManager │ NodeExecutorContainer        │
│  gRPC Server :9091                                   │
└─────────────────────────────────────────────────────┘
```

**存储依赖**：SQL Server（持久化）、Redis（分布式锁 + 任务队列）  
**可选依赖**：Nacos（服务注册与发现）

## 目录结构

```
├── types/           # 核心数据结构与枚举
├── proto/           # gRPC Protocol Buffers 定义及生成代码
├── interfaces/      # 所有核心接口定义
├── dag/             # DAG 解析器（环检测、就绪节点计算）
├── rpc/             # Engine↔Worker gRPC 客户端
├── engine/          # 调度中心实现
├── worker/          # 执行节点实现
│   └── executors/   # 内置执行器（log, sleep, http）
├── endpoint/http/   # Gin HTTP API 服务
├── storage/
│   ├── sqlserver/   # SQL Server 持久化实现
│   └── redis/       # Redis 分布式锁 + 任务队列
├── registry/nacos/  # Nacos 服务注册发现
└── examples/
    ├── serial/      # 串行工作流示例 A→B→C
    ├── parallel/    # 并行工作流示例 A→(B|C)→D
    └── branch/      # 分支工作流示例（成功/失败路由）
```

## 快速开始

### 前置条件

1. Go 1.21+
2. SQL Server（执行 `storage/sqlserver/schema.sql` 建表）
3. Redis

### 运行串行示例

```bash
# 修改 DSN 和 Redis 地址后运行
go run ./examples/serial/
```

### HTTP API

| 方法   | 路径                                    | 说明             |
|--------|-----------------------------------------|------------------|
| POST   | `/api/v1/workflows`                     | 创建工作流定义   |
| GET    | `/api/v1/workflows/:workflowId`         | 获取工作流定义   |
| PUT    | `/api/v1/workflows/:workflowId`         | 更新工作流定义   |
| DELETE | `/api/v1/workflows/:workflowId`         | 删除工作流定义   |
| POST   | `/api/v1/workflows/:workflowId/start`   | 启动工作流实例   |
| GET    | `/api/v1/instances/:instanceId`         | 获取实例状态     |

### 工作流定义示例（串行 A→B→C）

```json
{
  "id": "my-workflow",
  "name": "我的串行工作流",
  "startNodeId": "node-a",
  "endNodeId": "node-c",
  "nodes": {
    "node-a": {"id": "node-a", "type": "log", "name": "步骤A", "config": {"message": "执行A"}},
    "node-b": {"id": "node-b", "type": "sleep", "name": "步骤B", "config": {"duration_ms": 1000}},
    "node-c": {"id": "node-c", "type": "log", "name": "步骤C", "config": {"message": "执行C"}}
  },
  "connections": [
    {"id": "c1", "sourceId": "node-a", "targetId": "node-b", "type": "success"},
    {"id": "c2", "sourceId": "node-b", "targetId": "node-c", "type": "success"}
  ]
}
```

## 内置执行器

| 类型    | 说明           | 配置字段                              |
|---------|----------------|---------------------------------------|
| `log`   | 日志输出       | `message`（必填），`level`（可选）    |
| `sleep` | 等待指定时间   | `duration_ms`（毫秒）                 |
| `http`  | 发送 HTTP 请求 | `url`（必填），`method`，`timeout_ms`，`headers`，`body` |

## 自定义执行器

实现 `interfaces.NodeExecutor` 接口并注册到 `NodeExecutorContainer`：

```go
type MyExecutor struct{}

func (e *MyExecutor) Type() string { return "my-type" }
func (e *MyExecutor) Execute(ctx context.Context, task *types.Task) (*types.TaskResult, error) {
    // 执行逻辑
    return &types.TaskResult{Success: true}, nil
}

container.Register(&MyExecutor{})
```

## 依赖项

- `google.golang.org/grpc` — Engine↔Worker 通信
- `github.com/microsoft/go-mssqldb` — SQL Server 驱动
- `github.com/redis/go-redis/v9` — Redis 客户端
- `github.com/gin-gonic/gin` — HTTP API 框架
- `github.com/nacos-group/nacos-sdk-go/v2` — 服务注册发现（可选）
- `github.com/rs/zerolog` — 结构化日志
- `github.com/google/uuid` — UUID 生成
