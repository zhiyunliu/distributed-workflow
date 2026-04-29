-- D4 阶段系统管理数据库初始化脚本（SQL Server）
-- 新增系统用户、角色、菜单、权限关联、数据字典表
-- 不修改 D1-D3 已有业务表，仅新增索引优化查询性能

-- ─── sys_user 系统用户表 ──────────────────────────────────────────────────────

IF NOT EXISTS (SELECT 1 FROM sys.objects WHERE object_id = OBJECT_ID(N'sys_user') AND type = N'U')
BEGIN
    CREATE TABLE sys_user (
        id          BIGINT        IDENTITY(1,1) PRIMARY KEY,
        username    VARCHAR(64)   NOT NULL,
        password    VARCHAR(128)  NOT NULL,     -- bcrypt hash
        real_name   NVARCHAR(64)  NOT NULL,
        email       VARCHAR(128)  NULL,
        phone       VARCHAR(32)   NULL,
        avatar      VARCHAR(255)  NULL,
        status      BIT           NOT NULL DEFAULT 1,  -- 1=启用 0=禁用
        dept_id     BIGINT        NULL,
        create_time DATETIME2     NOT NULL DEFAULT GETDATE(),
        update_time DATETIME2     NOT NULL DEFAULT GETDATE(),
        create_by   VARCHAR(64)   NULL,
        update_by   VARCHAR(64)   NULL,
        deleted     BIT           NOT NULL DEFAULT 0
    );

    CREATE UNIQUE INDEX uidx_sys_user_username ON sys_user(username) WHERE deleted = 0;
END
GO

-- ─── sys_role 系统角色表 ──────────────────────────────────────────────────────

IF NOT EXISTS (SELECT 1 FROM sys.objects WHERE object_id = OBJECT_ID(N'sys_role') AND type = N'U')
BEGIN
    CREATE TABLE sys_role (
        id          BIGINT        IDENTITY(1,1) PRIMARY KEY,
        role_name   VARCHAR(64)   NOT NULL,
        role_code   VARCHAR(64)   NOT NULL,
        description NVARCHAR(255) NULL,
        status      BIT           NOT NULL DEFAULT 1,
        data_scope  INT           NOT NULL DEFAULT 1,  -- 1=全量 2=本部门 3=个人
        create_time DATETIME2     NOT NULL DEFAULT GETDATE(),
        update_time DATETIME2     NOT NULL DEFAULT GETDATE(),
        create_by   VARCHAR(64)   NULL,
        update_by   VARCHAR(64)   NULL,
        deleted     BIT           NOT NULL DEFAULT 0
    );

    CREATE UNIQUE INDEX uidx_sys_role_code ON sys_role(role_code) WHERE deleted = 0;
END
GO

-- ─── sys_menu 系统菜单表 ──────────────────────────────────────────────────────

IF NOT EXISTS (SELECT 1 FROM sys.objects WHERE object_id = OBJECT_ID(N'sys_menu') AND type = N'U')
BEGIN
    CREATE TABLE sys_menu (
        id          BIGINT        IDENTITY(1,1) PRIMARY KEY,
        parent_id   BIGINT        NOT NULL DEFAULT 0,
        menu_name   VARCHAR(64)   NOT NULL,
        menu_type   VARCHAR(10)   NOT NULL,   -- M=目录 C=菜单 F=按钮
        path        VARCHAR(255)  NULL,
        component   VARCHAR(255)  NULL,
        perms       VARCHAR(128)  NULL,       -- 权限标识
        icon        VARCHAR(128)  NULL,
        sort        INT           NOT NULL DEFAULT 0,
        visible     BIT           NOT NULL DEFAULT 1, -- 1=显示 0=隐藏
        is_frame    BIT           NOT NULL DEFAULT 0, -- 1=外链 0=内链
        create_time DATETIME2     NOT NULL DEFAULT GETDATE(),
        update_time DATETIME2     NOT NULL DEFAULT GETDATE(),
        deleted     BIT           NOT NULL DEFAULT 0
    );

    CREATE INDEX idx_sys_menu_parent ON sys_menu(parent_id) WHERE deleted = 0;
END
GO

-- ─── sys_user_role 用户角色关联表 ─────────────────────────────────────────────

