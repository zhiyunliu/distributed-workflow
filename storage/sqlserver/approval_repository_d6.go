package sqlserver

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/zhiyunliu/distributed-workflow/interfaces"
	"github.com/zhiyunliu/distributed-workflow/types"
)

// 编译时接口断言
var _ interfaces.AdvancedApprovalRepository = (*AdvancedApprovalRepository)(nil)

// AdvancedApprovalRepository SQL Server 高级审批数据仓库实现
type AdvancedApprovalRepository struct {
	db *sql.DB
}

// NewAdvancedApprovalRepository 创建高级审批数据仓库实例
func NewAdvancedApprovalRepository(db *sql.DB) *AdvancedApprovalRepository {
	return &AdvancedApprovalRepository{db: db}
}

// CreateDelegate 创建审批委托配置
func (r *AdvancedApprovalRepository) CreateDelegate(ctx context.Context, cfg *types.DelegateConfig) error {
	ctx, cancel := context.WithTimeout(ctx, defaultQueryTimeout)
	defer cancel()

	const q = `
INSERT INTO approval_delegate (delegator_id, agent_id, start_time, end_time, status, created_by, created_at, updated_at)
VALUES (@delegatorID, @agentID, @startTime, @endTime, 1, @createdBy, GETDATE(), GETDATE())`

	_, err := r.db.ExecContext(ctx, q,
		sql.Named("delegatorID", cfg.DelegatorID),
		sql.Named("agentID", cfg.AgentID),
		sql.Named("startTime", cfg.StartTime),
		sql.Named("endTime", cfg.EndTime),
		sql.Named("createdBy", cfg.DelegatorID),
	)
	return wrapDBErr(err, "CreateDelegate")
}

// GetActiveDelegateByDelegator 查询用户当前有效的委托（无结果返回 nil, nil）
func (r *AdvancedApprovalRepository) GetActiveDelegateByDelegator(ctx context.Context, delegatorID string) (*types.DelegateConfig, error) {
	ctx, cancel := context.WithTimeout(ctx, defaultQueryTimeout)
	defer cancel()

	const q = `
SELECT id, delegator_id, agent_id, start_time, end_time, status
FROM approval_delegate
WHERE delegator_id = @delegatorID
  AND status = 1
  AND start_time <= GETDATE()
  AND end_time >= GETDATE()`

	row := r.db.QueryRowContext(ctx, q, sql.Named("delegatorID", delegatorID))

	cfg := &types.DelegateConfig{}
	err := row.Scan(
		&cfg.ID,
		&cfg.DelegatorID,
		&cfg.AgentID,
		&cfg.StartTime,
		&cfg.EndTime,
		&cfg.Status,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, wrapDBErr(err, "GetActiveDelegateByDelegator")
	}
	return cfg, nil
}

// GetDelegatesByDelegator 查询用户所有委托记录，按创建时间倒序
func (r *AdvancedApprovalRepository) GetDelegatesByDelegator(ctx context.Context, delegatorID string) ([]*types.DelegateConfig, error) {
	ctx, cancel := context.WithTimeout(ctx, defaultQueryTimeout)
	defer cancel()

	const q = `
SELECT id, delegator_id, agent_id, start_time, end_time, status, created_at, updated_at
FROM approval_delegate
WHERE delegator_id = @delegatorID
ORDER BY created_at DESC`

	rows, err := r.db.QueryContext(ctx, q, sql.Named("delegatorID", delegatorID))
	if err != nil {
		return nil, wrapDBErr(err, "GetDelegatesByDelegator")
	}
	defer rows.Close()

	var results []*types.DelegateConfig
	for rows.Next() {
		cfg := &types.DelegateConfig{}
		var createdAt, updatedAt sql.NullTime
		if err := rows.Scan(
			&cfg.ID,
			&cfg.DelegatorID,
			&cfg.AgentID,
			&cfg.StartTime,
			&cfg.EndTime,
			&cfg.Status,
			&createdAt,
			&updatedAt,
		); err != nil {
			return nil, wrapDBErr(err, "GetDelegatesByDelegator scan")
		}
		results = append(results, cfg)
	}
	if err := rows.Err(); err != nil {
		return nil, wrapDBErr(err, "GetDelegatesByDelegator rows")
	}
	return results, nil
}

// CreateCC 创建抄送记录
func (r *AdvancedApprovalRepository) CreateCC(ctx context.Context, record *types.CCRecord) error {
	ctx, cancel := context.WithTimeout(ctx, defaultQueryTimeout)
	defer cancel()

	const q = `
INSERT INTO approval_cc (workflow_instance_id, node_id, cc_user_id, cc_time, is_read)
VALUES (@instanceID, @nodeID, @ccUserID, GETDATE(), 0)`

	_, err := r.db.ExecContext(ctx, q,
		sql.Named("instanceID", record.WorkflowInstanceID),
		sql.Named("nodeID", record.NodeID),
		sql.Named("ccUserID", record.CCUserID),
	)
	return wrapDBErr(err, "CreateCC")
}

