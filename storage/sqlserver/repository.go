package sqlserver

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	_ "github.com/microsoft/go-mssqldb"
	"github.com/rs/zerolog/log"

	"github.com/zhiyunliu/distributed-workflow/types"
)

const (
	// 查询超时默认值
	defaultQueryTimeout = 10 * time.Second
)

// Config SQL Server 连接配置
type Config struct {
	// DSN 数据库连接字符串，格式：sqlserver://user:password@host:port?database=dbname
	DSN string
	// MaxOpenConns 最大打开连接数，默认 25
	MaxOpenConns int
	// MaxIdleConns 最大空闲连接数，默认 10
	MaxIdleConns int
	// ConnMaxLifetime 连接最大生命周期，默认 30min
	ConnMaxLifetime time.Duration
}

// Repository SQL Server WorkflowRepository 实现
type Repository struct {
	db *sql.DB
}

// NewRepository 创建 SQL Server 存储实现，建立连接池
func NewRepository(cfg Config) (*Repository, error) {
	db, err := sql.Open("sqlserver", cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("open sqlserver connection: %w", err)
	}

	maxOpen := cfg.MaxOpenConns
	if maxOpen <= 0 {
		maxOpen = 25
	}
	maxIdle := cfg.MaxIdleConns
	if maxIdle <= 0 {
		maxIdle = 10
	}
	connLife := cfg.ConnMaxLifetime
	if connLife <= 0 {
		connLife = 30 * time.Minute
	}

	db.SetMaxOpenConns(maxOpen)
	db.SetMaxIdleConns(maxIdle)
	db.SetConnMaxLifetime(connLife)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err = db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("ping sqlserver: %w", err)
	}
	log.Info().Str("dsn_prefix", safeDSN(cfg.DSN)).Msg("sqlserver connected")
	return &Repository{db: db}, nil
}

// DB 返回底层数据库连接，供 D6 各子仓库复用
func (r *Repository) DB() *sql.DB {
	return r.db
}

// Close 关闭数据库连接池
func (r *Repository) Close() error {
	return r.db.Close()
}

// ─────────────────────────────────────────────────────────────────────────────
// 工作流定义相关
// ─────────────────────────────────────────────────────────────────────────────

// CreateWorkflowDef 新建工作流定义
func (r *Repository) CreateWorkflowDef(def *types.WorkflowDef) error {
	defJSON, err := json.Marshal(def)
	if err != nil {
		return fmt.Errorf("marshal workflow def: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
	defer cancel()

	const q = `
INSERT INTO workflow_defs (id, name, description, latest_version, definition, created_at, updated_at, created_by, updated_by, disabled)
VALUES (@id, @name, @desc, 1, @def, @createdAt, @updatedAt, @createdBy, @updatedBy, @disabled)`

	_, err = r.db.ExecContext(ctx, q,
		sql.Named("id", def.ID),
		sql.Named("name", def.Name),
		sql.Named("desc", def.Description),
		sql.Named("def", string(defJSON)),
		sql.Named("createdAt", def.CreatedAt),
		sql.Named("updatedAt", def.UpdatedAt),
		sql.Named("createdBy", def.CreatedBy),
		sql.Named("updatedBy", def.UpdatedBy),
		sql.Named("disabled", def.Disabled),
	)
	return wrapDBErr(err, "CreateWorkflowDef")
}

// GetWorkflowDef 按 ID 查询工作流定义
func (r *Repository) GetWorkflowDef(workflowID string) (*types.WorkflowDef, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
	defer cancel()

	const q = `SELECT definition FROM workflow_defs WHERE id = @id`
	row := r.db.QueryRowContext(ctx, q, sql.Named("id", workflowID))

	var defJSON string
	if err := row.Scan(&defJSON); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("workflow def '%s' not found", workflowID)
		}
		return nil, wrapDBErr(err, "GetWorkflowDef")
	}
	var def types.WorkflowDef
	if err := json.Unmarshal([]byte(defJSON), &def); err != nil {
		return nil, fmt.Errorf("unmarshal workflow def: %w", err)
	}
	return &def, nil
}

