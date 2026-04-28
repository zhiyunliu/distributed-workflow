package sqlserver

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/zhiyunliu/distributed-workflow/interfaces"
	"github.com/zhiyunliu/distributed-workflow/types"
)

// 编译时接口断言
var _ interfaces.FormRepository = (*FormRepository)(nil)

// FormRepository SQL Server 表单仓库实现
type FormRepository struct {
	db *sql.DB
}

// NewFormRepository 创建表单仓库实例，复用已有 *sql.DB 连接池
func NewFormRepository(db *sql.DB) *FormRepository {
	return &FormRepository{db: db}
}

// ─────────────────────────────────────────────────────────────────────────────
// 表单定义相关
// ─────────────────────────────────────────────────────────────────────────────

// CreateFormDef 创建表单定义
func (r *FormRepository) CreateFormDef(ctx context.Context, def *types.FormDefinition) error {
	schemaJSON, err := marshalFormSchema(def.FormSchema)
	if err != nil {
		return fmt.Errorf("marshal form_schema: %w", err)
	}

	const q = `
INSERT INTO form_definition (form_id, form_name, description, form_schema, version, status, created_by, created_at, updated_at, deleted)
VALUES (@formID, @formName, @description, @formSchema, @version, @status, @createdBy, @createdAt, @updatedAt, 0)`

	_, err = r.db.ExecContext(ctx, q,
		sql.Named("formID", def.FormID),
		sql.Named("formName", def.FormName),
		sql.Named("description", nullString(def.Description)),
		sql.Named("formSchema", schemaJSON),
		sql.Named("version", def.Version),
		sql.Named("status", def.Status),
		sql.Named("createdBy", def.CreatedBy),
		sql.Named("createdAt", def.CreatedAt),
		sql.Named("updatedAt", def.UpdatedAt),
	)
	return wrapDBErr(err, "CreateFormDef")
}

// GetFormDef 获取表单定义
func (r *FormRepository) GetFormDef(ctx context.Context, formID string) (*types.FormDefinition, error) {
	const q = `
SELECT form_id, form_name, description, form_schema, version, status, created_by, created_at, updated_at, deleted
FROM form_definition
WHERE form_id = @formID AND deleted = 0`

	row := r.db.QueryRowContext(ctx, q, sql.Named("formID", formID))
	return scanFormDef(row)
}

// UpdateFormDef 更新表单定义（不修改 version/status，由 PublishFormDef 负责）
func (r *FormRepository) UpdateFormDef(ctx context.Context, def *types.FormDefinition) error {
	schemaJSON, err := marshalFormSchema(def.FormSchema)
	if err != nil {
		return fmt.Errorf("marshal form_schema: %w", err)
	}

	const q = `
UPDATE form_definition
SET form_name   = @formName,
    description = @description,
    form_schema = @formSchema,
    updated_at  = GETDATE()
WHERE form_id = @formID AND deleted = 0`

	_, err = r.db.ExecContext(ctx, q,
		sql.Named("formName", def.FormName),
		sql.Named("description", nullString(def.Description)),
		sql.Named("formSchema", schemaJSON),
		sql.Named("formID", def.FormID),
	)
	return wrapDBErr(err, "UpdateFormDef")
}

