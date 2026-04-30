package sqlserver

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/zhiyunliu/distributed-workflow/runtime-execution/pkg/types"
)

type AnalyticsRepository struct {
	db *sql.DB
}

func NewAnalyticsRepository(db *sql.DB) *AnalyticsRepository {
	return &AnalyticsRepository{db: db}
}

func (r *AnalyticsRepository) GetSystemOverview(ctx context.Context) (*types.SystemOverview, error) {
	var overview types.SystemOverview
	query := `
	SELECT 
		(SELECT COUNT(1) FROM workflow_instances)                                                     AS total_insts,
		(SELECT COUNT(1) FROM workflow_instances WHERE start_time >= DATEADD(MONTH, -1, GETDATE()))  AS monthly_new,
		(SELECT COUNT(1) FROM workflow_instances WHERE status = 'running')                           AS active_insts,
		(SELECT COUNT(1) FROM workflow_defs)                                                         AS total_workflows
	`
	err := r.db.QueryRowContext(ctx, query).Scan(
		&overview.TotalInstances,
		&overview.MonthlyNew,
		&overview.ActiveInstances,
		&overview.TotalWorkflows,
	)
	if err != nil {
		return nil, err
	}
	return &overview, nil
}

func (r *AnalyticsRepository) GetWorkflowDailyStats(ctx context.Context, workflowID string, days int) ([]*types.WorkflowDailyStat, error) {
	query := fmt.Sprintf(`
	SELECT stat_date, start_count, complete_count, fail_count, avg_execution_time_ms, max_execution_time_ms, timeout_count, reject_count
	FROM workflow_stats_workflow_daily
	WHERE workflow_id = @p1 AND stat_date >= DATEADD(DAY, @p2, CAST(GETDATE() AS DATE))
	ORDER BY stat_date DESC
	`)
	rows, err := r.db.QueryContext(ctx, query, workflowID, -days)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []*types.WorkflowDailyStat
	for rows.Next() {
		var stat types.WorkflowDailyStat
		err := rows.Scan(
			&stat.StatDate,
			&stat.StartCount,
			&stat.CompleteCount,
			&stat.FailCount,
			&stat.AvgExecutionTimeMs,
			&stat.MaxExecutionTimeMs,
			&stat.TimeoutCount,
			&stat.RejectCount,
		)
		if err != nil {
			return nil, err
		}
		stats = append(stats, &stat)
	}
	return stats, nil
}

