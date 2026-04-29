package sqlserver

import (
	"context"
	"database/sql"
	"time"

	api "github.com/zhiyunliu/distributed-workflow/runtime-execution/pkg/api"
	"github.com/zhiyunliu/distributed-workflow/runtime-execution/internal/common/types"
)

// 编译时接口断言
var _ api.AnalyticsRepository = (*AnalyticsRepository)(nil)

// AnalyticsRepository SQL Server 数据分析仓库实现
type AnalyticsRepository struct {
	db *sql.DB
}

// NewAnalyticsRepository 创建数据分析仓库实例，复用已有 *sql.DB 连接池
func NewAnalyticsRepository(db *sql.DB) *AnalyticsRepository {
	return &AnalyticsRepository{db: db}
}

// GetWorkflowCounts 查询流程总览计数：定义数、实例总数、本月新增、活跃实例数
func (r *AnalyticsRepository) GetWorkflowCounts(ctx context.Context) (totalDefs, totalInsts, monthlyNew, activeInsts int64, err error) {
	ctx, cancel := context.WithTimeout(ctx, defaultQueryTimeout)
	defer cancel()

	const q = `
SELECT
    (SELECT COUNT(1) FROM workflow_defs WHERE disabled = 0)                                      AS total_defs,
    (SELECT COUNT(1) FROM workflow_instances)                                                     AS total_insts,
    (SELECT COUNT(1) FROM workflow_instances WHERE start_time >= DATEADD(MONTH, -1, GETDATE()))  AS monthly_new,
    (SELECT COUNT(1) FROM workflow_instances WHERE status = 'running')                           AS active_insts`

	row := r.db.QueryRowContext(ctx, q)
	if err = row.Scan(&totalDefs, &totalInsts, &monthlyNew, &activeInsts); err != nil {
		err = wrapDBErr(err, "GetWorkflowCounts")
	}
	return
}

// GetAvgExecutionTime 查询平均执行时长（毫秒）和流程通过率
func (r *AnalyticsRepository) GetAvgExecutionTime(ctx context.Context) (avgMs int64, passRate float64, err error) {
	ctx, cancel := context.WithTimeout(ctx, defaultQueryTimeout)
	defer cancel()

	const q = `
SELECT
    ISNULL(AVG(DATEDIFF(MILLISECOND, start_time, end_time)), 0) AS avg_ms,
    CAST(SUM(CASE WHEN status = 'completed' THEN 1 ELSE 0 END) AS FLOAT)
        / NULLIF(SUM(CASE WHEN status IN ('completed','failed') THEN 1 ELSE 0 END), 0) AS pass_rate
FROM workflow_instances
WHERE status IN ('completed','failed') AND end_time IS NOT NULL`

	var passRateNull sql.NullFloat64
	row := r.db.QueryRowContext(ctx, q)
	if err = row.Scan(&avgMs, &passRateNull); err != nil {
		err = wrapDBErr(err, "GetAvgExecutionTime")
		return
	}
	if passRateNull.Valid {
		passRate = passRateNull.Float64
	}
	return
}

// GetWorkflowDailyStats 查询流程日统计数据（来自 stats_workflow_daily）
func (r *AnalyticsRepository) GetWorkflowDailyStats(ctx context.Context, params types.StatsQueryParams) ([]*types.WorkflowDailyStats, error) {
	ctx, cancel := context.WithTimeout(ctx, defaultQueryTimeout)
	defer cancel()

	q := `
SELECT workflow_id,
       CONVERT(VARCHAR(10), stat_date, 23) AS stat_date,
       start_count, complete_count, fail_count, timeout_count, reject_count,
       avg_execution_time_ms, max_execution_time_ms
FROM stats_workflow_daily
WHERE stat_date BETWEEN @startDate AND @endDate`

	args := []interface{}{
		sql.Named("startDate", params.StartDate),
		sql.Named("endDate", params.EndDate),
	}

	if params.WorkflowID != "" {
		q += " AND workflow_id = @workflowId"
		args = append(args, sql.Named("workflowId", params.WorkflowID))
	}
	q += " ORDER BY stat_date ASC, workflow_id"

	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, wrapDBErr(err, "GetWorkflowDailyStats")
	}
	defer rows.Close()

	var result []*types.WorkflowDailyStats
	for rows.Next() {
		s := &types.WorkflowDailyStats{}
		if err := rows.Scan(
			&s.WorkflowID, &s.StatDate,
			&s.StartCount, &s.CompleteCount, &s.FailCount,
			&s.TimeoutCount, &s.RejectCount,
			&s.AvgExecutionTimeMs, &s.MaxExecutionTimeMs,
		); err != nil {
			return nil, wrapDBErr(err, "GetWorkflowDailyStats scan")
		}
		result = append(result, s)
	}
	if err := rows.Err(); err != nil {
		return nil, wrapDBErr(err, "GetWorkflowDailyStats rows")
	}
	return result, nil
}