// UpdateWorkflowDef 更新工作流定义
func (r *Repository) UpdateWorkflowDef(def *types.WorkflowDef) error {
	defJSON, err := json.Marshal(def)
	if err != nil {
		return fmt.Errorf("marshal workflow def: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
	defer cancel()

	const q = `
UPDATE workflow_defs
SET name = @name, description = @desc, definition = @def,
    updated_at = @updatedAt, updated_by = @updatedBy, disabled = @disabled
WHERE id = @id`

	_, err = r.db.ExecContext(ctx, q,
		sql.Named("name", def.Name),
		sql.Named("desc", def.Description),
		sql.Named("def", string(defJSON)),
		sql.Named("updatedAt", def.UpdatedAt),
		sql.Named("updatedBy", def.UpdatedBy),
		sql.Named("disabled", def.Disabled),
		sql.Named("id", def.ID),
	)
	return wrapDBErr(err, "UpdateWorkflowDef")
}

// DeleteWorkflowDef 删除工作流定义（硬删除，仅限无关联实例时调用）
func (r *Repository) DeleteWorkflowDef(workflowID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
	defer cancel()

	const q = `DELETE FROM workflow_defs WHERE id = @id`
	_, err := r.db.ExecContext(ctx, q, sql.Named("id", workflowID))
	return wrapDBErr(err, "DeleteWorkflowDef")
}

// ListWorkflowDefs 查询所有工作流定义
func (r *Repository) ListWorkflowDefs() ([]*types.WorkflowDef, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
	defer cancel()

	const q = `SELECT definition FROM workflow_defs WHERE disabled = 0`
	rows, err := r.db.QueryContext(ctx, q)
	if err != nil {
		return nil, wrapDBErr(err, "ListWorkflowDefs")
	}
	defer rows.Close()

	var defs []*types.WorkflowDef
	for rows.Next() {
		var defJSON string
		if err := rows.Scan(&defJSON); err != nil {
			return nil, wrapDBErr(err, "ListWorkflowDefs scan")
		}
		var def types.WorkflowDef
		if err := json.Unmarshal([]byte(defJSON), &def); err != nil {
			return nil, fmt.Errorf("unmarshal workflow def: %w", err)
		}
		defs = append(defs, &def)
	}
	return defs, rows.Err()
}

// ─────────────────────────────────────────────────────────────────────────────
// 流程实例相关
// ─────────────────────────────────────────────────────────────────────────────

// CreateWorkflowInstance 创建流程实例
func (r *Repository) CreateWorkflowInstance(instance *types.WorkflowInstance) error {
	inputJSON, err := marshalJSON(instance.InputData)
	if err != nil {
		return fmt.Errorf("marshal input_data: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
	defer cancel()

	const q = `
INSERT INTO workflow_instances (id, workflow_id, workflow_version, status, start_time, input_data, created_by)
VALUES (@id, @workflowID, @version, @status, @startTime, @inputData, @createdBy)`

	_, err = r.db.ExecContext(ctx, q,
		sql.Named("id", instance.ID),
		sql.Named("workflowID", instance.WorkflowID),
		sql.Named("version", instance.WorkflowVersion),
		sql.Named("status", string(instance.Status)),
		sql.Named("startTime", instance.StartTime),
		sql.Named("inputData", inputJSON),
		sql.Named("createdBy", instance.CreatedBy),
	)
	return wrapDBErr(err, "CreateWorkflowInstance")
}

// UpdateWorkflowInstance 更新流程实例（状态、结束时间、输出、错误信息）
func (r *Repository) UpdateWorkflowInstance(instance *types.WorkflowInstance) error {
	outputJSON, err := marshalJSON(instance.OutputData)
	if err != nil {
		return fmt.Errorf("marshal output_data: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
	defer cancel()

	const q = `
UPDATE workflow_instances
SET status = @status, end_time = @endTime, output_data = @outputData, error_message = @errMsg
WHERE id = @id`

	var endTime interface{}
	if instance.EndTime != nil {
		endTime = *instance.EndTime
	}

	_, err = r.db.ExecContext(ctx, q,
		sql.Named("status", string(instance.Status)),
		sql.Named("endTime", endTime),
		sql.Named("outputData", outputJSON),
		sql.Named("errMsg", instance.ErrorMessage),
		sql.Named("id", instance.ID),
	)
	return wrapDBErr(err, "UpdateWorkflowInstance")
}

// GetWorkflowInstance 查询流程实例
func (r *Repository) GetWorkflowInstance(instanceID string) (*types.WorkflowInstance, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
	defer cancel()

	const q = `
SELECT id, workflow_id, workflow_version, status, start_time, end_time,
       input_data, output_data, created_by, error_message
FROM workflow_instances WHERE id = @id`

	row := r.db.QueryRowContext(ctx, q, sql.Named("id", instanceID))
	return scanInstance(row)
}

// ListUnfinishedInstances 查询所有未完成的流程实例（pending/running）
func (r *Repository) ListUnfinishedInstances() ([]*types.WorkflowInstance, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
	defer cancel()

	const q = `
SELECT id, workflow_id, workflow_version, status, start_time, end_time,
       input_data, output_data, created_by, error_message
FROM workflow_instances WHERE status IN ('pending', 'running')`

	rows, err := r.db.QueryContext(ctx, q)
	if err != nil {
		return nil, wrapDBErr(err, "ListUnfinishedInstances")
	}
	defer rows.Close()

	var instances []*types.WorkflowInstance
	for rows.Next() {
		inst, err := scanInstanceRow(rows)
		if err != nil {
			return nil, err
		}
		instances = append(instances, inst)
	}
	return instances, rows.Err()
}

// ─────────────────────────────────────────────────────────────────────────────
// 节点状态相关
// ─────────────────────────────────────────────────────────────────────────────

// CreateWorkflowNodeState 创建节点执行状态记录
func (r *Repository) CreateWorkflowNodeState(state *types.WorkflowNodeState) error {
	inputJSON, err := marshalJSON(state.InputData)
	if err != nil {
		return fmt.Errorf("marshal node input_data: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
	defer cancel()

	const q = `
INSERT INTO workflow_node_states (instance_id, node_id, status, assigned_to, worker_ip,
    start_time, input_data, retry_count)
VALUES (@instanceID, @nodeID, @status, @assignedTo, @workerIP,
    @startTime, @inputData, @retryCount)`

	_, err = r.db.ExecContext(ctx, q,
		sql.Named("instanceID", state.InstanceID),
		sql.Named("nodeID", state.NodeID),
		sql.Named("status", string(state.Status)),
		sql.Named("assignedTo", nullString(state.AssignedTo)),
		sql.Named("workerIP", nullString(state.WorkerIP)),
		sql.Named("startTime", nullTime(state.StartTime)),
		sql.Named("inputData", inputJSON),
		sql.Named("retryCount", state.RetryCount),
	)
	return wrapDBErr(err, "CreateWorkflowNodeState")
}

// UpdateWorkflowNodeState 更新节点执行状态
func (r *Repository) UpdateWorkflowNodeState(state *types.WorkflowNodeState) error {
	outputJSON, err := marshalJSON(state.OutputData)
	if err != nil {
		return fmt.Errorf("marshal node output_data: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
	defer cancel()

	const q = `
UPDATE workflow_node_states
SET status = @status, assigned_to = @assignedTo, worker_ip = @workerIP,
    start_time = @startTime, end_time = @endTime,
    output_data = @outputData, error_message = @errMsg, retry_count = @retryCount
WHERE instance_id = @instanceID AND node_id = @nodeID`

	_, err = r.db.ExecContext(ctx, q,
		sql.Named("status", string(state.Status)),
		sql.Named("assignedTo", nullString(state.AssignedTo)),
		sql.Named("workerIP", nullString(state.WorkerIP)),
		sql.Named("startTime", nullTime(state.StartTime)),
		sql.Named("endTime", nullTime(state.EndTime)),
		sql.Named("outputData", outputJSON),
		sql.Named("errMsg", state.ErrorMessage),
		sql.Named("retryCount", state.RetryCount),
		sql.Named("instanceID", state.InstanceID),
		sql.Named("nodeID", state.NodeID),
	)
	return wrapDBErr(err, "UpdateWorkflowNodeState")
}

// GetWorkflowNodeState 查询指定节点状态
func (r *Repository) GetWorkflowNodeState(instanceID string, nodeID string) (*types.WorkflowNodeState, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
	defer cancel()

	const q = `
SELECT id, instance_id, node_id, status, assigned_to, worker_ip,
       start_time, end_time, input_data, output_data, error_message, retry_count
FROM workflow_node_states WHERE instance_id = @instanceID AND node_id = @nodeID`

	row := r.db.QueryRowContext(ctx, q,
		sql.Named("instanceID", instanceID),
		sql.Named("nodeID", nodeID),
	)
	return scanNodeState(row)
}

// ListWorkflowNodeStates 查询某实例下所有节点状态
func (r *Repository) ListWorkflowNodeStates(instanceID string) ([]*types.WorkflowNodeState, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
	defer cancel()

	const q = `
SELECT id, instance_id, node_id, status, assigned_to, worker_ip,
       start_time, end_time, input_data, output_data, error_message, retry_count
FROM workflow_node_states WHERE instance_id = @instanceID`

	rows, err := r.db.QueryContext(ctx, q, sql.Named("instanceID", instanceID))
	if err != nil {
		return nil, wrapDBErr(err, "ListWorkflowNodeStates")
	}
	defer rows.Close()

	var states []*types.WorkflowNodeState
	for rows.Next() {
		state, err := scanNodeStateRow(rows)
		if err != nil {
			return nil, err
		}
		states = append(states, state)
	}
	return states, rows.Err()
}

// ─────────────────────────────────────────────────────────────────────────────
// 上下文相关
// ─────────────────────────────────────────────────────────────────────────────

// CreateWorkflowContext 创建流程上下文（逐 key 插入）
func (r *Repository) CreateWorkflowContext(ctx *types.WorkflowContext) error {
	if len(ctx.Data) == 0 {
		return nil
	}
	dbCtx, cancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
	defer cancel()

	tx, err := r.db.BeginTx(dbCtx, nil)
	if err != nil {
		return wrapDBErr(err, "CreateWorkflowContext begin tx")
	}

	const q = `
INSERT INTO workflow_instance_context (instance_id, [key], value, created_at, updated_at)
VALUES (@instanceID, @key, @value, @createdAt, @updatedAt)`

	for k, v := range ctx.Data {
		valJSON, err := json.Marshal(v)
		if err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("marshal context value for key '%s': %w", k, err)
		}
		if _, err = tx.ExecContext(dbCtx, q,
			sql.Named("instanceID", ctx.InstanceID),
			sql.Named("key", k),
			sql.Named("value", string(valJSON)),
			sql.Named("createdAt", ctx.CreatedAt),
			sql.Named("updatedAt", ctx.UpdatedAt),
		); err != nil {
			_ = tx.Rollback()
			return wrapDBErr(err, "CreateWorkflowContext insert key="+k)
		}
	}
	return wrapDBErr(tx.Commit(), "CreateWorkflowContext commit")
}

// UpdateWorkflowContext 合并更新流程上下文（MERGE 语义：存在则更新，不存在则插入）
func (r *Repository) UpdateWorkflowContext(ctx *types.WorkflowContext) error {
	if len(ctx.Data) == 0 {
		return nil
	}
	dbCtx, cancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
	defer cancel()

	tx, err := r.db.BeginTx(dbCtx, nil)
	if err != nil {
		return wrapDBErr(err, "UpdateWorkflowContext begin tx")
	}

	const q = `
MERGE workflow_instance_context AS target
USING (VALUES (@instanceID, @key, @value, @updatedAt)) AS source(instance_id, [key], value, updated_at)
ON target.instance_id = source.instance_id AND target.[key] = source.[key]
WHEN MATCHED THEN
    UPDATE SET target.value = source.value, target.updated_at = source.updated_at
WHEN NOT MATCHED THEN
    INSERT (instance_id, [key], value, created_at, updated_at)
    VALUES (source.instance_id, source.[key], source.value, source.updated_at, source.updated_at);`

	now := time.Now()
	for k, v := range ctx.Data {
		valJSON, err := json.Marshal(v)
		if err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("marshal context value for key '%s': %w", k, err)
		}
		if _, err = tx.ExecContext(dbCtx, q,
			sql.Named("instanceID", ctx.InstanceID),
			sql.Named("key", k),
			sql.Named("value", string(valJSON)),
			sql.Named("updatedAt", now),
		); err != nil {
			_ = tx.Rollback()
			return wrapDBErr(err, "UpdateWorkflowContext merge key="+k)
		}
	}
	return wrapDBErr(tx.Commit(), "UpdateWorkflowContext commit")
}

// GetWorkflowContext 查询流程上下文，重建 Data map
func (r *Repository) GetWorkflowContext(instanceID string) (*types.WorkflowContext, error) {
	dbCtx, cancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
	defer cancel()

	const q = `
SELECT [key], value, created_at, updated_at
FROM workflow_instance_context WHERE instance_id = @instanceID`

	rows, err := r.db.QueryContext(dbCtx, q, sql.Named("instanceID", instanceID))
	if err != nil {
		return nil, wrapDBErr(err, "GetWorkflowContext")
	}
	defer rows.Close()

	ctx := &types.WorkflowContext{
		InstanceID: instanceID,
		Data:       make(map[string]interface{}),
	}
	for rows.Next() {
		var key, valJSON string
		var createdAt, updatedAt time.Time
		if err := rows.Scan(&key, &valJSON, &createdAt, &updatedAt); err != nil {
			return nil, wrapDBErr(err, "GetWorkflowContext scan")
		}
		var val interface{}
		if err := json.Unmarshal([]byte(valJSON), &val); err != nil {
			return nil, fmt.Errorf("unmarshal context value key='%s': %w", key, err)
		}
		ctx.Data[key] = val
		if createdAt.After(ctx.UpdatedAt) {
			ctx.CreatedAt = createdAt
		}
		if updatedAt.After(ctx.UpdatedAt) {
			ctx.UpdatedAt = updatedAt
		}
	}
	return ctx, rows.Err()
}

// DeleteWorkflowContext 删除流程上下文
func (r *Repository) DeleteWorkflowContext(instanceID string) error {
	dbCtx, cancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
	defer cancel()

	const q = `DELETE FROM workflow_instance_context WHERE instance_id = @instanceID`
	_, err := r.db.ExecContext(dbCtx, q, sql.Named("instanceID", instanceID))
	return wrapDBErr(err, "DeleteWorkflowContext")
}

// ─────────────────────────────────────────────────────────────────────────────
// 私有辅助函数
// ─────────────────────────────────────────────────────────────────────────────

func wrapDBErr(err error, op string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("sqlserver %s: %w", op, err)
}

func marshalJSON(v interface{}) (string, error) {
	if v == nil {
		return "", nil
	}
	b, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func nullString(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

func nullTime(t *time.Time) interface{} {
	if t == nil {
		return nil
	}
	return *t
}

// safeDSN 返回 DSN 前缀，隐藏密码
func safeDSN(dsn string) string {
	if len(dsn) > 30 {
		return dsn[:30] + "..."
	}
	return dsn
}

// scanInstance 从 QueryRow 扫描单个 WorkflowInstance
func scanInstance(row *sql.Row) (*types.WorkflowInstance, error) {
	var inst types.WorkflowInstance
	var endTime sql.NullTime
	var inputJSON, outputJSON, errMsg sql.NullString

	err := row.Scan(
		&inst.ID, &inst.WorkflowID, &inst.WorkflowVersion,
		&inst.Status, &inst.StartTime, &endTime,
		&inputJSON, &outputJSON, &inst.CreatedBy, &errMsg,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("workflow instance not found")
		}
		return nil, wrapDBErr(err, "scanInstance")
	}
	if endTime.Valid {
		inst.EndTime = &endTime.Time
	}
	if inputJSON.Valid && inputJSON.String != "" {
		_ = json.Unmarshal([]byte(inputJSON.String), &inst.InputData)
	}
	if outputJSON.Valid && outputJSON.String != "" {
		_ = json.Unmarshal([]byte(outputJSON.String), &inst.OutputData)
	}
	if errMsg.Valid {
		inst.ErrorMessage = errMsg.String
	}
	return &inst, nil
}

// scanner 统一接口，支持 *sql.Row 和 *sql.Rows
type scanner interface {
	Scan(dest ...interface{}) error
}

func scanInstanceRow(rows *sql.Rows) (*types.WorkflowInstance, error) {
	var inst types.WorkflowInstance
	var endTime sql.NullTime
	var inputJSON, outputJSON, errMsg sql.NullString

	err := rows.Scan(
		&inst.ID, &inst.WorkflowID, &inst.WorkflowVersion,
		&inst.Status, &inst.StartTime, &endTime,
		&inputJSON, &outputJSON, &inst.CreatedBy, &errMsg,
	)
	if err != nil {
		return nil, wrapDBErr(err, "scanInstanceRow")
	}
	if endTime.Valid {
		inst.EndTime = &endTime.Time
	}
	if inputJSON.Valid && inputJSON.String != "" {
		_ = json.Unmarshal([]byte(inputJSON.String), &inst.InputData)
	}
	if outputJSON.Valid && outputJSON.String != "" {
		_ = json.Unmarshal([]byte(outputJSON.String), &inst.OutputData)
	}
	if errMsg.Valid {
		inst.ErrorMessage = errMsg.String
	}
	return &inst, nil
}

func scanNodeState(row *sql.Row) (*types.WorkflowNodeState, error) {
	var state types.WorkflowNodeState
	var assignedTo, workerIP, inputJSON, outputJSON, errMsg sql.NullString
	var startTime, endTime sql.NullTime

	err := row.Scan(
		&state.ID, &state.InstanceID, &state.NodeID,
		&state.Status, &assignedTo, &workerIP,
		&startTime, &endTime, &inputJSON, &outputJSON,
		&errMsg, &state.RetryCount,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("workflow node state not found")
		}
		return nil, wrapDBErr(err, "scanNodeState")
	}
	if assignedTo.Valid {
		state.AssignedTo = assignedTo.String
	}
	if workerIP.Valid {
		state.WorkerIP = workerIP.String
	}
	if startTime.Valid {
		state.StartTime = &startTime.Time
	}
	if endTime.Valid {
		state.EndTime = &endTime.Time
	}
	if inputJSON.Valid && inputJSON.String != "" {
		_ = json.Unmarshal([]byte(inputJSON.String), &state.InputData)
	}
	if outputJSON.Valid && outputJSON.String != "" {
		_ = json.Unmarshal([]byte(outputJSON.String), &state.OutputData)
	}
	if errMsg.Valid {
		state.ErrorMessage = errMsg.String
	}
	return &state, nil
}

func scanNodeStateRow(rows *sql.Rows) (*types.WorkflowNodeState, error) {
	var state types.WorkflowNodeState
	var assignedTo, workerIP, inputJSON, outputJSON, errMsg sql.NullString
	var startTime, endTime sql.NullTime

	err := rows.Scan(
		&state.ID, &state.InstanceID, &state.NodeID,
		&state.Status, &assignedTo, &workerIP,
		&startTime, &endTime, &inputJSON, &outputJSON,
		&errMsg, &state.RetryCount,
	)
	if err != nil {
		return nil, wrapDBErr(err, "scanNodeStateRow")
	}
	if assignedTo.Valid {
		state.AssignedTo = assignedTo.String
	}
	if workerIP.Valid {
		state.WorkerIP = workerIP.String
	}
	if startTime.Valid {
		state.StartTime = &startTime.Time
	}
	if endTime.Valid {
		state.EndTime = &endTime.Time
	}
	if inputJSON.Valid && inputJSON.String != "" {
		_ = json.Unmarshal([]byte(inputJSON.String), &state.InputData)
	}
	if outputJSON.Valid && outputJSON.String != "" {
		_ = json.Unmarshal([]byte(outputJSON.String), &state.OutputData)
	}
	if errMsg.Valid {
		state.ErrorMessage = errMsg.String
	}
	return &state, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// D2 新增：流程实例列表查询
// ─────────────────────────────────────────────────────────────────────────────

// ListWorkflowInstances 分页查询流程实例列表
func (r *Repository) ListWorkflowInstances(workflowID string, status types.WorkflowStatus, pageSize, pageNum int) ([]*types.WorkflowInstance, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
	defer cancel()

	if pageSize <= 0 {
		pageSize = 20
	}
	if pageNum <= 0 {
		pageNum = 1
	}
	offset := (pageNum - 1) * pageSize

	q := `
SELECT id, workflow_id, workflow_version, status, start_time, end_time,
       input_data, output_data, created_by, error_message
FROM workflow_instances WHERE 1=1`

	var args []interface{}
	argIdx := 1
	if workflowID != "" {
		q += fmt.Sprintf(" AND workflow_id = @p%d", argIdx)
		args = append(args, sql.Named(fmt.Sprintf("p%d", argIdx), workflowID))
		argIdx++
	}
	if status != "" {
		q += fmt.Sprintf(" AND status = @p%d", argIdx)
		args = append(args, sql.Named(fmt.Sprintf("p%d", argIdx), string(status)))
		argIdx++
	}
	_ = argIdx
	q += ` ORDER BY start_time DESC OFFSET @offset ROWS FETCH NEXT @pageSize ROWS ONLY`
	args = append(args, sql.Named("offset", offset), sql.Named("pageSize", pageSize))

	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, wrapDBErr(err, "ListWorkflowInstances")
	}
	defer rows.Close()

	var instances []*types.WorkflowInstance
	for rows.Next() {
		inst, err := scanInstanceRow(rows)
		if err != nil {
			return nil, err
		}
		instances = append(instances, inst)
	}
	return instances, rows.Err()
}

// ─────────────────────────────────────────────────────────────────────────────
// D2 新增：节点状态批量操作
// ─────────────────────────────────────────────────────────────────────────────

// BatchUpdateNodeStates 批量更新节点状态（取消/跳过场景）
func (r *Repository) BatchUpdateNodeStates(instanceID string, nodeIDs []string, status types.WorkflowNodeStatus, reason string) error {
	if len(nodeIDs) == 0 {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
	defer cancel()

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return wrapDBErr(err, "BatchUpdateNodeStates begin tx")
	}

	const q = `
UPDATE workflow_node_states
SET status = @status, skipped_reason = @reason, end_time = @endTime
WHERE instance_id = @instanceID AND node_id = @nodeID`

	now := time.Now()
	for _, nodeID := range nodeIDs {
		if _, err := tx.ExecContext(ctx, q,
			sql.Named("status", string(status)),
			sql.Named("reason", reason),
			sql.Named("endTime", now),
			sql.Named("instanceID", instanceID),
			sql.Named("nodeID", nodeID),
		); err != nil {
			_ = tx.Rollback()
			return wrapDBErr(err, "BatchUpdateNodeStates update node="+nodeID)
		}
	}
	return wrapDBErr(tx.Commit(), "BatchUpdateNodeStates commit")
}

// GetAssignedNodesByWorker 获取分配给指定 Worker 且仍处于 running/assigned 状态的节点
func (r *Repository) GetAssignedNodesByWorker(workerID string) ([]*types.WorkflowNodeState, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
	defer cancel()

	const q = `
SELECT id, instance_id, node_id, status, assigned_to, worker_ip,
       start_time, end_time, input_data, output_data, error_message, retry_count
FROM workflow_node_states
WHERE assigned_to = @workerID AND status IN ('running', 'assigned')`

	rows, err := r.db.QueryContext(ctx, q, sql.Named("workerID", workerID))
	if err != nil {
		return nil, wrapDBErr(err, "GetAssignedNodesByWorker")
	}
	defer rows.Close()

	var states []*types.WorkflowNodeState
	for rows.Next() {
		state, err := scanNodeStateRow(rows)
		if err != nil {
			return nil, err
		}
		states = append(states, state)
	}
	return states, rows.Err()
}

// GetFailedNodes 获取指定实例中所有失败节点
func (r *Repository) GetFailedNodes(instanceID string) ([]*types.WorkflowNodeState, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
	defer cancel()

	const q = `
SELECT id, instance_id, node_id, status, assigned_to, worker_ip,
       start_time, end_time, input_data, output_data, error_message, retry_count
FROM workflow_node_states
WHERE instance_id = @instanceID AND status = 'failed'`

	rows, err := r.db.QueryContext(ctx, q, sql.Named("instanceID", instanceID))
	if err != nil {
		return nil, wrapDBErr(err, "GetFailedNodes")
	}
	defer rows.Close()

	var states []*types.WorkflowNodeState
	for rows.Next() {
		state, err := scanNodeStateRow(rows)
		if err != nil {
			return nil, err
		}
		states = append(states, state)
	}
	return states, rows.Err()
}

// ─────────────────────────────────────────────────────────────────────────────
// D2 新增：版本管理
// ─────────────────────────────────────────────────────────────────────────────

// CreateWorkflowVersion 创建新版本记录
func (r *Repository) CreateWorkflowVersion(ver *types.WorkflowVersion) error {
	defJSON, err := json.Marshal(ver.Definition)
	if err != nil {
		return fmt.Errorf("marshal version definition: %w", err)
	}
	var grayCfgJSON string
	if ver.GrayConfig != nil {
		b, err := json.Marshal(ver.GrayConfig)
		if err != nil {
			return fmt.Errorf("marshal gray_config: %w", err)
		}
		grayCfgJSON = string(b)
	}
	ctx, cancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
	defer cancel()

	const q = `
INSERT INTO workflow_versions (workflow_id, version, definition, change_log, created_by, created_at, is_current, gray_config)
VALUES (@workflowID, @version, @def, @changeLog, @createdBy, @createdAt, @isCurrent, @grayConfig)`

	_, err = r.db.ExecContext(ctx, q,
		sql.Named("workflowID", ver.WorkflowID),
		sql.Named("version", ver.Version),
		sql.Named("def", string(defJSON)),
		sql.Named("changeLog", ver.ChangeLog),
		sql.Named("createdBy", ver.CreatedBy),
		sql.Named("createdAt", ver.CreatedAt),
		sql.Named("isCurrent", ver.IsCurrent),
		sql.Named("grayConfig", nullString(grayCfgJSON)),
	)
	return wrapDBErr(err, "CreateWorkflowVersion")
}

// GetWorkflowVersion 查询指定版本
func (r *Repository) GetWorkflowVersion(workflowID string, version int) (*types.WorkflowVersion, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
	defer cancel()

	const q = `
SELECT id, workflow_id, version, definition, change_log, created_by, created_at, is_current, gray_config
FROM workflow_versions WHERE workflow_id = @workflowID AND version = @version`

	row := r.db.QueryRowContext(ctx, q,
		sql.Named("workflowID", workflowID),
		sql.Named("version", version),
	)
	return scanWorkflowVersion(row)
}

// GetCurrentWorkflowVersion 查询当前生效版本
func (r *Repository) GetCurrentWorkflowVersion(workflowID string) (*types.WorkflowVersion, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
	defer cancel()

	const q = `
SELECT id, workflow_id, version, definition, change_log, created_by, created_at, is_current, gray_config
FROM workflow_versions WHERE workflow_id = @workflowID AND is_current = 1`

	row := r.db.QueryRowContext(ctx, q, sql.Named("workflowID", workflowID))
	return scanWorkflowVersion(row)
}

// ListWorkflowVersions 列举所有版本
func (r *Repository) ListWorkflowVersions(workflowID string) ([]*types.WorkflowVersion, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
	defer cancel()

	const q = `
SELECT id, workflow_id, version, definition, change_log, created_by, created_at, is_current, gray_config
FROM workflow_versions WHERE workflow_id = @workflowID ORDER BY version ASC`

	rows, err := r.db.QueryContext(ctx, q, sql.Named("workflowID", workflowID))
	if err != nil {
		return nil, wrapDBErr(err, "ListWorkflowVersions")
	}
	defer rows.Close()

	var vers []*types.WorkflowVersion
	for rows.Next() {
		v, err := scanWorkflowVersionRow(rows)
		if err != nil {
			return nil, err
		}
		vers = append(vers, v)
	}
	return vers, rows.Err()
}

// SetCurrentVersion 将指定版本设为当前生效版本（事务：先清除所有 is_current，再设置目标）
func (r *Repository) SetCurrentVersion(workflowID string, version int) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
	defer cancel()

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return wrapDBErr(err, "SetCurrentVersion begin tx")
	}

	if _, err = tx.ExecContext(ctx,
		`UPDATE workflow_versions SET is_current = 0 WHERE workflow_id = @workflowID`,
		sql.Named("workflowID", workflowID),
	); err != nil {
		_ = tx.Rollback()
		return wrapDBErr(err, "SetCurrentVersion clear current")
	}
	if _, err = tx.ExecContext(ctx,
		`UPDATE workflow_versions SET is_current = 1 WHERE workflow_id = @workflowID AND version = @version`,
		sql.Named("workflowID", workflowID),
		sql.Named("version", version),
	); err != nil {
		_ = tx.Rollback()
		return wrapDBErr(err, "SetCurrentVersion set version")
	}
	if _, err = tx.ExecContext(ctx,
		`UPDATE workflow_defs SET current_version = @version WHERE id = @workflowID`,
		sql.Named("version", version),
		sql.Named("workflowID", workflowID),
	); err != nil {
		_ = tx.Rollback()
		return wrapDBErr(err, "SetCurrentVersion update defs")
	}
	return wrapDBErr(tx.Commit(), "SetCurrentVersion commit")
}

// UpdateVersionGrayConfig 更新版本灰度配置
func (r *Repository) UpdateVersionGrayConfig(workflowID string, version int, cfg *types.GrayReleaseConfig) error {
	var grayCfgJSON string
	if cfg != nil {
		b, err := json.Marshal(cfg)
		if err != nil {
			return fmt.Errorf("marshal gray_config: %w", err)
		}
		grayCfgJSON = string(b)
	}
	ctx, cancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
	defer cancel()

	const q = `
UPDATE workflow_versions SET gray_config = @grayConfig
WHERE workflow_id = @workflowID AND version = @version`

	_, err := r.db.ExecContext(ctx, q,
		sql.Named("grayConfig", nullString(grayCfgJSON)),
		sql.Named("workflowID", workflowID),
		sql.Named("version", version),
	)
	return wrapDBErr(err, "UpdateVersionGrayConfig")
}

// ─────────────────────────────────────────────────────────────────────────────
// D2 新增：死信队列
// ─────────────────────────────────────────────────────────────────────────────

// CreateDeadLetterTask 将任务入死信队列
func (r *Repository) CreateDeadLetterTask(task *types.DeadLetterTask) error {
	taskDataJSON, err := json.Marshal(task.TaskData)
	if err != nil {
		return fmt.Errorf("marshal task_data: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
	defer cancel()

	const q = `
INSERT INTO workflow_dead_letter_tasks
    (id, instance_id, node_id, worker_id, worker_ip, error_message, error_code, retry_count, task_data, created_at, resend_count)
VALUES (@id, @instanceID, @nodeID, @workerID, @workerIP, @errMsg, @errCode, @retryCount, @taskData, @createdAt, 0)`

	_, err = r.db.ExecContext(ctx, q,
		sql.Named("id", task.ID),
		sql.Named("instanceID", task.InstanceID),
		sql.Named("nodeID", task.NodeID),
		sql.Named("workerID", nullString(task.WorkerID)),
		sql.Named("workerIP", nullString(task.WorkerIP)),
		sql.Named("errMsg", task.Error),
		sql.Named("errCode", nullString(task.ErrorCode)),
		sql.Named("retryCount", task.RetryCount),
		sql.Named("taskData", string(taskDataJSON)),
		sql.Named("createdAt", task.CreatedAt),
	)
	return wrapDBErr(err, "CreateDeadLetterTask")
}

// GetDeadLetterTask 获取死信任务
func (r *Repository) GetDeadLetterTask(id string) (*types.DeadLetterTask, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
	defer cancel()

	const q = `
SELECT id, instance_id, node_id, worker_id, worker_ip, error_message, error_code,
       retry_count, task_data, created_at, resend_count, last_resend_at
FROM workflow_dead_letter_tasks WHERE id = @id`

	row := r.db.QueryRowContext(ctx, q, sql.Named("id", id))
	return scanDeadLetterTask(row)
}

// ListDeadLetterTasks 列举实例的死信任务
func (r *Repository) ListDeadLetterTasks(instanceID string) ([]*types.DeadLetterTask, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
	defer cancel()

	const q = `
SELECT id, instance_id, node_id, worker_id, worker_ip, error_message, error_code,
       retry_count, task_data, created_at, resend_count, last_resend_at
FROM workflow_dead_letter_tasks WHERE instance_id = @instanceID ORDER BY created_at ASC`

	rows, err := r.db.QueryContext(ctx, q, sql.Named("instanceID", instanceID))
	if err != nil {
		return nil, wrapDBErr(err, "ListDeadLetterTasks")
	}
	defer rows.Close()

	var tasks []*types.DeadLetterTask
	for rows.Next() {
		t, err := scanDeadLetterTaskRow(rows)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, t)
	}
	return tasks, rows.Err()
}

// UpdateDeadLetterTask 更新死信任务（重发计数等）
func (r *Repository) UpdateDeadLetterTask(task *types.DeadLetterTask) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
	defer cancel()

	const q = `
UPDATE workflow_dead_letter_tasks
SET resend_count = @resendCount, last_resend_at = @lastResendAt
WHERE id = @id`

	var lastResendAt interface{}
	if task.LastResendAt != nil {
		lastResendAt = *task.LastResendAt
	}
	_, err := r.db.ExecContext(ctx, q,
		sql.Named("resendCount", task.ResendCount),
		sql.Named("lastResendAt", lastResendAt),
		sql.Named("id", task.ID),
	)
	return wrapDBErr(err, "UpdateDeadLetterTask")
}

// ─────────────────────────────────────────────────────────────────────────────
// D2 私有扫描函数
// ─────────────────────────────────────────────────────────────────────────────

func scanWorkflowVersion(row *sql.Row) (*types.WorkflowVersion, error) {
	var ver types.WorkflowVersion
	var defJSON, changeLog, createdBy, grayCfg sql.NullString

	err := row.Scan(
		&ver.ID, &ver.WorkflowID, &ver.Version,
		&defJSON, &changeLog, &createdBy, &ver.CreatedAt, &ver.IsCurrent, &grayCfg,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("workflow version not found")
		}
		return nil, wrapDBErr(err, "scanWorkflowVersion")
	}
	if defJSON.Valid && defJSON.String != "" {
		ver.Definition = &types.WorkflowDef{}
		_ = json.Unmarshal([]byte(defJSON.String), ver.Definition)
	}
	if changeLog.Valid {
		ver.ChangeLog = changeLog.String
	}
	if createdBy.Valid {
		ver.CreatedBy = createdBy.String
	}
	if grayCfg.Valid && grayCfg.String != "" {
		ver.GrayConfig = &types.GrayReleaseConfig{}
		_ = json.Unmarshal([]byte(grayCfg.String), ver.GrayConfig)
	}
	return &ver, nil
}

func scanWorkflowVersionRow(rows *sql.Rows) (*types.WorkflowVersion, error) {
	var ver types.WorkflowVersion
	var defJSON, changeLog, createdBy, grayCfg sql.NullString

	err := rows.Scan(
		&ver.ID, &ver.WorkflowID, &ver.Version,
		&defJSON, &changeLog, &createdBy, &ver.CreatedAt, &ver.IsCurrent, &grayCfg,
	)
	if err != nil {
		return nil, wrapDBErr(err, "scanWorkflowVersionRow")
	}
	if defJSON.Valid && defJSON.String != "" {
		ver.Definition = &types.WorkflowDef{}
		_ = json.Unmarshal([]byte(defJSON.String), ver.Definition)
	}
	if changeLog.Valid {
		ver.ChangeLog = changeLog.String
	}
	if createdBy.Valid {
		ver.CreatedBy = createdBy.String
	}
	if grayCfg.Valid && grayCfg.String != "" {
		ver.GrayConfig = &types.GrayReleaseConfig{}
		_ = json.Unmarshal([]byte(grayCfg.String), ver.GrayConfig)
	}
	return &ver, nil
}

func scanDeadLetterTask(row *sql.Row) (*types.DeadLetterTask, error) {
	var t types.DeadLetterTask
	var workerID, workerIP, errCode, taskDataJSON sql.NullString
	var lastResendAt sql.NullTime

	err := row.Scan(
		&t.ID, &t.InstanceID, &t.NodeID,
		&workerID, &workerIP, &t.Error, &errCode,
		&t.RetryCount, &taskDataJSON, &t.CreatedAt, &t.ResendCount, &lastResendAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("dead letter task not found")
		}
		return nil, wrapDBErr(err, "scanDeadLetterTask")
	}
	if workerID.Valid {
		t.WorkerID = workerID.String
	}
	if workerIP.Valid {
		t.WorkerIP = workerIP.String
	}
	if errCode.Valid {
		t.ErrorCode = errCode.String
	}
	if taskDataJSON.Valid && taskDataJSON.String != "" {
		_ = json.Unmarshal([]byte(taskDataJSON.String), &t.TaskData)
	}
	if lastResendAt.Valid {
		t.LastResendAt = &lastResendAt.Time
	}
	return &t, nil
}

func scanDeadLetterTaskRow(rows *sql.Rows) (*types.DeadLetterTask, error) {
	var t types.DeadLetterTask
	var workerID, workerIP, errCode, taskDataJSON sql.NullString
	var lastResendAt sql.NullTime

	err := rows.Scan(
		&t.ID, &t.InstanceID, &t.NodeID,
		&workerID, &workerIP, &t.Error, &errCode,
		&t.RetryCount, &taskDataJSON, &t.CreatedAt, &t.ResendCount, &lastResendAt,
	)
	if err != nil {
		return nil, wrapDBErr(err, "scanDeadLetterTaskRow")
	}
	if workerID.Valid {
		t.WorkerID = workerID.String
	}
	if workerIP.Valid {
		t.WorkerIP = workerIP.String
	}
	if errCode.Valid {
		t.ErrorCode = errCode.String
	}
	if taskDataJSON.Valid && taskDataJSON.String != "" {
		_ = json.Unmarshal([]byte(taskDataJSON.String), &t.TaskData)
	}
	if lastResendAt.Valid {
		t.LastResendAt = &lastResendAt.Time
	}
	return &t, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// D3: 审计日志
// ─────────────────────────────────────────────────────────────────────────────

// CreateAuditLog 创建单条审计日志
func (r *Repository) CreateAuditLog(auditLog *types.WorkflowAuditLog) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
	defer cancel()

	var beforeJSON, afterJSON []byte
	if auditLog.BeforeData != nil {
		beforeJSON, _ = json.Marshal(auditLog.BeforeData)
	}
	if auditLog.AfterData != nil {
		afterJSON, _ = json.Marshal(auditLog.AfterData)
	}

	const q = `
INSERT INTO workflow_instance_logs
(id, instance_id, node_id, workflow_id, endpoint_id, operation_type,
 operator, operate_ip, operate_time, before_data, after_data, detail, trace_id, hash)
VALUES
(@id, @instanceId, @nodeId, @workflowId, @endpointId, @operationType,
 @operator, @operateIp, @operateTime, @beforeData, @afterData, @detail, @traceId, @hash)`

	_, err := r.db.ExecContext(ctx, q,
		sql.Named("id", auditLog.ID),
		sql.Named("instanceId", auditLog.InstanceID),
		sql.Named("nodeId", auditLog.NodeID),
		sql.Named("workflowId", auditLog.WorkflowID),
		sql.Named("endpointId", auditLog.EndpointID),
		sql.Named("operationType", string(auditLog.OperationType)),
		sql.Named("operator", auditLog.Operator),
		sql.Named("operateIp", auditLog.OperateIP),
		sql.Named("operateTime", auditLog.OperateTime),
		sql.Named("beforeData", nullStr(string(beforeJSON))),
		sql.Named("afterData", nullStr(string(afterJSON))),
		sql.Named("detail", auditLog.Detail),
		sql.Named("traceId", auditLog.TraceID),
		sql.Named("hash", auditLog.Hash),
	)
	return wrapDBErr(err, "CreateAuditLog")
}

// BatchCreateAuditLogs 批量创建审计日志
func (r *Repository) BatchCreateAuditLogs(logs []*types.WorkflowAuditLog) error {
	if len(logs) == 0 {
		return nil
	}
	for _, l := range logs {
		if err := r.CreateAuditLog(l); err != nil {
			return err
		}
	}
	return nil
}

// QueryAuditLogs 分页查询审计日志
func (r *Repository) QueryAuditLogs(filter types.AuditLogFilter, page, pageSize int) ([]*types.WorkflowAuditLog, int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
	defer cancel()

	where := " WHERE 1=1"
	args := []interface{}{}

	if filter.WorkflowID != "" {
		where += " AND workflow_id = @workflowId"
		args = append(args, sql.Named("workflowId", filter.WorkflowID))
	}
	if filter.InstanceID != "" {
		where += " AND instance_id = @instanceId"
		args = append(args, sql.Named("instanceId", filter.InstanceID))
	}
	if filter.NodeID != "" {
		where += " AND node_id = @nodeId"
		args = append(args, sql.Named("nodeId", filter.NodeID))
	}
	if filter.EndpointID != "" {
		where += " AND endpoint_id = @endpointId"
		args = append(args, sql.Named("endpointId", filter.EndpointID))
	}
	if filter.Operator != "" {
		where += " AND operator = @operator"
		args = append(args, sql.Named("operator", filter.Operator))
	}
	if filter.StartTime != nil {
		where += " AND operate_time >= @startTime"
		args = append(args, sql.Named("startTime", *filter.StartTime))
	}
	if filter.EndTime != nil {
		where += " AND operate_time <= @endTime"
		args = append(args, sql.Named("endTime", *filter.EndTime))
	}

	countQ := "SELECT COUNT(*) FROM workflow_instance_logs" + where
	var total int64
	if err := r.db.QueryRowContext(ctx, countQ, args...).Scan(&total); err != nil {
		return nil, 0, wrapDBErr(err, "QueryAuditLogs.count")
	}

	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	dataQ := "SELECT id, instance_id, node_id, workflow_id, endpoint_id, operation_type, " +
		"operator, operate_ip, operate_time, before_data, after_data, detail, trace_id, hash " +
		"FROM workflow_instance_logs" + where +
		" ORDER BY operate_time DESC OFFSET @offset ROWS FETCH NEXT @pageSize ROWS ONLY"
	args = append(args, sql.Named("offset", offset), sql.Named("pageSize", pageSize))

	rows, err := r.db.QueryContext(ctx, dataQ, args...)
	if err != nil {
		return nil, 0, wrapDBErr(err, "QueryAuditLogs.query")
	}
	defer rows.Close()

	var result []*types.WorkflowAuditLog
	for rows.Next() {
		l, scanErr := scanAuditLogRow(rows)
		if scanErr != nil {
			return nil, 0, scanErr
		}
		result = append(result, l)
	}
	return result, total, rows.Err()
}

// GetAuditLog 获取单条审计日志
func (r *Repository) GetAuditLog(logID string) (*types.WorkflowAuditLog, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
	defer cancel()

	const q = `
SELECT id, instance_id, node_id, workflow_id, endpoint_id, operation_type,
       operator, operate_ip, operate_time, before_data, after_data, detail, trace_id, hash
FROM workflow_instance_logs WHERE id = @id`

	row := r.db.QueryRowContext(ctx, q, sql.Named("id", logID))
	return scanAuditLog(row)
}

// GetLatestAuditLog 获取最新一条审计日志（用于哈希链初始化）
func (r *Repository) GetLatestAuditLog() (*types.WorkflowAuditLog, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
	defer cancel()

	const q = `
SELECT TOP 1 id, instance_id, node_id, workflow_id, endpoint_id, operation_type,
       operator, operate_ip, operate_time, before_data, after_data, detail, trace_id, hash
FROM workflow_instance_logs
ORDER BY operate_time DESC`

	row := r.db.QueryRowContext(ctx, q)
	l, err := scanAuditLog(row)
	if err != nil {
		// 无记录时返回 nil 而不是错误
		if err.Error() == "audit log not found" {
			return nil, nil
		}
		return nil, err
	}
	return l, nil
}

// ArchiveAuditLogs 归档（物理删除）指定时间之前的审计日志
func (r *Repository) ArchiveAuditLogs(beforeTime time.Time) error {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	const q = `DELETE FROM workflow_instance_logs WHERE operate_time < @beforeTime`
	_, err := r.db.ExecContext(ctx, q, sql.Named("beforeTime", beforeTime))
	return wrapDBErr(err, "ArchiveAuditLogs")
}

// ─────────────────────────────────────────────────────────────────────────────
// D3: 审批记录
// ─────────────────────────────────────────────────────────────────────────────

// CreateApprovalRecord 创建审批记录
func (r *Repository) CreateApprovalRecord(record *types.ApprovalRecord) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
	defer cancel()

	var formJSON []byte
	if record.FormData != nil {
		formJSON, _ = json.Marshal(record.FormData)
	}

	const q = `
INSERT INTO workflow_approval_records
(id, instance_id, node_id, approver, action, comment, form_data, operate_time, operate_ip)
VALUES
(@id, @instanceId, @nodeId, @approver, @action, @comment, @formData, @operateTime, @operateIp)`

	_, err := r.db.ExecContext(ctx, q,
		sql.Named("id", record.ID),
		sql.Named("instanceId", record.InstanceID),
		sql.Named("nodeId", record.NodeID),
		sql.Named("approver", record.Approver),
		sql.Named("action", record.Action),
		sql.Named("comment", record.Comment),
		sql.Named("formData", nullStr(string(formJSON))),
		sql.Named("operateTime", record.OperateTime),
		sql.Named("operateIp", record.OperateIP),
	)
	return wrapDBErr(err, "CreateApprovalRecord")
}

// GetApprovalRecords 获取节点审批记录列表
func (r *Repository) GetApprovalRecords(instanceID, nodeID string) ([]*types.ApprovalRecord, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
	defer cancel()

	const q = `
SELECT id, instance_id, node_id, approver, action, comment, form_data, operate_time, operate_ip
FROM workflow_approval_records
WHERE instance_id = @instanceId AND node_id = @nodeId
ORDER BY operate_time ASC`

	rows, err := r.db.QueryContext(ctx, q,
		sql.Named("instanceId", instanceID),
		sql.Named("nodeId", nodeID),
	)
	if err != nil {
		return nil, wrapDBErr(err, "GetApprovalRecords")
	}
	defer rows.Close()

	var records []*types.ApprovalRecord
	for rows.Next() {
		rec, scanErr := scanApprovalRecordRow(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		records = append(records, rec)
	}
	return records, rows.Err()
}

// GetPendingApprovalNodeStates 查询待审批节点状态
func (r *Repository) GetPendingApprovalNodeStates(filter types.ApprovalTaskFilter) ([]*types.WorkflowNodeState, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
	defer cancel()

	where := " WHERE ns.approval_status = 'pending' AND ns.status = 'waiting'"
	args := []interface{}{}

	if filter.InstanceID != "" {
		where += " AND ns.instance_id = @instanceId"
		args = append(args, sql.Named("instanceId", filter.InstanceID))
	}
	if filter.NodeID != "" {
		where += " AND ns.node_id = @nodeId"
		args = append(args, sql.Named("nodeId", filter.NodeID))
	}

	q := "SELECT ns.id, ns.instance_id, ns.node_id, ns.status, ns.assigned_to, ns.worker_ip, " +
		"ns.start_time, ns.end_time, ns.input_data, ns.output_data, ns.error_message, " +
		"ns.retry_count, ns.next_retry_time, ns.last_retry_time, ns.error_code, ns.skipped_reason, " +
		"ns.approval_status, ns.current_approver_index " +
		"FROM workflow_node_states ns" + where + " ORDER BY ns.id ASC"

	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, wrapDBErr(err, "GetPendingApprovalNodeStates")
	}
	defer rows.Close()

	var states []*types.WorkflowNodeState
	for rows.Next() {
		ns, scanErr := scanNodeStateRowD3(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		states = append(states, ns)
	}
	return states, rows.Err()
}

// ─────────────────────────────────────────────────────────────────────────────
// D3: 端点管理
// ─────────────────────────────────────────────────────────────────────────────

// CreateEndpoint 创建端点
func (r *Repository) CreateEndpoint(endpoint *types.WorkflowEndpoint) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
	defer cancel()

	var cfgJSON, schedCfgJSON []byte
	if endpoint.Config != nil {
		cfgJSON, _ = json.Marshal(endpoint.Config)
	}
	if endpoint.ScheduleConfig != nil {
		schedCfgJSON, _ = json.Marshal(endpoint.ScheduleConfig)
	}

	const q = `
INSERT INTO workflow_endpoints
(id, name, type, workflow_id, config, schedule_config, path, created_by, disabled, created_at, updated_at)
VALUES
(@id, @name, @type, @workflowId, @config, @scheduleConfig, @path, @createdBy, @disabled, @createdAt, @updatedAt)`

	_, err := r.db.ExecContext(ctx, q,
		sql.Named("id", endpoint.ID),
		sql.Named("name", endpoint.Name),
		sql.Named("type", string(endpoint.Type)),
		sql.Named("workflowId", endpoint.WorkflowID),
		sql.Named("config", nullStr(string(cfgJSON))),
		sql.Named("scheduleConfig", nullStr(string(schedCfgJSON))),
		sql.Named("path", endpoint.Path),
		sql.Named("createdBy", endpoint.CreatedBy),
		sql.Named("disabled", endpoint.Disabled),
		sql.Named("createdAt", endpoint.CreatedAt),
		sql.Named("updatedAt", endpoint.UpdatedAt),
	)
	return wrapDBErr(err, "CreateEndpoint")
}

// GetEndpoint 获取端点
func (r *Repository) GetEndpoint(endpointID string) (*types.WorkflowEndpoint, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
	defer cancel()

	const q = `
SELECT id, name, type, workflow_id, config, schedule_config, path, created_by, disabled, created_at, updated_at
FROM workflow_endpoints WHERE id = @id`

	row := r.db.QueryRowContext(ctx, q, sql.Named("id", endpointID))
	return scanEndpoint(row)
}

// UpdateEndpoint 更新端点
func (r *Repository) UpdateEndpoint(endpoint *types.WorkflowEndpoint) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
	defer cancel()

	var cfgJSON, schedCfgJSON []byte
	if endpoint.Config != nil {
		cfgJSON, _ = json.Marshal(endpoint.Config)
	}
	if endpoint.ScheduleConfig != nil {
		schedCfgJSON, _ = json.Marshal(endpoint.ScheduleConfig)
	}

	const q = `
UPDATE workflow_endpoints
SET name = @name, config = @config, schedule_config = @scheduleConfig,
    path = @path, disabled = @disabled, updated_at = @updatedAt
WHERE id = @id`

	_, err := r.db.ExecContext(ctx, q,
		sql.Named("name", endpoint.Name),
		sql.Named("config", nullStr(string(cfgJSON))),
		sql.Named("scheduleConfig", nullStr(string(schedCfgJSON))),
		sql.Named("path", endpoint.Path),
		sql.Named("disabled", endpoint.Disabled),
		sql.Named("updatedAt", endpoint.UpdatedAt),
		sql.Named("id", endpoint.ID),
	)
	return wrapDBErr(err, "UpdateEndpoint")
}

// DeleteEndpoint 删除端点
func (r *Repository) DeleteEndpoint(endpointID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
	defer cancel()

	const q = `DELETE FROM workflow_endpoints WHERE id = @id`
	_, err := r.db.ExecContext(ctx, q, sql.Named("id", endpointID))
	return wrapDBErr(err, "DeleteEndpoint")
}

// ListEndpoints 查询端点列表
func (r *Repository) ListEndpoints(filter types.EndpointFilter, page, pageSize int) ([]*types.WorkflowEndpoint, int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
	defer cancel()

	where := " WHERE 1=1"
	args := []interface{}{}

	if filter.WorkflowID != "" {
		where += " AND workflow_id = @workflowId"
		args = append(args, sql.Named("workflowId", filter.WorkflowID))
	}
	if string(filter.Type) != "" {
		where += " AND type = @type"
		args = append(args, sql.Named("type", string(filter.Type)))
	}
	if filter.Disabled != nil {
		where += " AND disabled = @disabled"
		args = append(args, sql.Named("disabled", *filter.Disabled))
	}

	var total int64
	countQ := "SELECT COUNT(*) FROM workflow_endpoints" + where
	if err := r.db.QueryRowContext(ctx, countQ, args...).Scan(&total); err != nil {
		return nil, 0, wrapDBErr(err, "ListEndpoints.count")
	}

	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	dataQ := "SELECT id, name, type, workflow_id, config, schedule_config, path, created_by, disabled, created_at, updated_at " +
		"FROM workflow_endpoints" + where +
		" ORDER BY created_at DESC OFFSET @offset ROWS FETCH NEXT @pageSize ROWS ONLY"
	args = append(args, sql.Named("offset", offset), sql.Named("pageSize", pageSize))

	rows, err := r.db.QueryContext(ctx, dataQ, args...)
	if err != nil {
		return nil, 0, wrapDBErr(err, "ListEndpoints.query")
	}
	defer rows.Close()

	var endpoints []*types.WorkflowEndpoint
	for rows.Next() {
		ep, scanErr := scanEndpointRow(rows)
		if scanErr != nil {
			return nil, 0, scanErr
		}
		endpoints = append(endpoints, ep)
	}
	return endpoints, total, rows.Err()
}

// UpdateEndpointTriggeredCount 更新端点触发计数
func (r *Repository) UpdateEndpointTriggeredCount(endpointID string, count int) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
	defer cancel()

	const q = `
UPDATE workflow_endpoints
SET schedule_config = JSON_MODIFY(schedule_config, '$.triggeredCount', @count), updated_at = @updatedAt
WHERE id = @id`

	_, err := r.db.ExecContext(ctx, q,
		sql.Named("count", count),
		sql.Named("updatedAt", time.Now().UTC()),
		sql.Named("id", endpointID),
	)
	return wrapDBErr(err, "UpdateEndpointTriggeredCount")
}

// ─────────────────────────────────────────────────────────────────────────────
// D3 私有扫描函数
// ─────────────────────────────────────────────────────────────────────────────

func scanAuditLog(row *sql.Row) (*types.WorkflowAuditLog, error) {
	var l types.WorkflowAuditLog
	var beforeJSON, afterJSON, hash sql.NullString

	err := row.Scan(
		&l.ID, &l.InstanceID, &l.NodeID, &l.WorkflowID, &l.EndpointID,
		&l.OperationType, &l.Operator, &l.OperateIP, &l.OperateTime,
		&beforeJSON, &afterJSON, &l.Detail, &l.TraceID, &hash,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("audit log not found")
		}
		return nil, wrapDBErr(err, "scanAuditLog")
	}
	if beforeJSON.Valid && beforeJSON.String != "" {
		_ = json.Unmarshal([]byte(beforeJSON.String), &l.BeforeData)
	}
	if afterJSON.Valid && afterJSON.String != "" {
		_ = json.Unmarshal([]byte(afterJSON.String), &l.AfterData)
	}
	if hash.Valid {
		l.Hash = hash.String
	}
	return &l, nil
}

