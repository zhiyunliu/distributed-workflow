-- DistributedWorkflow D2 数据库升级脚本
-- 版本：v0.5.0
-- 说明：在 D1 基础表结构上进行增量升级，兼容已有数据
-- 执行前提：已执行 D1 的 schema.sql

-- ─────────────────────────────────────────────────────────────────────────────
-- 1. workflow_defs 表新增字段
-- ─────────────────────────────────────────────────────────────────────────────
ALTER TABLE workflow_defs
ADD
    current_version INT NOT NULL DEFAULT 1,
    global_retry_policy NVARCHAR(MAX),
    timeout INT NOT NULL DEFAULT 0;
GO

-- ─────────────────────────────────────────────────────────────────────────────
-- 2. workflow_versions 表（完整版，D1 基础表替换为完整结构）
-- ─────────────────────────────────────────────────────────────────────────────
DROP TABLE IF EXISTS workflow_versions;
GO

CREATE TABLE workflow_versions (
    id          INT IDENTITY(1,1) PRIMARY KEY,
    workflow_id VARCHAR(64)     NOT NULL,
    version     INT             NOT NULL,
    definition  NVARCHAR(MAX)   NOT NULL,
    change_log  NVARCHAR(MAX),
    created_by  VARCHAR(64)     NOT NULL DEFAULT '',
    created_at  DATETIME2       NOT NULL DEFAULT GETDATE(),
    is_current  BIT             NOT NULL DEFAULT 0,
    gray_config NVARCHAR(MAX),
    CONSTRAINT fk_wv_workflow_id FOREIGN KEY (workflow_id) REFERENCES workflow_defs(id),
    CONSTRAINT uq_wv_workflow_version UNIQUE (workflow_id, version)
);
GO

CREATE INDEX idx_workflow_versions_workflow_id ON workflow_versions(workflow_id);
CREATE INDEX idx_workflow_versions_is_current  ON workflow_versions(workflow_id, is_current);
GO

-- ─────────────────────────────────────────────────────────────────────────────
-- 3. workflow_instances 表新增字段
-- ─────────────────────────────────────────────────────────────────────────────
ALTER TABLE workflow_instances
ADD
    pause_time          DATETIME2,
    cancel_time         DATETIME2,
    parent_instance_id  VARCHAR(64),
    parent_node_id      VARCHAR(64),
    is_subflow_instance BIT        NOT NULL DEFAULT 0,
    pause_node_ids      NVARCHAR(MAX),
    operator            VARCHAR(64);
GO

CREATE INDEX idx_workflow_instances_parent_instance_id ON workflow_instances(parent_instance_id);
CREATE INDEX idx_workflow_instances_status_created_at  ON workflow_instances(status, created_at);
GO

-- ─────────────────────────────────────────────────────────────────────────────
-- 4. workflow_node_states 表新增字段
-- ─────────────────────────────────────────────────────────────────────────────
ALTER TABLE workflow_node_states
ADD
    next_retry_time DATETIME2,
    last_retry_time DATETIME2,
    error_code      VARCHAR(64),
    skipped_reason  NVARCHAR(MAX);
GO

CREATE INDEX idx_workflow_node_states_next_retry_time        ON workflow_node_states(next_retry_time);
CREATE INDEX idx_workflow_node_states_assigned_to_status     ON workflow_node_states(assigned_to, status);
GO

-- ─────────────────────────────────────────────────────────────────────────────
-- 5. 新增 workflow_dead_letter_tasks 死信队列表
-- ─────────────────────────────────────────────────────────────────────────────
CREATE TABLE workflow_dead_letter_tasks (
    id              VARCHAR(64)   PRIMARY KEY,
    instance_id     VARCHAR(64)   NOT NULL,
    node_id         VARCHAR(64)   NOT NULL,
    worker_id       VARCHAR(64),
    worker_ip       VARCHAR(64),
    error_message   NVARCHAR(MAX) NOT NULL,
    error_code      VARCHAR(64),
    retry_count     INT           NOT NULL DEFAULT 0,
    task_data       NVARCHAR(MAX) NOT NULL,
    created_at      DATETIME2     NOT NULL DEFAULT GETDATE(),
    resend_count    INT           NOT NULL DEFAULT 0,
    last_resend_at  DATETIME2,
    CONSTRAINT fk_dlt_instance_id FOREIGN KEY (instance_id) REFERENCES workflow_instances(id)
);
GO

CREATE INDEX idx_workflow_dead_letter_instance_id ON workflow_dead_letter_tasks(instance_id);
CREATE INDEX idx_workflow_dead_letter_created_at  ON workflow_dead_letter_tasks(created_at);
GO

-- ─────────────────────────────────────────────────────────────────────────────
-- 6. 新增 workflow_context_snapshots 上下文快照表（用于暂停/恢复断点）
-- ─────────────────────────────────────────────────────────────────────────────
CREATE TABLE workflow_context_snapshots (
    id              VARCHAR(64)   PRIMARY KEY,
    instance_id     VARCHAR(64)   NOT NULL,
    context_data    NVARCHAR(MAX) NOT NULL,
    created_at      DATETIME2     NOT NULL DEFAULT GETDATE(),
    CONSTRAINT fk_wcs_instance_id FOREIGN KEY (instance_id) REFERENCES workflow_instances(id)
);
GO

CREATE INDEX idx_workflow_context_snapshots_instance_id ON workflow_context_snapshots(instance_id);
GO
