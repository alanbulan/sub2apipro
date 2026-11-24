package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

// --- Plan Repository ---

type scheduledTestPlanRepository struct {
	db *sql.DB
}

func NewScheduledTestPlanRepository(db *sql.DB) service.ScheduledTestPlanRepository {
	return &scheduledTestPlanRepository{db: db}
}

func (r *scheduledTestPlanRepository) Create(ctx context.Context, plan *service.ScheduledTestPlan) (*service.ScheduledTestPlan, error) {
	row := r.db.QueryRowContext(ctx, `
		INSERT INTO scheduled_test_plans (account_id, model_id, cron_expression, enabled, max_results, auto_recover, next_run_at, created_at, updated_at, pelican_config)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW(), $8)
		RETURNING id, account_id, model_id, cron_expression, enabled, max_results, auto_recover, last_run_at, next_run_at, created_at, updated_at, pelican_config
	`, plan.AccountID, plan.ModelID, plan.CronExpression, plan.Enabled, plan.MaxResults, plan.AutoRecover, plan.NextRunAt, marshalIntelligenceConfig(plan.PelicanConfig))
	return scanPlan(row)
}

func (r *scheduledTestPlanRepository) GetByID(ctx context.Context, id int64) (*service.ScheduledTestPlan, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, account_id, model_id, cron_expression, enabled, max_results, auto_recover, last_run_at, next_run_at, created_at, updated_at, pelican_config
		FROM scheduled_test_plans WHERE id = $1
	`, id)
	return scanPlan(row)
}

func (r *scheduledTestPlanRepository) ListByAccountID(ctx context.Context, accountID int64) ([]*service.ScheduledTestPlan, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, account_id, model_id, cron_expression, enabled, max_results, auto_recover, last_run_at, next_run_at, created_at, updated_at, pelican_config
		FROM scheduled_test_plans WHERE account_id = $1
		ORDER BY created_at DESC
	`, accountID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	return scanPlans(rows)
}

func (r *scheduledTestPlanRepository) ListDue(ctx context.Context, now time.Time) ([]*service.ScheduledTestPlan, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, account_id, model_id, cron_expression, enabled, max_results, auto_recover, last_run_at, next_run_at, created_at, updated_at, pelican_config
		FROM scheduled_test_plans
		WHERE enabled = true AND next_run_at <= $1 AND (running_until IS NULL OR running_until <= $1)
		ORDER BY next_run_at ASC
	`, now)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	return scanPlans(rows)
}

func (r *scheduledTestPlanRepository) Update(ctx context.Context, plan *service.ScheduledTestPlan) (*service.ScheduledTestPlan, error) {
	row := r.db.QueryRowContext(ctx, `
		UPDATE scheduled_test_plans
		SET model_id = $2, cron_expression = $3, enabled = $4, max_results = $5, auto_recover = $6, next_run_at = $7, updated_at = NOW(), pelican_config = $8
		WHERE id = $1
		RETURNING id, account_id, model_id, cron_expression, enabled, max_results, auto_recover, last_run_at, next_run_at, created_at, updated_at, pelican_config
	`, plan.ID, plan.ModelID, plan.CronExpression, plan.Enabled, plan.MaxResults, plan.AutoRecover, plan.NextRunAt, marshalIntelligenceConfig(plan.PelicanConfig))
	return scanPlan(row)
}

func (r *scheduledTestPlanRepository) Delete(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM scheduled_test_plans WHERE id = $1`, id)
	return err
}