func scanAuditLogRow(rows *sql.Rows) (*types.WorkflowAuditLog, error) {
	var l types.WorkflowAuditLog
	var beforeJSON, afterJSON, hash sql.NullString

	err := rows.Scan(
		&l.ID, &l.InstanceID, &l.NodeID, &l.WorkflowID, &l.EndpointID,
		&l.OperationType, &l.Operator, &l.OperateIP, &l.OperateTime,
		&beforeJSON, &afterJSON, &l.Detail, &l.TraceID, &hash,
	)
	if err != nil {
		return nil, wrapDBErr(err, "scanAuditLogRow")
	}
	if beforeJSON.Valid && beforeJSON.String != "" {
		_ = json.Unmarshal([]byte(beforeJSON.String), &l.BeforeData)
	}
	if afterJSON.Valid && afterJSON.String != "" {
		_ = json.Unmarshal([]byte(afterJSON.String), &l.AfterData)
	}
	if hash.Valid {
		l.Hash = hash.String
	}
	return &l, nil
}

func scanApprovalRecordRow(rows *sql.Rows) (*types.ApprovalRecord, error) {
	var rec types.ApprovalRecord
	var formJSON sql.NullString

	err := rows.Scan(
		&rec.ID, &rec.InstanceID, &rec.NodeID,
		&rec.Approver, &rec.Action, &rec.Comment,
		&formJSON, &rec.OperateTime, &rec.OperateIP,
	)
	if err != nil {
		return nil, wrapDBErr(err, "scanApprovalRecordRow")
	}
	if formJSON.Valid && formJSON.String != "" {
		_ = json.Unmarshal([]byte(formJSON.String), &rec.FormData)
	}
	return &rec, nil
}