// ListFormDefs 分页查询表单定义，支持 status 过滤和关键字搜索
func (r *FormRepository) ListFormDefs(ctx context.Context, params types.FormListParams) ([]*types.FormDefinition, int64, error) {
	page := params.Page
	if page <= 0 {
		page = 1
	}
	pageSize := params.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	// 构建动态 WHERE 子句
	whereParts := []string{"deleted = 0"}
	args := []interface{}{}
	argIdx := 1

	if params.Status != nil {
		whereParts = append(whereParts, fmt.Sprintf("status = @p%d", argIdx))
		args = append(args, sql.Named(fmt.Sprintf("p%d", argIdx), *params.Status))
		argIdx++
	}
	if params.Keyword != "" {
		whereParts = append(whereParts, fmt.Sprintf("(form_name LIKE @p%d OR description LIKE @p%d)", argIdx, argIdx))
		args = append(args, sql.Named(fmt.Sprintf("p%d", argIdx), "%"+params.Keyword+"%"))
		argIdx++
	}

	where := strings.Join(whereParts, " AND ")

	// 查询总数
	countQ := fmt.Sprintf("SELECT COUNT(1) FROM form_definition WHERE %s", where)
	var total int64
	if err := r.db.QueryRowContext(ctx, countQ, args...).Scan(&total); err != nil {
		return nil, 0, wrapDBErr(err, "ListFormDefs count")
	}
	if total == 0 {
		return []*types.FormDefinition{}, 0, nil
	}

	// 分页查询
	dataQ := fmt.Sprintf(`
SELECT form_id, form_name, description, form_schema, version, status, created_by, created_at, updated_at, deleted
FROM form_definition
WHERE %s
ORDER BY created_at DESC
OFFSET @offset ROWS FETCH NEXT @pageSize ROWS ONLY`, where)

	pageArgs := append(args,
		sql.Named("offset", offset),
		sql.Named("pageSize", pageSize),
	)

	rows, err := r.db.QueryContext(ctx, dataQ, pageArgs...)
	if err != nil {
		return nil, 0, wrapDBErr(err, "ListFormDefs query")
	}
	defer rows.Close()

	var defs []*types.FormDefinition
	for rows.Next() {
		def, err := scanFormDefRow(rows)
		if err != nil {
			return nil, 0, err
		}
		defs = append(defs, def)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, wrapDBErr(err, "ListFormDefs rows")
	}
	return defs, total, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// 版本管理相关
// ─────────────────────────────────────────────────────────────────────────────

// PublishFormDef 发布表单版本：递增 version，写入 form_version_history，更新 status=1
func (r *FormRepository) PublishFormDef(ctx context.Context, formID, changeLog, operatorID string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return wrapDBErr(err, "PublishFormDef begin tx")
	}

	// 1. 查询当前 version 和 form_schema
	var currentVersion int
	var schemaJSON string
	const selQ = `SELECT version, form_schema FROM form_definition WHERE form_id = @formID AND deleted = 0`
	if err = tx.QueryRowContext(ctx, selQ, sql.Named("formID", formID)).Scan(&currentVersion, &schemaJSON); err != nil {
		_ = tx.Rollback()
		if err == sql.ErrNoRows {
			return fmt.Errorf("form definition '%s' not found", formID)
		}
		return wrapDBErr(err, "PublishFormDef select version")
	}

	newVersion := currentVersion + 1

	// 2. 写入版本历史
	const insQ = `
INSERT INTO form_version_history (form_id, version, form_schema, change_log, created_by, created_at)
VALUES (@formID, @version, @formSchema, @changeLog, @createdBy, GETDATE())`

	if _, err = tx.ExecContext(ctx, insQ,
		sql.Named("formID", formID),
		sql.Named("version", newVersion),
		sql.Named("formSchema", schemaJSON),
		sql.Named("changeLog", nullString(changeLog)),
		sql.Named("createdBy", operatorID),
	); err != nil {
		_ = tx.Rollback()
		return wrapDBErr(err, "PublishFormDef insert history")
	}

	// 3. 更新 form_definition：version、status、updated_at
	const updQ = `
UPDATE form_definition
SET version    = @version,
    status     = @status,
    updated_at = GETDATE()
WHERE form_id = @formID`

	if _, err = tx.ExecContext(ctx, updQ,
		sql.Named("version", newVersion),
		sql.Named("status", types.FormStatusPublished),
		sql.Named("formID", formID),
	); err != nil {
		_ = tx.Rollback()
		return wrapDBErr(err, "PublishFormDef update definition")
	}

	return wrapDBErr(tx.Commit(), "PublishFormDef commit")
}

