package sqlserver

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/zhiyunliu/distributed-workflow/runtime-execution/internal/common/types"
)

// CreateWorkflowTemplate 创建流程模板
func (r *Repository) CreateWorkflowTemplate(template *types.WorkflowTemplate) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
	defer cancel()

	workflowDefJSON, err := marshalTemplateWorkflowDef(template.WorkflowDef)
	if err != nil {
		return err
	}

	const q = `
INSERT INTO workflow_templates
(template_id, template_name, category_id, description, workflow_def, version, author, tags, icon, status, visible_scope, visible_range, install_count, start_count, created_at, updated_at, deleted)
VALUES
(@templateID, @templateName, @categoryID, @description, @workflowDef, @version, @author, @tags, @icon, @status, @visibleScope, @visibleRange, @installCount, @startCount, @createdAt, @updatedAt, 0)`

	_, err = r.db.ExecContext(ctx, q,
		sql.Named("templateID", template.TemplateID),
		sql.Named("templateName", template.TemplateName),
		sql.Named("categoryID", template.CategoryID),
		sql.Named("description", nullStr(template.Description)),
		sql.Named("workflowDef", workflowDefJSON),
		sql.Named("version", template.Version),
		sql.Named("author", template.Author),
		sql.Named("tags", nullStr(strings.Join(template.Tags, ","))),
		sql.Named("icon", nullStr(template.Icon)),
		sql.Named("status", int(template.Status)),
		sql.Named("visibleScope", int(template.VisibleScope)),
		sql.Named("visibleRange", nullStr(marshalStringSlice(template.VisibleRange))),
		sql.Named("installCount", template.InstallCount),
		sql.Named("startCount", template.StartCount),
		sql.Named("createdAt", template.CreatedAt),
		sql.Named("updatedAt", template.UpdatedAt),
	)
	return wrapDBErr(err, "CreateWorkflowTemplate")
}

// UpdateWorkflowTemplate 更新流程模板
func (r *Repository) UpdateWorkflowTemplate(template *types.WorkflowTemplate) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
	defer cancel()

	workflowDefJSON, err := marshalTemplateWorkflowDef(template.WorkflowDef)
	if err != nil {
		return err
	}

	const q = `
UPDATE workflow_templates
SET template_name = @templateName,
    category_id = @categoryID,
    description = @description,
    workflow_def = @workflowDef,
    version = @version,
    tags = @tags,
    icon = @icon,
    status = @status,
    visible_scope = @visibleScope,
    visible_range = @visibleRange,
    updated_at = @updatedAt
WHERE template_id = @templateID AND deleted = 0`

	_, err = r.db.ExecContext(ctx, q,
		sql.Named("templateID", template.TemplateID),
		sql.Named("templateName", template.TemplateName),
		sql.Named("categoryID", template.CategoryID),
		sql.Named("description", nullStr(template.Description)),
		sql.Named("workflowDef", workflowDefJSON),
		sql.Named("version", template.Version),
		sql.Named("tags", nullStr(strings.Join(template.Tags, ","))),
		sql.Named("icon", nullStr(template.Icon)),
		sql.Named("status", int(template.Status)),
		sql.Named("visibleScope", int(template.VisibleScope)),
		sql.Named("visibleRange", nullStr(marshalStringSlice(template.VisibleRange))),
		sql.Named("updatedAt", template.UpdatedAt),
	)
	return wrapDBErr(err, "UpdateWorkflowTemplate")
}

// GetWorkflowTemplate 获取流程模板详情
func (r *Repository) GetWorkflowTemplate(templateID string) (*types.WorkflowTemplate, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
	defer cancel()

	const q = `
SELECT t.template_id, t.template_name, t.category_id, ISNULL(c.category_name, ''), t.description, t.workflow_def,
       t.version, t.author, t.tags, t.icon, t.status, t.visible_scope, t.visible_range,
       t.install_count, t.start_count, t.created_at, t.updated_at
FROM workflow_templates t
LEFT JOIN workflow_template_categories c ON c.category_id = t.category_id AND c.deleted = 0
WHERE t.template_id = @templateID AND t.deleted = 0`

	row := r.db.QueryRowContext(ctx, q, sql.Named("templateID", templateID))
	return scanWorkflowTemplate(row)
}