func (r *scheduledTestPlanRepository) UpdateAfterRun(ctx context.Context, id int64, lastRunAt time.Time, nextRunAt time.Time) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE scheduled_test_plans SET last_run_at = $2, next_run_at = $3, running_until = NULL, updated_at = NOW() WHERE id = $1
	`, id, lastRunAt, nextRunAt)
	return err
}

// --- Result Repository ---

type scheduledTestResultRepository struct {
	db *sql.DB
}

func NewScheduledTestResultRepository(db *sql.DB) service.ScheduledTestResultRepository {
	return &scheduledTestResultRepository{db: db}
}

func (r *scheduledTestResultRepository) Create(ctx context.Context, result *service.ScheduledTestResult) (*service.ScheduledTestResult, error) {
	row := r.db.QueryRowContext(ctx, `
		INSERT INTO scheduled_test_results (plan_id, status, response_text, error_message, latency_ms, started_at, finished_at, created_at, pelican_config)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), $8)
		RETURNING id, plan_id, status, response_text, error_message, latency_ms, started_at, finished_at, created_at, pelican_config
	`, result.PlanID, result.Status, result.ResponseText, result.ErrorMessage, result.LatencyMs, result.StartedAt, result.FinishedAt, marshalIntelligenceConfig(result.PelicanConfig))

	out := &service.ScheduledTestResult{}
	var config []byte
	if err := row.Scan(
		&out.ID, &out.PlanID, &out.Status, &out.ResponseText, &out.ErrorMessage,
		&out.LatencyMs, &out.StartedAt, &out.FinishedAt, &out.CreatedAt, &config,
	); err != nil {
		return nil, err
	}
	if err := unmarshalIntelligenceConfig(config, &out.PelicanConfig); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *scheduledTestResultRepository) ListByPlanID(ctx context.Context, planID int64, limit int, includeContent ...bool) ([]*service.ScheduledTestResult, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, plan_id, status, CASE WHEN $3 THEN response_text ELSE '' END, error_message, latency_ms, started_at, finished_at, created_at, pelican_config
		FROM scheduled_test_results
		WHERE plan_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`, planID, limit, len(includeContent) == 0 || includeContent[0])
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var results []*service.ScheduledTestResult
	for rows.Next() {
		r := &service.ScheduledTestResult{}
		var config []byte
		if err := rows.Scan(
			&r.ID, &r.PlanID, &r.Status, &r.ResponseText, &r.ErrorMessage,
			&r.LatencyMs, &r.StartedAt, &r.FinishedAt, &r.CreatedAt, &config,
		); err != nil {
			return nil, err
		}
		if err := unmarshalIntelligenceConfig(config, &r.PelicanConfig); err != nil {
			return nil, err
		}
		results = append(results, r)
	}
	return results, rows.Err()
}

func (r *scheduledTestResultRepository) PruneOldResults(ctx context.Context, planID int64, keepCount int) error {
	_, err := r.db.ExecContext(ctx, `
		DELETE FROM scheduled_test_results
		WHERE id IN (
			SELECT id FROM (
				SELECT id, ROW_NUMBER() OVER (PARTITION BY plan_id ORDER BY created_at DESC) AS rn
				FROM scheduled_test_results
				WHERE plan_id = $1
			) ranked
			WHERE rn > $2
		)
	`, planID, keepCount)
	return err
}

// --- scan helpers ---

type scannable interface {
	Scan(dest ...any) error
}

func scanPlan(row scannable) (*service.ScheduledTestPlan, error) {
	p := &service.ScheduledTestPlan{}
	var config []byte
	if err := row.Scan(
		&p.ID, &p.AccountID, &p.ModelID, &p.CronExpression, &p.Enabled, &p.MaxResults, &p.AutoRecover,
		&p.LastRunAt, &p.NextRunAt, &p.CreatedAt, &p.UpdatedAt, &config,
	); err != nil {
		return nil, err
	}
	if err := unmarshalIntelligenceConfig(config, &p.PelicanConfig); err != nil {
		return nil, err
	}
	return p, nil
}

func scanPlans(rows *sql.Rows) ([]*service.ScheduledTestPlan, error) {
	var plans []*service.ScheduledTestPlan
	for rows.Next() {
		p, err := scanPlan(rows)
		if err != nil {
			return nil, err
		}
		plans = append(plans, p)
	}
	return plans, rows.Err()
}

func marshalIntelligenceConfig(c *service.PelicanTestConfig) any {
	if c == nil {
		return nil
	}
	b, _ := json.Marshal(c)
	return string(b)
}
func unmarshalIntelligenceConfig(b []byte, out **service.PelicanTestConfig) error {
	if len(b) == 0 {
		return nil
	}
	return json.Unmarshal(b, out)
}

// A database lease prevents duplicate charged tests across runner replicas.
func (r *scheduledTestPlanRepository) ClaimRun(ctx context.Context, id int64, now, until time.Time) (bool, error) {
	result, err := r.db.ExecContext(ctx, `UPDATE scheduled_test_plans SET running_until=$3 WHERE id=$1 AND enabled=true AND next_run_at <= $2 AND (running_until IS NULL OR running_until <= $2)`, id, now, until)
	if err != nil {
		return false, err
	}
	n, err := result.RowsAffected()
	return n == 1, err
}
