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