// RollbackFormDef 回滚表单到指定版本：取历史 schema，创建新版本记录，更新定义
func (r *FormRepository) RollbackFormDef(ctx context.Context, formID string, version int, operatorID string) error {
	// 1. 查询历史版本的 form_schema
	const histQ = `SELECT form_schema FROM form_version_history WHERE form_id = @formID AND version = @version`
	var histSchemaJSON string
	if err := r.db.QueryRowContext(ctx, histQ,
		sql.Named("formID", formID),
		sql.Named("version", version),
	).Scan(&histSchemaJSON); err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("form version history '%s' v%d not found", formID, version)
		}
		return wrapDBErr(err, "RollbackFormDef select history")
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return wrapDBErr(err, "RollbackFormDef begin tx")
	}

	// 2. 查询当前 version
	var currentVersion int
	const selQ = `SELECT version FROM form_definition WHERE form_id = @formID AND deleted = 0`
	if err = tx.QueryRowContext(ctx, selQ, sql.Named("formID", formID)).Scan(&currentVersion); err != nil {
		_ = tx.Rollback()
		if err == sql.ErrNoRows {
			return fmt.Errorf("form definition '%s' not found", formID)
		}
		return wrapDBErr(err, "RollbackFormDef select current version")
	}

	newVersion := currentVersion + 1
	changeLog := fmt.Sprintf("rollback to v%d", version)

	// 3. 写入新版本历史记录（内容为回滚目标 schema）
	const insQ = `
INSERT INTO form_version_history (form_id, version, form_schema, change_log, created_by, created_at)
VALUES (@formID, @version, @formSchema, @changeLog, @createdBy, GETDATE())`

	if _, err = tx.ExecContext(ctx, insQ,
		sql.Named("formID", formID),
		sql.Named("version", newVersion),
		sql.Named("formSchema", histSchemaJSON),
		sql.Named("changeLog", changeLog),
		sql.Named("createdBy", operatorID),
	); err != nil {
		_ = tx.Rollback()
		return wrapDBErr(err, "RollbackFormDef insert history")
	}

	// 4. 更新 form_definition 的 form_schema 和 version
	const updQ = `
UPDATE form_definition
SET form_schema = @formSchema,
    version     = @version,
    updated_at  = GETDATE()
WHERE form_id = @formID`

	if _, err = tx.ExecContext(ctx, updQ,
		sql.Named("formSchema", histSchemaJSON),
		sql.Named("version", newVersion),
		sql.Named("formID", formID),
	); err != nil {
		_ = tx.Rollback()
		return wrapDBErr(err, "RollbackFormDef update definition")
	}

	return wrapDBErr(tx.Commit(), "RollbackFormDef commit")
}

// GetFormVersionHistory 查询表单所有版本历史（按版本号降序）
func (r *FormRepository) GetFormVersionHistory(ctx context.Context, formID string) ([]*types.FormVersionHistory, error) {
	const q = `
SELECT id, form_id, version, form_schema, change_log, created_by, created_at
FROM form_version_history
WHERE form_id = @formID
ORDER BY version DESC`

	rows, err := r.db.QueryContext(ctx, q, sql.Named("formID", formID))
	if err != nil {
		return nil, wrapDBErr(err, "GetFormVersionHistory")
	}
	defer rows.Close()

	var histories []*types.FormVersionHistory
	for rows.Next() {
		h, err := scanFormVersionHistoryRow(rows)
		if err != nil {
			return nil, err
		}
		histories = append(histories, h)
	}
	return histories, wrapDBErr(rows.Err(), "GetFormVersionHistory rows")
}

// ─────────────────────────────────────────────────────────────────────────────
// 表单实例相关
// ─────────────────────────────────────────────────────────────────────────────