func scanNodeStateRowD3(rows *sql.Rows) (*types.WorkflowNodeState, error) {
	var ns types.WorkflowNodeState
	var assignedTo, workerIP, errMsg, inputJSON, outputJSON, errCode, skippedReason, approvalStatus sql.NullString
	var startTime, endTime, nextRetryTime, lastRetryTime sql.NullTime
	var currentApproverIndex sql.NullInt32

	err := rows.Scan(
		&ns.ID, &ns.InstanceID, &ns.NodeID, &ns.Status, &assignedTo, &workerIP,
		&startTime, &endTime, &inputJSON, &outputJSON, &errMsg,
		&ns.RetryCount, &nextRetryTime, &lastRetryTime, &errCode, &skippedReason,
		&approvalStatus, &currentApproverIndex,
	)
	if err != nil {
		return nil, wrapDBErr(err, "scanNodeStateRowD3")
	}
	if assignedTo.Valid {
		ns.AssignedTo = assignedTo.String
	}
	if workerIP.Valid {
		ns.WorkerIP = workerIP.String
	}
	if errMsg.Valid {
		ns.ErrorMessage = errMsg.String
	}
	if inputJSON.Valid && inputJSON.String != "" {
		_ = json.Unmarshal([]byte(inputJSON.String), &ns.InputData)
	}
	if outputJSON.Valid && outputJSON.String != "" {
		_ = json.Unmarshal([]byte(outputJSON.String), &ns.OutputData)
	}
	if errCode.Valid {
		ns.ErrorCode = errCode.String
	}
	if skippedReason.Valid {
		ns.SkippedReason = skippedReason.String
	}
	if startTime.Valid {
		ns.StartTime = &startTime.Time
	}
	if endTime.Valid {
		ns.EndTime = &endTime.Time
	}
	if nextRetryTime.Valid {
		ns.NextRetryTime = &nextRetryTime.Time
	}
	if lastRetryTime.Valid {
		ns.LastRetryTime = &lastRetryTime.Time
	}
	if approvalStatus.Valid {
		ns.ApprovalStatus = types.ApprovalStatus(approvalStatus.String)
	}
	if currentApproverIndex.Valid {
		ns.CurrentApproverIndex = int(currentApproverIndex.Int32)
	}
	return &ns, nil
}