// GetApprovalDailyStats 查询审批日统计数据（来自 stats_approval_daily）
func (r *AnalyticsRepository) GetApprovalDailyStats(ctx context.Context, params types.StatsQueryParams) ([]*types.ApprovalPerformanceStats, error) {
	ctx, cancel := context.WithTimeout(ctx, defaultQueryTimeout)
	defer cancel()

	q := `
SELECT user_id,
       CONVERT(VARCHAR(10), stat_date, 23) AS stat_date,
       approve_count, reject_count, avg_approval_time_ms, timeout_count
FROM stats_approval_daily
WHERE stat_date BETWEEN @startDate AND @endDate`

	args := []interface{}{
		sql.Named("startDate", params.StartDate),
		sql.Named("endDate", params.EndDate),
	}

	if params.UserID != "" {
		q += " AND user_id = @userId"
		args = append(args, sql.Named("userId", params.UserID))
	}
	q += " ORDER BY stat_date ASC"

	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, wrapDBErr(err, "GetApprovalDailyStats")
	}
	defer rows.Close()

	var result []*types.ApprovalPerformanceStats
	for rows.Next() {
		s := &types.ApprovalPerformanceStats{}
		if err := rows.Scan(
			&s.UserID, &s.StatDate,
			&s.ApproveCount, &s.RejectCount, &s.AvgApprovalTimeMs, &s.TimeoutCount,
		); err != nil {
			return nil, wrapDBErr(err, "GetApprovalDailyStats scan")
		}
		result = append(result, s)
	}
	if err := rows.Err(); err != nil {
		return nil, wrapDBErr(err, "GetApprovalDailyStats rows")
	}
	return result, nil
}

// GetInstanceTrendData 按日期范围查询流程实例趋势，缺失日期补零
func (r *AnalyticsRepository) GetInstanceTrendData(ctx context.Context, params types.StatsQueryParams) (*types.StatsTrendData, error) {
	ctx, cancel := context.WithTimeout(ctx, defaultQueryTimeout)
	defer cancel()

	const q = `
SELECT CONVERT(VARCHAR(10), start_time, 23)                             AS day_date,
       COUNT(1)                                                          AS total,
       SUM(CASE WHEN status = 'completed' THEN 1 ELSE 0 END)            AS completed,
       SUM(CASE WHEN status = 'failed'    THEN 1 ELSE 0 END)            AS failed
FROM workflow_instances
WHERE start_time >= @startDate
  AND start_time < DATEADD(DAY, 1, CAST(@endDate AS DATE))
GROUP BY CONVERT(VARCHAR(10), start_time, 23)
ORDER BY day_date`

	rows, err := r.db.QueryContext(ctx, q,
		sql.Named("startDate", params.StartDate),
		sql.Named("endDate", params.EndDate),
	)
	if err != nil {
		return nil, wrapDBErr(err, "GetInstanceTrendData")
	}
	defer rows.Close()

	type dayRow struct{ total, completed, failed int64 }
	dayMap := make(map[string]dayRow)
	for rows.Next() {
		var d string
		var dr dayRow
		if err := rows.Scan(&d, &dr.total, &dr.completed, &dr.failed); err != nil {
			return nil, wrapDBErr(err, "GetInstanceTrendData scan")
		}
		dayMap[d] = dr
	}
	if err := rows.Err(); err != nil {
		return nil, wrapDBErr(err, "GetInstanceTrendData rows")
	}

	dates := buildDateRange(params.StartDate, params.EndDate)
	totalData := make([]int64, len(dates))
	completedData := make([]int64, len(dates))
	failedData := make([]int64, len(dates))

	for i, d := range dates {
		if dr, ok := dayMap[d]; ok {
			totalData[i] = dr.total
			completedData[i] = dr.completed
			failedData[i] = dr.failed
		}
	}

	return &types.StatsTrendData{
		Dates: dates,
		Series: []types.StatsSeries{
			{Name: "总计", Data: totalData},
			{Name: "已完成", Data: completedData},
			{Name: "失败", Data: failedData},
		},
	}, nil
}