// CreateFormInstance 创建表单实例
func (r *FormRepository) CreateFormInstance(ctx context.Context, inst *types.FormInstance) error {
	dataJSON, err := marshalFormSchema(inst.FormData)
	if err != nil {
		return fmt.Errorf("marshal form_data: %w", err)
	}

	const q = `
INSERT INTO form_instance (instance_id, form_id, form_version, workflow_instance_id, node_id, form_data, status, created_by, created_at, updated_at)
VALUES (@instanceID, @formID, @formVersion, @workflowInstanceID, @nodeID, @formData, @status, @createdBy, @createdAt, @updatedAt)`

	_, err = r.db.ExecContext(ctx, q,
		sql.Named("instanceID", inst.InstanceID),
		sql.Named("formID", inst.FormID),
		sql.Named("formVersion", inst.FormVersion),
		sql.Named("workflowInstanceID", inst.WorkflowInstanceID),
		sql.Named("nodeID", nullString(inst.NodeID)),
		sql.Named("formData", dataJSON),
		sql.Named("status", inst.Status),
		sql.Named("createdBy", inst.CreatedBy),
		sql.Named("createdAt", inst.CreatedAt),
		sql.Named("updatedAt", inst.UpdatedAt),
	)
	return wrapDBErr(err, "CreateFormInstance")
}

// UpdateFormInstance 更新表单实例（form_data、status）
func (r *FormRepository) UpdateFormInstance(ctx context.Context, inst *types.FormInstance) error {
	dataJSON, err := marshalFormSchema(inst.FormData)
	if err != nil {
		return fmt.Errorf("marshal form_data: %w", err)
	}

	const q = `
UPDATE form_instance
SET form_data  = @formData,
    status     = @status,
    updated_at = GETDATE()
WHERE instance_id = @instanceID`

	_, err = r.db.ExecContext(ctx, q,
		sql.Named("formData", dataJSON),
		sql.Named("status", inst.Status),
		sql.Named("instanceID", inst.InstanceID),
	)
	return wrapDBErr(err, "UpdateFormInstance")
}

// GetFormInstance 按 instance_id 查询表单实例
func (r *FormRepository) GetFormInstance(ctx context.Context, instanceID string) (*types.FormInstance, error) {
	const q = `
SELECT instance_id, form_id, form_version, workflow_instance_id, node_id, form_data, status, created_by, created_at, updated_at
FROM form_instance
WHERE instance_id = @instanceID`

	row := r.db.QueryRowContext(ctx, q, sql.Named("instanceID", instanceID))
	inst, err := scanFormInstance(row)
	if err != nil {
		return nil, err
	}
	return inst, nil
}

// GetFormInstanceByWorkflow 按流程实例 ID 和节点 ID 查询表单实例
func (r *FormRepository) GetFormInstanceByWorkflow(ctx context.Context, workflowInstanceID, nodeID string) (*types.FormInstance, error) {
	const q = `
SELECT instance_id, form_id, form_version, workflow_instance_id, node_id, form_data, status, created_by, created_at, updated_at
FROM form_instance
WHERE workflow_instance_id = @workflowInstanceID AND node_id = @nodeID`

	row := r.db.QueryRowContext(ctx, q,
		sql.Named("workflowInstanceID", workflowInstanceID),
		sql.Named("nodeID", nodeID),
	)
	inst, err := scanFormInstance(row)
	if err != nil {
		return nil, err
	}
	return inst, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// 私有辅助函数
// ─────────────────────────────────────────────────────────────────────────────

// marshalFormSchema 序列化 map[string]interface{} 为 JSON 字符串
func marshalFormSchema(m map[string]interface{}) (string, error) {
	if len(m) == 0 {
		return "{}", nil
	}
	b, err := json.Marshal(m)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// unmarshalFormSchema 反序列化 JSON 字符串为 map[string]interface{}
func unmarshalFormSchema(s string) (map[string]interface{}, error) {
	if s == "" || s == "{}" {
		return map[string]interface{}{}, nil
	}
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(s), &m); err != nil {
		return nil, err
	}
	return m, nil
}

// nullStr 同 nullString，提供内部别名便于模板仓库风格一致
func nullStr(s string) interface{} {
	return nullString(s)
}

// rowScanner 统一 *sql.Row 与 *sql.Rows 的 Scan 方法
type rowScanner interface {
	Scan(dest ...interface{}) error
}

// scanFormDef 从 *sql.Row 扫描 FormDefinition
func scanFormDef(row *sql.Row) (*types.FormDefinition, error) {
	var def types.FormDefinition
	var description sql.NullString
	var schemaJSON string
	var deleted bool

	err := row.Scan(
		&def.FormID, &def.FormName, &description,
		&schemaJSON, &def.Version, &def.Status,
		&def.CreatedBy, &def.CreatedAt, &def.UpdatedAt, &deleted,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("form definition not found")
		}
		return nil, wrapDBErr(err, "scanFormDef")
	}
	if description.Valid {
		def.Description = description.String
	}
	def.Deleted = deleted

	schema, err := unmarshalFormSchema(schemaJSON)
	if err != nil {
		return nil, fmt.Errorf("unmarshal form_schema: %w", err)
	}
	def.FormSchema = schema
	return &def, nil
}

