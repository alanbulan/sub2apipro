package repository

import (
	"context"
	"encoding/json"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

// Shared by ordinary result creation and transactional quality-batch collection.
func insertScheduledTestResult(ctx context.Context, q planRowQuerier, result *service.ScheduledTestResult) (*service.ScheduledTestResult, error) {
	row := q.QueryRowContext(ctx, `
		INSERT INTO scheduled_test_results (plan_id, status, response_text, error_message, latency_ms, started_at, finished_at, created_at, pelican_config, quality_action, quality_judgment, quality_round_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), $8, $9, $10, $11)
		RETURNING id, plan_id, status, response_text, error_message, latency_ms, started_at, finished_at, created_at, pelican_config, quality_action, quality_judgment, quality_round_id
	`, result.PlanID, result.Status, result.ResponseText, result.ErrorMessage, result.LatencyMs, result.StartedAt, result.FinishedAt, marshalPelicanConfig(result.PelicanConfig), result.QualityAction, marshalQualityJudgment(result.QualityJudgment), result.QualityRoundID)

	out := &service.ScheduledTestResult{}
	var judgment []byte
	var config []byte
	if err := row.Scan(
		&out.ID, &out.PlanID, &out.Status, &out.ResponseText, &out.ErrorMessage,
		&out.LatencyMs, &out.StartedAt, &out.FinishedAt, &out.CreatedAt, &config, &out.QualityAction, &judgment, &out.QualityRoundID,
	); err != nil {
		return nil, err
	}
	if len(config) > 0 {
		if err := json.Unmarshal(config, &out.PelicanConfig); err != nil {
			return nil, err
		}
	}
	if len(judgment) > 0 {
		if err := json.Unmarshal(judgment, &out.QualityJudgment); err != nil {
			return nil, err
		}
	}
	return out, nil
}