// RefreshWorkflowDailyStats 使用 MERGE 刷新指定日期的 stats_workflow_daily
func (r *AnalyticsRepository) RefreshWorkflowDailyStats(ctx context.Context, date string) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	const q = `
MERGE INTO stats_workflow_daily AS target
USING (
    SELECT workflow_id,
           COUNT(1)                                                            AS start_count,
           SUM(CASE WHEN status = 'completed' THEN 1 ELSE 0 END)              AS complete_count,
           SUM(CASE WHEN status = 'failed'    THEN 1 ELSE 0 END)              AS fail_count,
           ISNULL(AVG(DATEDIFF(MILLISECOND, start_time, end_time)), 0)        AS avg_ms,
           ISNULL(MAX(DATEDIFF(MILLISECOND, start_time, end_time)), 0)        AS max_ms
    FROM workflow_instances
    WHERE CONVERT(DATE, start_time) = @p1
    GROUP BY workflow_id
) AS source
ON  target.workflow_id = source.workflow_id
AND target.stat_date   = @p1
WHEN MATCHED THEN
    UPDATE SET
        start_count           = source.start_count,
        complete_count        = source.complete_count,
        fail_count            = source.fail_count,
        avg_execution_time_ms = source.avg_ms,
        max_execution_time_ms = source.max_ms
WHEN NOT MATCHED THEN
    INSERT (workflow_id, stat_date, start_count, complete_count, fail_count,
            avg_execution_time_ms, max_execution_time_ms, timeout_count, reject_count)
    VALUES (source.workflow_id, @p1, source.start_count, source.complete_count, source.fail_count,
            source.avg_ms, source.max_ms, 0, 0);`

	_, err := r.db.ExecContext(ctx, q, sql.Named("p1", date))
	return wrapDBErr(err, "RefreshWorkflowDailyStats")
}

// RefreshApprovalDailyStats 使用 MERGE 刷新指定日期的 stats_approval_daily
func (r *AnalyticsRepository) RefreshApprovalDailyStats(ctx context.Context, date string) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	const q = `
MERGE INTO stats_approval_daily AS target
USING (
    SELECT approver                                                              AS user_id,
           SUM(CASE WHEN action = 'approve' THEN 1 ELSE 0 END)                  AS approve_count,
           SUM(CASE WHEN action = 'reject'  THEN 1 ELSE 0 END)                  AS reject_count,
           0                                                                      AS avg_ms
    FROM workflow_approval_records
    WHERE CONVERT(DATE, operate_time) = @p1
    GROUP BY approver
) AS source
ON  target.user_id   = source.user_id
AND target.stat_date = @p1
WHEN MATCHED THEN
    UPDATE SET
        approve_count        = source.approve_count,
        reject_count         = source.reject_count,
        avg_approval_time_ms = source.avg_ms
WHEN NOT MATCHED THEN
    INSERT (user_id, stat_date, approve_count, reject_count, avg_approval_time_ms, timeout_count)
    VALUES (source.user_id, @p1, source.approve_count, source.reject_count, source.avg_ms, 0);`

	_, err := r.db.ExecContext(ctx, q, sql.Named("p1", date))
	return wrapDBErr(err, "RefreshApprovalDailyStats")
}

// buildDateRange 生成 [startDate, endDate] 闭区间内每天的日期字符串（格式 "2006-01-02"）
func buildDateRange(startDate, endDate string) []string {
	start, err := time.Parse("2006-01-02", startDate)
	if err != nil {
		return nil
	}
	end, err := time.Parse("2006-01-02", endDate)
	if err != nil {
		return nil
	}
	var dates []string
	for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
		dates = append(dates, d.Format("2006-01-02"))
	}
	return dates
}