// ListWorkflowTemplates 查询流程模板列表
func (r *Repository) ListWorkflowTemplates(filter types.WorkflowTemplateFilter, page, pageSize int) ([]*types.WorkflowTemplate, int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
	defer cancel()

	where, args := buildTemplateWhere(filter)

	countQ := `
SELECT COUNT(*)
FROM workflow_templates t
LEFT JOIN workflow_template_categories c ON c.category_id = t.category_id AND c.deleted = 0` + where

	var total int64
	if err := r.db.QueryRowContext(ctx, countQ, args...).Scan(&total); err != nil {
		return nil, 0, wrapDBErr(err, "ListWorkflowTemplates.count")
	}

	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	dataQ := `
SELECT t.template_id, t.template_name, t.category_id, ISNULL(c.category_name, ''), t.description, t.workflow_def,
       t.version, t.author, t.tags, t.icon, t.status, t.visible_scope, t.visible_range,
       t.install_count, t.start_count, t.created_at, t.updated_at
FROM workflow_templates t
LEFT JOIN workflow_template_categories c ON c.category_id = t.category_id AND c.deleted = 0` + where + `
ORDER BY t.updated_at DESC
OFFSET @offset ROWS FETCH NEXT @pageSize ROWS ONLY`

	args = append(args, sql.Named("offset", offset), sql.Named("pageSize", pageSize))
	rows, err := r.db.QueryContext(ctx, dataQ, args...)
	if err != nil {
		return nil, 0, wrapDBErr(err, "ListWorkflowTemplates.query")
	}
	defer rows.Close()

	templates := make([]*types.WorkflowTemplate, 0)
	for rows.Next() {
		item, scanErr := scanWorkflowTemplateRow(rows)
		if scanErr != nil {
			return nil, 0, scanErr
		}
		templates = append(templates, item)
	}
	return templates, total, rows.Err()
}

// PublishWorkflowTemplate 发布流程模板
func (r *Repository) PublishWorkflowTemplate(templateID string, version string, updatedAt time.Time) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
	defer cancel()

	const q = `
UPDATE workflow_templates
SET status = @status, version = @version, updated_at = @updatedAt
WHERE template_id = @templateID AND deleted = 0`

	_, err := r.db.ExecContext(ctx, q,
		sql.Named("status", int(types.WorkflowTemplateStatusPublished)),
		sql.Named("version", version),
		sql.Named("updatedAt", updatedAt),
		sql.Named("templateID", templateID),
	)
	return wrapDBErr(err, "PublishWorkflowTemplate")
}

// IncrementWorkflowTemplateInstallCount 增加模板安装次数
func (r *Repository) IncrementWorkflowTemplateInstallCount(templateID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
	defer cancel()

	const q = `
UPDATE workflow_templates
SET install_count = install_count + 1, updated_at = @updatedAt
WHERE template_id = @templateID AND deleted = 0`

	_, err := r.db.ExecContext(ctx, q,
		sql.Named("updatedAt", time.Now().UTC()),
		sql.Named("templateID", templateID),
	)
	return wrapDBErr(err, "IncrementWorkflowTemplateInstallCount")
}

// ListWorkflowTemplateCategories 查询模板分类
func (r *Repository) ListWorkflowTemplateCategories() ([]*types.WorkflowTemplateCategory, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
	defer cancel()

	const q = `
SELECT category_id, category_name, parent_id, sort, description, created_at
FROM workflow_template_categories
WHERE deleted = 0
ORDER BY sort ASC, category_id ASC`

	rows, err := r.db.QueryContext(ctx, q)
	if err != nil {
		return nil, wrapDBErr(err, "ListWorkflowTemplateCategories")
	}
	defer rows.Close()

	categories := make([]*types.WorkflowTemplateCategory, 0)
	for rows.Next() {
		var item types.WorkflowTemplateCategory
		var description sql.NullString
		if err = rows.Scan(
			&item.CategoryID,
			&item.CategoryName,
			&item.ParentID,
			&item.Sort,
			&description,
			&item.CreatedAt,
		); err != nil {
			return nil, wrapDBErr(err, "ListWorkflowTemplateCategories.scan")
		}
		if description.Valid {
			item.Description = description.String
		}
		categories = append(categories, &item)
	}
	return categories, rows.Err()
}

func buildTemplateWhere(filter types.WorkflowTemplateFilter) (string, []interface{}) {
	where := " WHERE t.deleted = 0"
	args := make([]interface{}, 0)

	if filter.Keyword != "" {
		where += " AND (t.template_name LIKE @keyword OR t.description LIKE @keyword)"
		args = append(args, sql.Named("keyword", "%"+filter.Keyword+"%"))
	}
	if filter.CategoryID > 0 {
		where += " AND t.category_id = @categoryID"
		args = append(args, sql.Named("categoryID", filter.CategoryID))
	}
	if filter.Status != nil {
		where += " AND t.status = @status"
		args = append(args, sql.Named("status", int(*filter.Status)))
	}
	if filter.Author != "" {
		where += " AND t.author = @author"
		args = append(args, sql.Named("author", filter.Author))
	}
	if filter.OnlyPublished {
		where += " AND t.status = @publishedStatus"
		args = append(args, sql.Named("publishedStatus", int(types.WorkflowTemplateStatusPublished)))
	}

	visibleParts := make([]string, 0)
	if filter.VisibleToUser != "" {
		visibleParts = append(visibleParts, "(t.visible_scope = @visibleUserScope AND t.visible_range LIKE @visibleUser)")
		args = append(args,
			sql.Named("visibleUserScope", int(types.WorkflowTemplateVisibleScopeUsers)),
			sql.Named("visibleUser", "%\""+filter.VisibleToUser+"\"%"),
		)
	}
	for idx, deptID := range filter.VisibleDeptIDs {
		if deptID == "" {
			continue
		}
		scopeName := fmt.Sprintf("visibleDeptScope%d", idx)
		deptName := fmt.Sprintf("visibleDept%d", idx)
		visibleParts = append(visibleParts, fmt.Sprintf("(t.visible_scope = @%s AND t.visible_range LIKE @%s)", scopeName, deptName))
		args = append(args,
			sql.Named(scopeName, int(types.WorkflowTemplateVisibleScopeDept)),
			sql.Named(deptName, "%\""+deptID+"\"%"),
		)
	}
	if len(visibleParts) > 0 {
		where += " AND (t.visible_scope = @companyScope OR " + strings.Join(visibleParts, " OR ") + ")"
		args = append(args, sql.Named("companyScope", int(types.WorkflowTemplateVisibleScopeCompany)))
	}

	return where, args
}

