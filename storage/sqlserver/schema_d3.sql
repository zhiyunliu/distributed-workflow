-- D3 数据库 Schema 扩展脚本
-- 执行顺序：本脚本依赖 schema_d1.sql 和 schema_d2.sql 已执行

-- ─────────────────────────────────────────────────────────────────────────────
-- 1. 审计日志表
-- ─────────────────────────────────────────────────────────────────────────────
IF NOT EXISTS (SELECT * FROM sys.tables WHERE name = 'workflow_instance_logs')
BEGIN
    CREATE TABLE workflow_instance_logs (
        id              NVARCHAR(64)    NOT NULL PRIMARY KEY,      -- 日志唯一ID
        instance_id     NVARCHAR(64)    NOT NULL DEFAULT '',       -- 流程实例ID
        node_id         NVARCHAR(64)    NOT NULL DEFAULT '',       -- 节点ID
        workflow_id     NVARCHAR(64)    NOT NULL DEFAULT '',       -- 流程定义ID
        endpoint_id     NVARCHAR(64)    NOT NULL DEFAULT '',       -- 端点ID
        operation_type  NVARCHAR(64)    NOT NULL,                  -- 操作类型
        operator        NVARCHAR(128)   NOT NULL DEFAULT '',       -- 操作人
        operate_ip      NVARCHAR(64)    NOT NULL DEFAULT '',       -- 操作IP
        operate_time    DATETIME2       NOT NULL DEFAULT GETUTCDATE(), -- 操作时间
        before_data     NVARCHAR(MAX)   NULL,                      -- 变更前数据(JSON)
        after_data      NVARCHAR(MAX)   NULL,                      -- 变更后数据(JSON)
        detail          NVARCHAR(MAX)   NOT NULL DEFAULT '',       -- 详情描述
        trace_id        NVARCHAR(128)   NOT NULL DEFAULT ''        -- 链路追踪ID
    );

    CREATE INDEX idx_wil_instance_id    ON workflow_instance_logs(instance_id);
    CREATE INDEX idx_wil_workflow_id    ON workflow_instance_logs(workflow_id);
    CREATE INDEX idx_wil_endpoint_id    ON workflow_instance_logs(endpoint_id);
    CREATE INDEX idx_wil_operator       ON workflow_instance_logs(operator);
    CREATE INDEX idx_wil_operate_time   ON workflow_instance_logs(operate_time);
    CREATE INDEX idx_wil_operation_type ON workflow_instance_logs(operation_type);
END
GO

-- ─────────────────────────────────────────────────────────────────────────────
-- 2. 人工审批记录表
-- ─────────────────────────────────────────────────────────────────────────────
IF NOT EXISTS (SELECT * FROM sys.tables WHERE name = 'workflow_approval_records')
BEGIN
    CREATE TABLE workflow_approval_records (
        id           NVARCHAR(64)  NOT NULL PRIMARY KEY,      -- 审批记录ID
        instance_id  NVARCHAR(64)  NOT NULL,                  -- 流程实例ID
        node_id      NVARCHAR(64)  NOT NULL,                  -- 节点ID
        approver     NVARCHAR(128) NOT NULL,                  -- 审批人
        action       NVARCHAR(32)  NOT NULL,                  -- 操作：approve/reject/cancel
        comment      NVARCHAR(MAX) NOT NULL DEFAULT '',       -- 审批意见
        form_data    NVARCHAR(MAX) NULL,                      -- 表单数据(JSON)
        operate_time DATETIME2     NOT NULL DEFAULT GETUTCDATE(), -- 操作时间
        operate_ip   NVARCHAR(64)  NOT NULL DEFAULT ''        -- 操作IP
    );

    CREATE INDEX idx_war_instance_id ON workflow_approval_records(instance_id);
    CREATE INDEX idx_war_node_id     ON workflow_approval_records(node_id);
    CREATE INDEX idx_war_approver    ON workflow_approval_records(approver);
END
GO

-- ─────────────────────────────────────────────────────────────────────────────
-- 3. 端点表（D3新增）
-- ─────────────────────────────────────────────────────────────────────────────
IF NOT EXISTS (SELECT * FROM sys.tables WHERE name = 'workflow_endpoints')
BEGIN
    CREATE TABLE workflow_endpoints (
        id              NVARCHAR(64)    NOT NULL PRIMARY KEY,      -- 端点唯一ID
        name            NVARCHAR(255)   NOT NULL,                  -- 端点名称
        type            NVARCHAR(32)    NOT NULL,                  -- 端点类型：http/schedule
        workflow_id     NVARCHAR(64)    NOT NULL,                  -- 关联流程定义ID
        config          NVARCHAR(MAX)   NULL,                      -- 通用配置(JSON)
        schedule_config NVARCHAR(MAX)   NULL,                      -- 定时配置(JSON)
        path            NVARCHAR(512)   NOT NULL DEFAULT '',       -- HTTP端点路径
        created_by      NVARCHAR(128)   NOT NULL DEFAULT '',       -- 创建人
        disabled        BIT             NOT NULL DEFAULT 0,        -- 是否禁用
        created_at      DATETIME2       NOT NULL DEFAULT GETUTCDATE(), -- 创建时间
        updated_at      DATETIME2       NOT NULL DEFAULT GETUTCDATE()  -- 更新时间
    );

    CREATE INDEX idx_we_workflow_id ON workflow_endpoints(workflow_id);
    CREATE INDEX idx_we_type        ON workflow_endpoints(type);
    CREATE INDEX idx_we_disabled    ON workflow_endpoints(disabled);
END
GO

-- ─────────────────────────────────────────────────────────────────────────────
-- 4. 扩展 workflow_node_states 表（D3新增字段）
-- ─────────────────────────────────────────────────────────────────────────────
IF NOT EXISTS (SELECT * FROM sys.columns WHERE object_id = OBJECT_ID('workflow_node_states') AND name = 'approval_status')
BEGIN
    ALTER TABLE workflow_node_states
        ADD approval_status        NVARCHAR(32) NOT NULL DEFAULT '',
            current_approver_index INT          NOT NULL DEFAULT 0;
END
GO
