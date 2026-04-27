-- DistributedWorkflow SQL Server 初始化脚本
-- 版本：v1.0 (D1阶段)
-- 兼容：SQL Server 2019+

-- ─────────────────────────────────────────────────────────────────────────────
-- 创建工作流定义表
-- ─────────────────────────────────────────────────────────────────────────────
IF NOT EXISTS (SELECT * FROM sys.tables WHERE name = 'workflow_defs')
BEGIN
    CREATE TABLE workflow_defs (
        -- 工作流唯一标识
        id          VARCHAR(64)     NOT NULL,
        -- 工作流名称
        name        VARCHAR(255)    NOT NULL,
        -- 工作流描述
        description NVARCHAR(MAX)   NULL,
        -- 最新版本号，D1阶段固定为1
        latest_version INT          NOT NULL DEFAULT 1,
        -- 工作流定义 JSON 字符串（完整 WorkflowDef）
        definition  NVARCHAR(MAX)   NOT NULL,
        -- 创建时间
        created_at  DATETIME2       NOT NULL DEFAULT GETDATE(),
        -- 更新时间
        updated_at  DATETIME2       NOT NULL DEFAULT GETDATE(),
        -- 创建人
        created_by  VARCHAR(64)     NOT NULL DEFAULT '',
        -- 更新人
        updated_by  VARCHAR(64)     NOT NULL DEFAULT '',
        -- 是否禁用：0=启用，1=禁用
        disabled    BIT             NOT NULL DEFAULT 0,

        CONSTRAINT pk_workflow_defs PRIMARY KEY (id)
    );
END;
GO

-- ─────────────────────────────────────────────────────────────────────────────
-- 创建工作流实例表
-- ─────────────────────────────────────────────────────────────────────────────
IF NOT EXISTS (SELECT * FROM sys.tables WHERE name = 'workflow_instances')
BEGIN
    CREATE TABLE workflow_instances (
        -- 实例唯一标识
        id               VARCHAR(64)     NOT NULL,
        -- 关联的工作流定义 ID
        workflow_id      VARCHAR(64)     NOT NULL,
        -- 工作流版本号，D1阶段固定为1
        workflow_version INT             NOT NULL DEFAULT 1,
        -- 实例状态：pending/running/completed/failed
        status           VARCHAR(32)     NOT NULL,
        -- 启动时间
        start_time       DATETIME2       NOT NULL DEFAULT GETDATE(),
        -- 结束时间
        end_time         DATETIME2       NULL,
        -- 输入参数 JSON 字符串
        input_data       NVARCHAR(MAX)   NULL,
        -- 输出结果 JSON 字符串
        output_data      NVARCHAR(MAX)   NULL,
        -- 创建人
        created_by       VARCHAR(64)     NOT NULL DEFAULT '',
        -- 失败错误信息
        error_message    NVARCHAR(MAX)   NULL,

        CONSTRAINT pk_workflow_instances PRIMARY KEY (id),
        CONSTRAINT fk_workflow_instances_workflow_id FOREIGN KEY (workflow_id)
            REFERENCES workflow_defs(id)
    );
END;
GO

-- ─────────────────────────────────────────────────────────────────────────────
-- 创建节点执行状态表
-- ─────────────────────────────────────────────────────────────────────────────
IF NOT EXISTS (SELECT * FROM sys.tables WHERE name = 'workflow_node_states')
BEGIN
    CREATE TABLE workflow_node_states (
        -- 自增主键
        id            INT             NOT NULL IDENTITY(1,1),
        -- 关联的流程实例 ID
        instance_id   VARCHAR(64)     NOT NULL,
        -- 节点 ID
        node_id       VARCHAR(64)     NOT NULL,
        -- 节点状态：pending/assigned/running/completed/failed
        status        VARCHAR(32)     NOT NULL,
        -- 分配的 Worker ID
        assigned_to   VARCHAR(128)    NULL,
        -- 分配的 Worker 服务器 IP 地址
        worker_ip     VARCHAR(64)     NULL,
        -- 执行开始时间
        start_time    DATETIME2       NULL,
        -- 执行结束时间
        end_time      DATETIME2       NULL,
        -- 节点输入数据 JSON
        input_data    NVARCHAR(MAX)   NULL,
        -- 节点输出数据 JSON
        output_data   NVARCHAR(MAX)   NULL,
        -- 失败错误信息
        error_message NVARCHAR(MAX)   NULL,
        -- 已重试次数
        retry_count   INT             NOT NULL DEFAULT 0,

        CONSTRAINT pk_workflow_node_states PRIMARY KEY (id),
        CONSTRAINT uq_workflow_node_states UNIQUE (instance_id, node_id),
        CONSTRAINT fk_workflow_node_states_instance_id FOREIGN KEY (instance_id)
            REFERENCES workflow_instances(id)
    );