// GetCCList 分页查询抄送给指定用户的列表，返回记录列表和总数
func (r *AdvancedApprovalRepository) GetCCList(ctx context.Context, userID string, page, pageSize int) ([]*types.CCRecord, int64, error) {
	ctx, cancel := context.WithTimeout(ctx, defaultQueryTimeout)
	defer cancel()

	const countQ = `SELECT COUNT(1) FROM approval_cc WHERE cc_user_id = @userID`
	row := r.db.QueryRowContext(ctx, countQ, sql.Named("userID", userID))
	var total int64
	if err := row.Scan(&total); err != nil {
		return nil, 0, wrapDBErr(err, "GetCCList count")
	}
	if total == 0 {
		return []*types.CCRecord{}, 0, nil
	}

	offset := (page - 1) * pageSize
	const listQ = `
SELECT id, workflow_instance_id, node_id, cc_user_id, cc_time, is_read, read_time
FROM approval_cc
WHERE cc_user_id = @userID
ORDER BY cc_time DESC
OFFSET @offset ROWS FETCH NEXT @pageSize ROWS ONLY`

	rows, err := r.db.QueryContext(ctx, listQ,
		sql.Named("userID", userID),
		sql.Named("offset", offset),
		sql.Named("pageSize", pageSize),
	)
	if err != nil {
		return nil, 0, wrapDBErr(err, "GetCCList query")
	}
	defer rows.Close()

	var records []*types.CCRecord
	for rows.Next() {
		rec := &types.CCRecord{}
		var isReadInt int
		var readTime sql.NullTime
		if err := rows.Scan(
			&rec.ID,
			&rec.WorkflowInstanceID,
			&rec.NodeID,
			&rec.CCUserID,
			&rec.CCTime,
			&isReadInt,
			&readTime,
		); err != nil {
			return nil, 0, wrapDBErr(err, "GetCCList scan")
		}
		rec.IsRead = isReadInt == 1
		if readTime.Valid {
			rec.ReadTime = &readTime.Time
		}
		records = append(records, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, wrapDBErr(err, "GetCCList rows")
	}
	return records, total, nil
}

// MarkCCRead 标记抄送记录已读
func (r *AdvancedApprovalRepository) MarkCCRead(ctx context.Context, ccID int64) error {
	ctx, cancel := context.WithTimeout(ctx, defaultQueryTimeout)
	defer cancel()

	const q = `UPDATE approval_cc SET is_read = 1, read_time = GETDATE() WHERE id = @id`
	_, err := r.db.ExecContext(ctx, q, sql.Named("id", ccID))
	return wrapDBErr(err, "MarkCCRead")
}

// CreateAddSignRecord 创建加签记录，sign_users 序列化为 JSON 字符串存储
func (r *AdvancedApprovalRepository) CreateAddSignRecord(ctx context.Context, record *types.AddSignRecord) error {
	ctx, cancel := context.WithTimeout(ctx, defaultQueryTimeout)
	defer cancel()

	signUsersJSON, err := json.Marshal(record.SignUsers)
	if err != nil {
		return fmt.Errorf("marshal sign_users: %w", err)
	}

	const q = `
INSERT INTO approval_add_sign (workflow_instance_id, node_id, operator_id, sign_type, sign_users, created_at)
VALUES (@instanceID, @nodeID, @operatorID, @signType, @signUsers, GETDATE())`

	_, err = r.db.ExecContext(ctx, q,
		sql.Named("instanceID", record.WorkflowInstanceID),
		sql.Named("nodeID", record.NodeID),
		sql.Named("operatorID", record.OperatorID),
		sql.Named("signType", record.SignType),
		sql.Named("signUsers", string(signUsersJSON)),
	)
	return wrapDBErr(err, "CreateAddSignRecord")
}

// GetAddSignRecords 查询指定实例和节点的加签记录，sign_users 反序列化为 []string
func (r *AdvancedApprovalRepository) GetAddSignRecords(ctx context.Context, instanceID, nodeID string) ([]*types.AddSignRecord, error) {
	ctx, cancel := context.WithTimeout(ctx, defaultQueryTimeout)
	defer cancel()

	const q = `
SELECT id, workflow_instance_id, node_id, operator_id, sign_type, sign_users, created_at
FROM approval_add_sign
WHERE workflow_instance_id = @instanceID AND node_id = @nodeID`

	rows, err := r.db.QueryContext(ctx, q,
		sql.Named("instanceID", instanceID),
		sql.Named("nodeID", nodeID),
	)
	if err != nil {
		return nil, wrapDBErr(err, "GetAddSignRecords")
	}
	defer rows.Close()

	var records []*types.AddSignRecord
	for rows.Next() {
		rec := &types.AddSignRecord{}
		var signUsersJSON string
		if err := rows.Scan(
			&rec.ID,
			&rec.WorkflowInstanceID,
			&rec.NodeID,
			&rec.OperatorID,
			&rec.SignType,
			&signUsersJSON,
			&rec.CreatedAt,
		); err != nil {
			return nil, wrapDBErr(err, "GetAddSignRecords scan")
		}
		if err := json.Unmarshal([]byte(signUsersJSON), &rec.SignUsers); err != nil {
			return nil, fmt.Errorf("unmarshal sign_users for record %d: %w", rec.ID, err)
		}
		records = append(records, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, wrapDBErr(err, "GetAddSignRecords rows")
	}
	return records, nil
}
