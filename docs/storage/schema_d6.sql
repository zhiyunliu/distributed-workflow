-- D6 阶段数据库初始化脚本（SQL Server）
-- 版本: v1.0.0
-- 包含: 表单设计器、高级审批、数据分析汇总、OAuth绑定、插件市场、原有表索引优化

-- ============================================================
-- 1. 表单设计器相关表
-- ============================================================

-- 表单定义表
IF OBJECT_ID('form_definition', 'U') IS NULL
BEGIN
    CREATE TABLE form_definition (
        form_id     VARCHAR(64)    NOT NULL,
        form_name   NVARCHAR(255)  NOT NULL,                          -- 表单名称
        description NVARCHAR(MAX),                                    -- 表单描述
        form_schema NVARCHAR(MAX)  NOT NULL,                          -- 表单JSON Schema定义
        version     INT            NOT NULL DEFAULT 1,                -- 当前版本号，从1开始自增
        status      INT            NOT NULL DEFAULT 0,                -- 0=草稿 1=已发布 2=已停用
        created_by  VARCHAR(64)    NOT NULL,                          -- 创建人
        created_at  DATETIME2      NOT NULL DEFAULT GETDATE(),
        updated_at  DATETIME2      NOT NULL DEFAULT GETDATE(),
        deleted     BIT            NOT NULL DEFAULT 0,                -- 软删除标记
        CONSTRAINT PK_form_definition PRIMARY KEY (form_id)
    );
END
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = 'idx_form_def_status')
BEGIN
    CREATE INDEX idx_form_def_status ON form_definition(status, deleted);
END
GO

-- 表单实例表
IF OBJECT_ID('form_instance', 'U') IS NULL
BEGIN
    CREATE TABLE form_instance (
        instance_id          VARCHAR(64)   NOT NULL,
        form_id              VARCHAR(64)   NOT NULL,                  -- 关联的表单定义ID
        form_version         INT           NOT NULL,                  -- 绑定的表单版本号
        workflow_instance_id VARCHAR(64)   NOT NULL,                  -- 关联的流程实例ID
        node_id              VARCHAR(64),                             -- 关联的流程节点ID（启动表单为空）
        form_data            NVARCHAR(MAX) NOT NULL,                  -- 表单提交数据JSON
        status               INT           NOT NULL DEFAULT 0,        -- 0=草稿 1=已提交 2=已修改
        created_by           VARCHAR(64)   NOT NULL,
        created_at           DATETIME2     NOT NULL DEFAULT GETDATE(),
        updated_at           DATETIME2     NOT NULL DEFAULT GETDATE(),
        CONSTRAINT PK_form_instance PRIMARY KEY (instance_id)
    );
END
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = 'idx_form_instance_workflow')
BEGIN
    CREATE INDEX idx_form_instance_workflow ON form_instance(workflow_instance_id, node_id);
END
GO

-- 表单版本历史表
IF OBJECT_ID('form_version_history', 'U') IS NULL
BEGIN
    CREATE TABLE form_version_history (
        id         BIGINT        IDENTITY(1,1) NOT NULL,
        form_id    VARCHAR(64)   NOT NULL,                            -- 关联的表单定义ID
        version    INT           NOT NULL,                            -- 版本号
        form_schema NVARCHAR(MAX) NOT NULL,                           -- 该版本的表单Schema
        change_log NVARCHAR(MAX),                                     -- 版本变更说明
        created_by VARCHAR(64)   NOT NULL,
        created_at DATETIME2     NOT NULL DEFAULT GETDATE(),
        CONSTRAINT PK_form_version_history PRIMARY KEY (id),
        CONSTRAINT UQ_form_version UNIQUE (form_id, version)
    );
END
GO

-- ============================================================
-- 2. 高级审批相关表
-- ============================================================

-- 审批委托表
IF OBJECT_ID('approval_delegate', 'U') IS NULL
BEGIN
    CREATE TABLE approval_delegate (
        id           BIGINT     IDENTITY(1,1) NOT NULL,
        delegator_id VARCHAR(64) NOT NULL,                            -- 委托人（原审批人）
        agent_id     VARCHAR(64) NOT NULL,                            -- 代理人
        start_time   DATETIME2  NOT NULL,                             -- 委托开始时间
        end_time     DATETIME2  NOT NULL,                             -- 委托结束时间
        status       INT        NOT NULL DEFAULT 1,                   -- 1=有效 2=已失效
        created_by   VARCHAR(64) NOT NULL,
        created_at   DATETIME2  NOT NULL DEFAULT GETDATE(),
        updated_at   DATETIME2  NOT NULL DEFAULT GETDATE(),
        CONSTRAINT PK_approval_delegate PRIMARY KEY (id)
    );
END
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = 'idx_approval_delegate_delegator')
BEGIN
    CREATE INDEX idx_approval_delegate_delegator ON approval_delegate(delegator_id, status);
END
GO