IF NOT EXISTS (SELECT 1 FROM sys.objects WHERE object_id = OBJECT_ID(N'sys_user_role') AND type = N'U')
BEGIN
    CREATE TABLE sys_user_role (
        id      BIGINT IDENTITY(1,1) PRIMARY KEY,
        user_id BIGINT NOT NULL,
        role_id BIGINT NOT NULL
    );

    CREATE UNIQUE INDEX uidx_user_role ON sys_user_role(user_id, role_id);
    CREATE INDEX idx_user_role_user ON sys_user_role(user_id);
    CREATE INDEX idx_user_role_role ON sys_user_role(role_id);
END
GO

-- ─── sys_role_menu 角色菜单关联表 ────────────────────────────────────────────

IF NOT EXISTS (SELECT 1 FROM sys.objects WHERE object_id = OBJECT_ID(N'sys_role_menu') AND type = N'U')
BEGIN
    CREATE TABLE sys_role_menu (
        id      BIGINT IDENTITY(1,1) PRIMARY KEY,
        role_id BIGINT NOT NULL,
        menu_id BIGINT NOT NULL
    );

    CREATE UNIQUE INDEX uidx_role_menu ON sys_role_menu(role_id, menu_id);
    CREATE INDEX idx_role_menu_role ON sys_role_menu(role_id);
END
GO

-- ─── sys_dictionary_info 数据字典表 ──────────────────────────────────────────

IF NOT EXISTS (SELECT 1 FROM sys.objects WHERE object_id = OBJECT_ID(N'sys_dictionary_info') AND type = N'U')
BEGIN
    CREATE TABLE sys_dictionary_info (
        dic_id      BIGINT        IDENTITY(1,1) PRIMARY KEY,
        dict_type   VARCHAR(128)  NOT NULL,  -- 字典类型；distributedworkflow 表示类型定义行
        dict_name   NVARCHAR(128) NOT NULL,  -- 显示名称
        dict_value  VARCHAR(255)  NOT NULL,  -- 字典值
        dict_group  VARCHAR(128)  NOT NULL DEFAULT '*', -- 分组，默认 *
        sort        INT           NOT NULL DEFAULT 0,
        status      BIT           NOT NULL DEFAULT 1,
        remark      NVARCHAR(255) NULL,
        create_time DATETIME2     NOT NULL DEFAULT GETDATE(),
        update_time DATETIME2     NOT NULL DEFAULT GETDATE()
    );

    CREATE INDEX idx_dict_info_type  ON sys_dictionary_info(dict_type);
    CREATE INDEX idx_dict_info_group ON sys_dictionary_info(dict_group);
    CREATE INDEX idx_dict_info_type_value ON sys_dictionary_info(dict_type, dict_value);
END
GO

-- ─── 已有表索引优化 ──────────────────────────────────────────────────────────

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = 'idx_workflow_instances_create_time')
BEGIN
    CREATE INDEX idx_workflow_instances_create_time
        ON workflow_instances(create_time DESC);
END
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = 'idx_approval_records_approver')
BEGIN
    CREATE INDEX idx_approval_records_approver
        ON workflow_approval_records(approver, operate_time DESC);
END
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = 'idx_audit_operate_time_desc')
BEGIN
    CREATE INDEX idx_audit_operate_time_desc
        ON workflow_instance_logs(operate_time DESC);
END
GO

-- ─── 初始化默认数据 ─────────────────────────────────────────────────────────

-- 超级管理员角色（若不存在）
IF NOT EXISTS (SELECT 1 FROM sys_role WHERE role_code = 'super_admin' AND deleted = 0)
BEGIN
    INSERT INTO sys_role (role_name, role_code, description, status, data_scope, create_by)
    VALUES (N'超级管理员', 'super_admin', N'拥有所有权限', 1, 1, 'system');
END
GO

-- 默认管理员账号（密码: Admin@123，bcrypt hash 在应用启动时按需生成）
-- 此处使用占位密码，应用启动时若用户不存在则自动初始化
IF NOT EXISTS (SELECT 1 FROM sys_user WHERE username = 'admin' AND deleted = 0)
BEGIN
    -- 密码 Admin@123 的 bcrypt hash（cost=10）
    INSERT INTO sys_user (username, password, real_name, status, create_by)
    VALUES ('admin', '$2a$10$placeholder_will_be_set_at_startup', N'系统管理员', 1, 'system');
END
GO