func marshalTemplateWorkflowDef(def *types.WorkflowDef) (string, error) {
	if def == nil {
		return "", fmt.Errorf("workflow template definition is required")
	}
	data, err := json.Marshal(def)
	if err != nil {
		return "", fmt.Errorf("marshal workflow template definition: %w", err)
	}
	return string(data), nil
}

func marshalStringSlice(items []string) string {
	if len(items) == 0 {
		return ""
	}
	data, _ := json.Marshal(items)
	return string(data)
}

func unmarshalTemplateTags(input sql.NullString) []string {
	if !input.Valid || strings.TrimSpace(input.String) == "" {
		return nil
	}
	parts := strings.Split(input.String, ",")
	tags := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			tags = append(tags, part)
		}
	}
	return tags
}

func unmarshalVisibleRange(input sql.NullString) []string {
	if !input.Valid || strings.TrimSpace(input.String) == "" {
		return nil
	}
	var items []string
	if err := json.Unmarshal([]byte(input.String), &items); err != nil {
		return nil
	}
	return items
}

func scanWorkflowTemplate(row *sql.Row) (*types.WorkflowTemplate, error) {
	var item types.WorkflowTemplate
	var categoryName, description, workflowDef, tags, icon, visibleRange sql.NullString
	var status, visibleScope int

	err := row.Scan(
		&item.TemplateID,
		&item.TemplateName,
		&item.CategoryID,
		&categoryName,
		&description,
		&workflowDef,
		&item.Version,
		&item.Author,
		&tags,
		&icon,
		&status,
		&visibleScope,
		&visibleRange,
		&item.InstallCount,
		&item.StartCount,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("workflow template not found")
		}
		return nil, wrapDBErr(err, "scanWorkflowTemplate")
	}
	fillWorkflowTemplate(&item, categoryName, description, workflowDef, tags, icon, status, visibleScope, visibleRange)
	return &item, nil
}

func scanWorkflowTemplateRow(rows *sql.Rows) (*types.WorkflowTemplate, error) {
	var item types.WorkflowTemplate
	var categoryName, description, workflowDef, tags, icon, visibleRange sql.NullString
	var status, visibleScope int

	err := rows.Scan(
		&item.TemplateID,
		&item.TemplateName,
		&item.CategoryID,
		&categoryName,
		&description,
		&workflowDef,
		&item.Version,
		&item.Author,
		&tags,
		&icon,
		&status,
		&visibleScope,
		&visibleRange,
		&item.InstallCount,
		&item.StartCount,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	if err != nil {
		return nil, wrapDBErr(err, "scanWorkflowTemplateRow")
	}
	fillWorkflowTemplate(&item, categoryName, description, workflowDef, tags, icon, status, visibleScope, visibleRange)
	return &item, nil
}

func fillWorkflowTemplate(
	item *types.WorkflowTemplate,
	categoryName, description, workflowDef, tags, icon sql.NullString,
	status, visibleScope int,
	visibleRange sql.NullString,
) {
	if categoryName.Valid {
		item.CategoryName = categoryName.String
	}
	if description.Valid {
		item.Description = description.String
	}
	if workflowDef.Valid && workflowDef.String != "" {
		item.WorkflowDef = &types.WorkflowDef{}
		_ = json.Unmarshal([]byte(workflowDef.String), item.WorkflowDef)
	}
	item.Tags = unmarshalTemplateTags(tags)
	if icon.Valid {
		item.Icon = icon.String
	}
	item.Status = types.WorkflowTemplateStatus(status)
	item.VisibleScope = types.WorkflowTemplateVisibleScope(visibleScope)
	item.VisibleRange = unmarshalVisibleRange(visibleRange)
}
