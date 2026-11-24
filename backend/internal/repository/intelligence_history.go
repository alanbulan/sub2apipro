package repository

import (
	"context"
	"encoding/json"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

func (r *scheduledTestResultRepository) GetResult(ctx context.Context, planID, resultID int64) (*service.ScheduledTestResult, error) {
	out := &service.ScheduledTestResult{}
	var config []byte
	err := r.db.QueryRowContext(ctx, `SELECT id, plan_id, status, response_text, error_message, latency_ms, started_at, finished_at, created_at, pelican_config
        FROM scheduled_test_results WHERE plan_id=$1 AND id=$2`, planID, resultID).Scan(
		&out.ID, &out.PlanID, &out.Status, &out.ResponseText, &out.ErrorMessage,
		&out.LatencyMs, &out.StartedAt, &out.FinishedAt, &out.CreatedAt, &config)
	if err != nil {
		return nil, err
	}
	if err := unmarshalIntelligenceConfig(config, &out.PelicanConfig); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *scheduledTestResultRepository) ListPelicanHistory(ctx context.Context, beforeID int64, limit int, accountID int64) ([]*service.PelicanHistoryResult, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT r.id, r.plan_id, r.status, r.error_message,
 r.latency_ms, r.started_at, r.finished_at, r.created_at, r.pelican_config, a.id, a.name
 FROM scheduled_test_results r
 JOIN scheduled_test_plans p ON p.id = r.plan_id
 JOIN accounts a ON a.id = p.account_id
 WHERE p.pelican_config IS NOT NULL AND p.pelican_config->'quality' IS NULL AND a.deleted_at IS NULL
 AND ($1::bigint = 0 OR r.id < $1)
 AND ($3::bigint = 0 OR p.account_id = $3)
 ORDER BY r.id DESC LIMIT $2`, beforeID, limit, accountID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	results := make([]*service.PelicanHistoryResult, 0)
	for rows.Next() {
		result := &service.PelicanHistoryResult{}
		var config []byte
		if err := rows.Scan(&result.ID, &result.PlanID, &result.Status, &result.ErrorMessage,
			&result.LatencyMs, &result.StartedAt, &result.FinishedAt, &result.CreatedAt,
			&config, &result.AccountID, &result.AccountName); err != nil {
			return nil, err
		}
		if len(config) > 0 {
			if err := json.Unmarshal(config, &result.PelicanConfig); err != nil {
				return nil, err
			}
		}
		results = append(results, result)
	}
	return results, rows.Err()
}