func scanEndpoint(row *sql.Row) (*types.WorkflowEndpoint, error) {
	var ep types.WorkflowEndpoint
	var cfgJSON, schedCfgJSON sql.NullString

	err := row.Scan(
		&ep.ID, &ep.Name, &ep.Type, &ep.WorkflowID,
		&cfgJSON, &schedCfgJSON, &ep.Path, &ep.CreatedBy,
		&ep.Disabled, &ep.CreatedAt, &ep.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("endpoint not found")
		}
		return nil, wrapDBErr(err, "scanEndpoint")
	}
	if cfgJSON.Valid && cfgJSON.String != "" {
		_ = json.Unmarshal([]byte(cfgJSON.String), &ep.Config)
	}
	if schedCfgJSON.Valid && schedCfgJSON.String != "" {
		ep.ScheduleConfig = &types.ScheduleEndpointConfig{}
		_ = json.Unmarshal([]byte(schedCfgJSON.String), ep.ScheduleConfig)
	}
	return &ep, nil
}

func scanEndpointRow(rows *sql.Rows) (*types.WorkflowEndpoint, error) {
	var ep types.WorkflowEndpoint
	var cfgJSON, schedCfgJSON sql.NullString

	err := rows.Scan(
		&ep.ID, &ep.Name, &ep.Type, &ep.WorkflowID,
		&cfgJSON, &schedCfgJSON, &ep.Path, &ep.CreatedBy,
		&ep.Disabled, &ep.CreatedAt, &ep.UpdatedAt,
	)
	if err != nil {
		return nil, wrapDBErr(err, "scanEndpointRow")
	}
	if cfgJSON.Valid && cfgJSON.String != "" {
		_ = json.Unmarshal([]byte(cfgJSON.String), &ep.Config)
	}
	if schedCfgJSON.Valid && schedCfgJSON.String != "" {
		ep.ScheduleConfig = &types.ScheduleEndpointConfig{}
		_ = json.Unmarshal([]byte(schedCfgJSON.String), ep.ScheduleConfig)
	}
	return &ep, nil
}
