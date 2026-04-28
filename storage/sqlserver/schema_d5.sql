-- D5 阶段第一批数据库初始化脚本（SQL Server）
-- 当前仅落地流程模板市场所需表结构，其他 D5 表将在后续批次补齐

IF NOT EXISTS (SELECT 1 FROM sys.objects WHERE object_id = OBJECT_ID(N'workflow_template_categories') AND type = N'U')
BEGIN
    CREATE TABLE workflow_template_categories (
        category_id BIGINT IDENTITY(1,1) PRIMARY KEY,
        category_name VARCHAR(64) NOT NULL UNIQUE,
        parent_id BIGINT NOT NULL DEFAULT 0,
        sort INT NOT NULL DEFAULT 0,
        description NVARCHAR(255),
        created_at DATETIME2 NOT NULL DEFAULT GETDATE(),
        deleted BIT NOT NULL DEFAULT 0
    );
END
GO

IF NOT EXISTS (SELECT 1 FROM sys.objects WHERE object_id = OBJECT_ID(N'workflow_templates') AND type = N'U')
BEGIN
    CREATE TABLE workflow_templates (
        template_id VARCHAR(64) PRIMARY KEY,
        template_name VARCHAR(255) NOT NULL,
        category_id BIGINT NOT NULL,
        description NVARCHAR(MAX),
        workflow_def NVARCHAR(MAX) NOT NULL,
        version VARCHAR(32) NOT NULL DEFAULT '1.0.0',
        author VARCHAR(64) NOT NULL,
        tags VARCHAR(255),
        icon VARCHAR(255),
        status INT NOT NULL DEFAULT 0,
        visible_scope INT NOT NULL DEFAULT 1,
        visible_range NVARCHAR(MAX),
        install_count INT NOT NULL DEFAULT 0,
        start_count INT NOT NULL DEFAULT 0,
        created_at DATETIME2 NOT NULL DEFAULT GETDATE(),
        updated_at DATETIME2 NOT NULL DEFAULT GETDATE(),
        deleted BIT NOT NULL DEFAULT 0
    );
END
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = 'idx_template_category')
BEGIN
    CREATE INDEX idx_template_category ON workflow_templates(category_id);
END
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = 'idx_template_status')
BEGIN
    CREATE INDEX idx_template_status ON workflow_templates(status);
END
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = 'idx_template_author')
BEGIN
    CREATE INDEX idx_template_author ON workflow_templates(author);
END
GO

IF NOT EXISTS (SELECT 1 FROM workflow_template_categories WHERE category_name = '行政审批' AND deleted = 0)
BEGIN
    INSERT INTO workflow_template_categories (category_name, parent_id, sort, description)
    VALUES ('行政审批', 0, 10, N'请假、报销、出差等行政审批类模板');
END
GO

IF NOT EXISTS (SELECT 1 FROM workflow_template_categories WHERE category_name = '运维自动化' AND deleted = 0)
BEGIN
    INSERT INTO workflow_template_categories (category_name, parent_id, sort, description)
    VALUES ('运维自动化', 0, 20, N'发布、巡检、备份、故障处理等运维类模板');
END
GO

IF NOT EXISTS (SELECT 1 FROM workflow_template_categories WHERE category_name = '研发协同' AND deleted = 0)
BEGIN
    INSERT INTO workflow_template_categories (category_name, parent_id, sort, description)
    VALUES ('研发协同', 0, 30, N'需求评审、代码发布、缺陷流转等研发协同模板');
END
GO