// scanFormDefRow 从 *sql.Rows 扫描 FormDefinition
func scanFormDefRow(rows *sql.Rows) (*types.FormDefinition, error) {
	var def types.FormDefinition
	var description sql.NullString
	var schemaJSON string
	var deleted bool

	if err := rows.Scan(
		&def.FormID, &def.FormName, &description,
		&schemaJSON, &def.Version, &def.Status,
		&def.CreatedBy, &def.CreatedAt, &def.UpdatedAt, &deleted,
	); err != nil {
		return nil, wrapDBErr(err, "scanFormDefRow")
	}
	if description.Valid {
		def.Description = description.String
	}
	def.Deleted = deleted

	schema, err := unmarshalFormSchema(schemaJSON)
	if err != nil {
		return nil, fmt.Errorf("unmarshal form_schema: %w", err)
	}
	def.FormSchema = schema
	return &def, nil
}

// scanFormVersionHistoryRow 从 *sql.Rows 扫描 FormVersionHistory
func scanFormVersionHistoryRow(rows *sql.Rows) (*types.FormVersionHistory, error) {
	var h types.FormVersionHistory
	var schemaJSON string
	var changeLog sql.NullString

	if err := rows.Scan(
		&h.ID, &h.FormID, &h.Version,
		&schemaJSON, &changeLog, &h.CreatedBy, &h.CreatedAt,
	); err != nil {
		return nil, wrapDBErr(err, "scanFormVersionHistoryRow")
	}
	if changeLog.Valid {
		h.ChangeLog = changeLog.String
	}

	schema, err := unmarshalFormSchema(schemaJSON)
	if err != nil {
		return nil, fmt.Errorf("unmarshal form_schema version history: %w", err)
	}
	h.FormSchema = schema
	return &h, nil
}

// scanFormInstance 从 *sql.Row 扫描 FormInstance
func scanFormInstance(row *sql.Row) (*types.FormInstance, error) {
	var inst types.FormInstance
	var nodeID sql.NullString
	var dataJSON string

	err := row.Scan(
		&inst.InstanceID, &inst.FormID, &inst.FormVersion,
		&inst.WorkflowInstanceID, &nodeID,
		&dataJSON, &inst.Status,
		&inst.CreatedBy, &inst.CreatedAt, &inst.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("form instance not found")
		}
		return nil, wrapDBErr(err, "scanFormInstance")
	}
	if nodeID.Valid {
		inst.NodeID = nodeID.String
	}

	formData, err := unmarshalFormSchema(dataJSON)
	if err != nil {
		return nil, fmt.Errorf("unmarshal form_data: %w", err)
	}
	inst.FormData = formData
	return &inst, nil
}