END;
GO

-- ─────────────────────────────────────────────────────────────────────────────
-- 创建流程实例上下文表
-- ─────────────────────────────────────────────────────────────────────────────
IF NOT EXISTS (SELECT * FROM sys.tables WHERE name = 'workflow_instance_context')
BEGIN
    CREATE TABLE workflow_instance_context (
        -- 自增主键
        id          INT             NOT NULL IDENTITY(1,1),
        -- 关联的流程实例 ID
        instance_id VARCHAR(64)     NOT NULL,
        -- 上下文键名
        key         VARCHAR(255)    NOT NULL,
        -- 上下文值 JSON 字符串
        value       NVARCHAR(MAX)   NOT NULL,
        -- 创建时间
        created_at  DATETIME2       NOT NULL DEFAULT GETDATE(),
        -- 更新时间
        updated_at  DATETIME2       NOT NULL DEFAULT GETDATE(),

        CONSTRAINT pk_workflow_instance_context PRIMARY KEY (id),
        CONSTRAINT uq_workflow_instance_context UNIQUE (instance_id, [key]),
        CONSTRAINT fk_workflow_instance_context_instance_id FOREIGN KEY (instance_id)
            REFERENCES workflow_instances(id)
    );
END;
GO

-- ─────────────────────────────────────────────────────────────────────────────
-- 索引创建
-- ─────────────────────────────────────────────────────────────────────────────

-- workflow_instances: 按工作流 ID 查询
IF NOT EXISTS (SELECT * FROM sys.indexes WHERE name = 'idx_workflow_instances_workflow_id')
    CREATE NONCLUSTERED INDEX idx_workflow_instances_workflow_id
        ON workflow_instances(workflow_id);

-- workflow_instances: 按状态查询（引擎重启恢复）
IF NOT EXISTS (SELECT * FROM sys.indexes WHERE name = 'idx_workflow_instances_status')
    CREATE NONCLUSTERED INDEX idx_workflow_instances_status
        ON workflow_instances(status);

-- workflow_node_states: 按实例 ID 查询
IF NOT EXISTS (SELECT * FROM sys.indexes WHERE name = 'idx_workflow_node_states_instance_id')
    CREATE NONCLUSTERED INDEX idx_workflow_node_states_instance_id
        ON workflow_node_states(instance_id);

-- workflow_node_states: 按状态查询
IF NOT EXISTS (SELECT * FROM sys.indexes WHERE name = 'idx_workflow_node_states_status')
    CREATE NONCLUSTERED INDEX idx_workflow_node_states_status
        ON workflow_node_states(status);

-- workflow_node_states: 按 Worker ID 查询（Worker 下线任务转移）
IF NOT EXISTS (SELECT * FROM sys.indexes WHERE name = 'idx_workflow_node_states_assigned_to')
    CREATE NONCLUSTERED INDEX idx_workflow_node_states_assigned_to
        ON workflow_node_states(assigned_to);

-- workflow_node_states: 按 Worker IP 查询
IF NOT EXISTS (SELECT * FROM sys.indexes WHERE name = 'idx_workflow_node_states_worker_ip')
    CREATE NONCLUSTERED INDEX idx_workflow_node_states_worker_ip
        ON workflow_node_states(worker_ip);

-- workflow_instance_context: 按实例 ID 查询
IF NOT EXISTS (SELECT * FROM sys.indexes WHERE name = 'idx_workflow_instance_context_instance_id')
    CREATE NONCLUSTERED INDEX idx_workflow_instance_context_instance_id
        ON workflow_instance_context(instance_id);
GO