// GetWorkflowDailyStatsByDate 查询流程日统计数据（来自 workflow_stats_workflow_daily）
func (r *AnalyticsRepository) GetWorkflowDailyStatsByDate(ctx context.Context, workflowID string, date time.Time) (*types.WorkflowDailyStat, error) {
	query := `
	SELECT workflow_id, stat_date, start_count, complete_count, fail_count, avg_execution_time_ms, max_execution_time_ms, timeout_count, reject_count
	FROM workflow_stats_workflow_daily
	WHERE workflow_id = @p1 AND stat_date = @p2`
	
	var stat types.WorkflowDailyStat
	err := r.db.QueryRowContext(ctx, query, workflowID, date.Format("2006-01-02")).Scan(
		&stat.WorkflowID,
		&stat.StatDate,
		&stat.StartCount,
		&stat.CompleteCount,
		&stat.FailCount,
		&stat.AvgExecutionTimeMs,
		&stat.MaxExecutionTimeMs,
		&stat.TimeoutCount,
		&stat.RejectCount,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &stat, nil
}

func (r *AnalyticsRepository) GetApprovalDailyStats(ctx context.Context, userID string, days int) ([]*types.ApprovalDailyStat, error) {
	query := fmt.Sprintf(`
	SELECT stat_date, approve_count, reject_count, avg_approval_time_ms, timeout_count
	FROM workflow_stats_approval_daily
	WHERE user_id = @p1 AND stat_date >= DATEADD(DAY, @p2, CAST(GETDATE() AS DATE))
	ORDER BY stat_date DESC
	`)
	rows, err := r.db.QueryContext(ctx, query, userID, -days)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []*types.ApprovalDailyStat
	for rows.Next() {
		var stat types.ApprovalDailyStat
		err := rows.Scan(
			&stat.StatDate,
			&stat.ApproveCount,
			&stat.RejectCount,
			&stat.AvgApprovalTimeMs,
			&stat.TimeoutCount,
		)
		if err != nil {
			return nil, err
		}
		stats = append(stats, &stat)
	}
	return stats, nil
}

func (r *AnalyticsRepository) GetTopWorkflows(ctx context.Context, topN int, dateRange string) ([]*types.WorkflowRankItem, error) {
	var dateAddClause string
	switch dateRange {
	case "day":
		dateAddClause = "DATEADD(DAY, -1, GETDATE())"
	case "week":
		dateAddClause = "DATEADD(WEEK, -1, GETDATE())"
	case "month":
		dateAddClause = "DATEADD(MONTH, -1, GETDATE())"
	default:
		dateAddClause = "DATEADD(DAY, -7, GETDATE())"
	}

	query := fmt.Sprintf(`
	SELECT TOP %d wd.name, SUM(wsd.start_count) as total_start_count
	FROM workflow_stats_workflow_daily wsd
	JOIN workflow_defs wd ON wsd.workflow_id = wd.id
	WHERE wsd.stat_date >= %s
	GROUP BY wd.name
	ORDER BY total_start_count DESC
	`, topN, dateAddClause)

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*types.WorkflowRankItem
	for rows.Next() {
		var item types.WorkflowRankItem
		err := rows.Scan(
			&item.Name,
			&item.Count,
		)
		if err != nil {
			return nil, err
		}
		items = append(items, &item)
	}
	return items, nil
}

// RefreshWorkflowDailyStats 使用 MERGE 刷新指定日期的 workflow_stats_workflow_daily
func (r *AnalyticsRepository) RefreshWorkflowDailyStats(ctx context.Context, date time.Time) error {
	// 将日期格式化为当天的开始时间
	statDate := date.Format("2006-01-02")

	query := `
	MERGE INTO workflow_stats_workflow_daily AS target
	USING (
		SELECT 
			workflow_id,
			@p1 AS stat_date,
			COUNT(*) AS start_count,
			SUM(CASE WHEN status = 'completed' THEN 1 ELSE 0 END) AS complete_count,
			SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END) AS fail_count,
			AVG(CAST(DATEDIFF(SECOND, start_time, end_time) AS BIGINT)) * 1000 AS avg_execution_time_ms,
			MAX(CAST(DATEDIFF(SECOND, start_time, end_time) AS BIGINT)) * 1000 AS max_execution_time_ms,
			SUM(CASE WHEN DATEDIFF(SECOND, start_time, ISNULL(end_time, GETDATE())) > 3600 THEN 1 ELSE 0 END) AS timeout_count,
			SUM(CASE WHEN error_message LIKE '%reject%' THEN 1 ELSE 0 END) AS reject_count
		FROM workflow_instances
		WHERE CAST(start_time AS DATE) = @p1
		GROUP BY workflow_id
	) AS source
	ON (target.workflow_id = source.workflow_id AND target.stat_date = source.stat_date)
	WHEN MATCHED THEN
		UPDATE SET 
			start_count = source.start_count,
			complete_count = source.complete_count,
			fail_count = source.fail_count,
			avg_execution_time_ms = source.avg_execution_time_ms,
			max_execution_time_ms = source.max_execution_time_ms,
			timeout_count = source.timeout_count,
			reject_count = source.reject_count
	WHEN NOT MATCHED THEN
		INSERT (workflow_id, stat_date, start_count, complete_count, fail_count, 
				avg_execution_time_ms, max_execution_time_ms, timeout_count, reject_count)
		VALUES (source.workflow_id, source.stat_date, source.start_count, source.complete_count, 
				source.fail_count, source.avg_execution_time_ms, source.max_execution_time_ms, 
				source.timeout_count, source.reject_count);`

	_, err := r.db.ExecContext(ctx, query, statDate)
	return err
}

func (r *AnalyticsRepository) RefreshApprovalDailyStats(ctx context.Context, date time.Time) error {
	statDate := date.Format("2006-01-02")

	query := `
	MERGE INTO workflow_stats_approval_daily AS target
	USING (
		SELECT 
			approver AS user_id,
			@p1 AS stat_date,
			SUM(CASE WHEN action = 'approve' THEN 1 ELSE 0 END) AS approve_count,
			SUM(CASE WHEN action = 'reject' THEN 1 ELSE 0 END) AS reject_count,
			AVG(CAST(DATEDIFF(SECOND, operate_time, LEAD(operate_time) OVER (PARTITION BY approver ORDER BY operate_time)) AS BIGINT)) * 1000 AS avg_approval_time_ms,
			SUM(CASE WHEN DATEDIFF(SECOND, operate_time, GETDATE()) > 86400 THEN 1 ELSE 0 END) AS timeout_count
		FROM workflow_approval_records
		WHERE CAST(operate_time AS DATE) = @p1
		GROUP BY approver
	) AS source
	ON (target.user_id = source.user_id AND target.stat_date = source.stat_date)
	WHEN MATCHED THEN
		UPDATE SET 
			approve_count = source.approve_count,
			reject_count = source.reject_count,
			avg_approval_time_ms = source.avg_approval_time_ms,
			timeout_count = source.timeout_count
	WHEN NOT MATCHED THEN
		INSERT (user_id, stat_date, approve_count, reject_count, avg_approval_time_ms, timeout_count)
		VALUES (source.user_id, source.stat_date, source.approve_count, source.reject_count, 
				source.avg_approval_time_ms, source.timeout_count);`

	_, err := r.db.ExecContext(ctx, query, statDate)
	return err
}

func (r *AnalyticsRepository) Close() error {
	return r.db.Close()
}