-- 审批抄送表
IF OBJECT_ID('approval_cc', 'U') IS NULL
BEGIN
    CREATE TABLE approval_cc (
        id                   BIGINT     IDENTITY(1,1) NOT NULL,
        workflow_instance_id VARCHAR(64) NOT NULL,                    -- 流程实例ID
        node_id              VARCHAR(64) NOT NULL,                    -- 抄送节点ID
        cc_user_id           VARCHAR(64) NOT NULL,                    -- 被抄送用户ID
        cc_time              DATETIME2  NOT NULL DEFAULT GETDATE(),   -- 抄送时间
        is_read              BIT        NOT NULL DEFAULT 0,           -- 是否已读
        read_time            DATETIME2,                               -- 已读时间
        CONSTRAINT PK_approval_cc PRIMARY KEY (id)
    );
END
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = 'idx_approval_cc_user')
BEGIN
    CREATE INDEX idx_approval_cc_user ON approval_cc(cc_user_id, is_read);
END
GO

-- 审批加签记录表
IF OBJECT_ID('approval_add_sign', 'U') IS NULL
BEGIN
    CREATE TABLE approval_add_sign (
        id                   BIGINT        IDENTITY(1,1) NOT NULL,
        workflow_instance_id VARCHAR(64)   NOT NULL,                  -- 流程实例ID
        node_id              VARCHAR(64)   NOT NULL,                  -- 节点ID
        operator_id          VARCHAR(64)   NOT NULL,                  -- 加签操作人
        sign_type            VARCHAR(32)   NOT NULL,                  -- before=前加签 after=后加签
        sign_users           NVARCHAR(MAX) NOT NULL,                  -- 加签人员列表JSON
        created_at           DATETIME2     NOT NULL DEFAULT GETDATE(),
        CONSTRAINT PK_approval_add_sign PRIMARY KEY (id)
    );
END
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = 'idx_add_sign_instance')
BEGIN
    CREATE INDEX idx_add_sign_instance ON approval_add_sign(workflow_instance_id, node_id);
END
GO

-- ============================================================
-- 3. 数据分析汇总表
-- ============================================================

-- 流程指标日汇总表
IF OBJECT_ID('stats_workflow_daily', 'U') IS NULL
BEGIN
    CREATE TABLE stats_workflow_daily (
        id                    BIGINT      IDENTITY(1,1) NOT NULL,
        workflow_id           VARCHAR(64) NOT NULL,                   -- 流程定义ID
        stat_date             DATE        NOT NULL,                   -- 统计日期
        start_count           INT         NOT NULL DEFAULT 0,         -- 当日启动实例数
        complete_count        INT         NOT NULL DEFAULT 0,         -- 当日完成实例数
        fail_count            INT         NOT NULL DEFAULT 0,         -- 当日失败实例数
        avg_execution_time_ms BIGINT      NOT NULL DEFAULT 0,         -- 平均执行时长（毫秒）
        max_execution_time_ms BIGINT      NOT NULL DEFAULT 0,         -- 最长执行时长（毫秒）
        timeout_count         INT         NOT NULL DEFAULT 0,         -- 超时实例数
        reject_count          INT         NOT NULL DEFAULT 0,         -- 被驳回实例数
        CONSTRAINT PK_stats_workflow_daily PRIMARY KEY (id),
        CONSTRAINT UQ_stats_workflow_date UNIQUE (workflow_id, stat_date)
    );
END
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = 'idx_stats_workflow_date')
BEGIN
    CREATE INDEX idx_stats_workflow_date ON stats_workflow_daily(stat_date);
END
GO

-- 审批指标日汇总表
IF OBJECT_ID('stats_approval_daily', 'U') IS NULL
BEGIN
    CREATE TABLE stats_approval_daily (
        id                   BIGINT      IDENTITY(1,1) NOT NULL,
        user_id              VARCHAR(64) NOT NULL,                    -- 审批人用户ID
        stat_date            DATE        NOT NULL,                    -- 统计日期
        approve_count        INT         NOT NULL DEFAULT 0,          -- 当日通过审批数
        reject_count         INT         NOT NULL DEFAULT 0,          -- 当日驳回审批数
        avg_approval_time_ms BIGINT      NOT NULL DEFAULT 0,          -- 平均审批时长（毫秒）
        timeout_count        INT         NOT NULL DEFAULT 0,          -- 超时未处理数
        CONSTRAINT PK_stats_approval_daily PRIMARY KEY (id),
        CONSTRAINT UQ_stats_approval_date UNIQUE (user_id, stat_date)
    );
END
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = 'idx_stats_approval_date')
BEGIN
    CREATE INDEX idx_stats_approval_date ON stats_approval_daily(stat_date);
END
GO

-- ============================================================
-- 4. OAuth绑定表
-- ============================================================

-- 第三方平台OAuth绑定表
IF OBJECT_ID('oauth_bindings', 'U') IS NULL
BEGIN
    CREATE TABLE oauth_bindings (
        id               BIGINT         IDENTITY(1,1) NOT NULL,
        sys_user_id      VARCHAR(64)    NOT NULL,                     -- 系统用户ID
        platform         VARCHAR(32)    NOT NULL,                     -- 平台：dingtalk/wechatwork
        open_id          VARCHAR(128)   NOT NULL,                     -- 平台唯一标识
        union_id         VARCHAR(128),                                -- 平台联合标识
        nickname         NVARCHAR(128),                               -- 平台昵称
        avatar_url       NVARCHAR(512),                               -- 头像URL
        access_token     NVARCHAR(MAX),                               -- 加密存储的AccessToken
        refresh_token    NVARCHAR(MAX),                               -- 加密存储的RefreshToken
        token_expire_at  DATETIME2,                                   -- Token过期时间
        created_at       DATETIME2      NOT NULL DEFAULT GETDATE(),
        updated_at       DATETIME2      NOT NULL DEFAULT GETDATE(),
        CONSTRAINT PK_oauth_bindings PRIMARY KEY (id),
        CONSTRAINT UQ_oauth_platform_openid UNIQUE (platform, open_id)
    );
END
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = 'idx_oauth_sys_user')
BEGIN
    CREATE INDEX idx_oauth_sys_user ON oauth_bindings(sys_user_id, platform);
END
GO

-- ============================================================
-- 5. 插件相关表
-- ============================================================

-- 插件信息表（插件市场）
IF OBJECT_ID('plugin_info', 'U') IS NULL
BEGIN
    CREATE TABLE plugin_info (
        plugin_id      VARCHAR(64)    NOT NULL,
        plugin_name    VARCHAR(128)   NOT NULL,                       -- 插件名称
        description    NVARCHAR(MAX),                                 -- 插件描述
        author         VARCHAR(64)    NOT NULL,                       -- 作者
        version        VARCHAR(32)    NOT NULL,                       -- 当前版本
        plugin_type    VARCHAR(64)    NOT NULL,                       -- 插件类型：node_executor/connector/notification/report
        plugin_package NVARCHAR(MAX)  NOT NULL,                       -- 插件包信息（安装路径或下载地址）
        config_schema  NVARCHAR(MAX),                                 -- 插件配置Schema（JSON）
        status         INT            NOT NULL DEFAULT 0,             -- 0=未安装 1=已安装 2=已启用 3=已停用
        is_official    BIT            NOT NULL DEFAULT 0,             -- 是否官方插件
        download_count INT            NOT NULL DEFAULT 0,             -- 下载次数
        rating         DECIMAL(2,1)   NOT NULL DEFAULT 0.0,           -- 评分（0.0-5.0）
        created_at     DATETIME2      NOT NULL DEFAULT GETDATE(),
        updated_at     DATETIME2      NOT NULL DEFAULT GETDATE(),
        CONSTRAINT PK_plugin_info PRIMARY KEY (plugin_id),
        CONSTRAINT UQ_plugin_name UNIQUE (plugin_name)
    );
END
GO

-- 插件安装配置表
IF OBJECT_ID('plugin_config', 'U') IS NULL
BEGIN
    CREATE TABLE plugin_config (
        id             BIGINT        IDENTITY(1,1) NOT NULL,
        plugin_id      VARCHAR(64)   NOT NULL,                        -- 关联插件ID
        plugin_version VARCHAR(32)   NOT NULL,                        -- 安装的版本
        config         NVARCHAR(MAX),                                 -- 插件运行配置（JSON）
        status         INT           NOT NULL DEFAULT 1,              -- 1=已启用 2=已停用
        installed_at   DATETIME2     NOT NULL DEFAULT GETDATE(),
        updated_at     DATETIME2     NOT NULL DEFAULT GETDATE(),
        CONSTRAINT PK_plugin_config PRIMARY KEY (id),
        CONSTRAINT UQ_plugin_config_id UNIQUE (plugin_id)
    );
END
GO

-- ============================================================
-- 6. 原有表索引优化（追加，不修改原有结构）
-- ============================================================

-- 优化流程实例查询性能
IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = 'idx_workflow_instances_workflow_id_status')
BEGIN
    CREATE INDEX idx_workflow_instances_workflow_id_status
        ON workflow_instances(workflow_id, status, create_time DESC);
END
GO

-- 优化节点状态查询性能
IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = 'idx_workflow_node_states_instance_id_status')
BEGIN
    CREATE INDEX idx_workflow_node_states_instance_id_status
        ON workflow_node_states(instance_id, status);
END
GO

-- 优化审计日志查询性能
IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = 'idx_audit_workflow_id_operation_type')
BEGIN
    CREATE INDEX idx_audit_workflow_id_operation_type
        ON workflow_instance_logs(workflow_id, operation_type, operate_time DESC);
END
GO

-- 优化审批记录查询性能
IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = 'idx_approval_instance_node_status')
BEGIN
    CREATE INDEX idx_approval_instance_node_status
        ON workflow_approval_records(instance_id, node_id, operate_time DESC);
END
GO
