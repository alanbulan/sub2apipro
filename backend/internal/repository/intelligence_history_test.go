package repository

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

// Runs only against the isolated Actions PostgreSQL used by the canonical
// migration rehearsal; no real account, credential or model request is involved.
func testCanonicalIntelligenceHistory(t *testing.T, ctx context.Context, db *sql.DB) {
	t.Helper()
	_, err := db.ExecContext(ctx, `INSERT INTO accounts(id,name,platform,type,credentials) VALUES
        (69001,'IQ source A','openai','apikey','{"api_key":"must-not-leak"}'),
        (69002,'IQ source B','openai','apikey','{}')`)
	require.NoError(t, err)
	plans := &scheduledTestPlanRepository{db: db}
	results := &scheduledTestResultRepository{db: db}
	svc := service.NewScheduledTestService(plans, results)
	var stored []*service.ScheduledTestResult
	for _, id := range []int64{69001, 69002} {
		plan, err := svc.CreatePlan(ctx, &service.ScheduledTestPlan{AccountID: id, ModelID: "fixture-model", CronExpression: "0 */6 * * *", Enabled: false, MaxResults: 100, PelicanConfig: &service.PelicanTestConfig{QuestionKind: "candy", ParallelCount: 1}})
		require.NoError(t, err)
		for i := 0; i < 2; i++ {
			result, err := results.Create(ctx, &service.ScheduledTestResult{PlanID: plan.ID, Status: "success", ResponseText: "21", StartedAt: time.Now(), FinishedAt: time.Now(), PelicanConfig: plan.PelicanConfig})
			require.NoError(t, err)
			stored = append(stored, result)
		}
	}
	// Connection tests and soft-deleted accounts never enter the IQ dashboard.
	_, err = db.ExecContext(ctx, `INSERT INTO scheduled_test_plans(id,account_id,model_id,cron_expression,enabled,max_results) VALUES(69001,69001,'fixture-model','0 */6 * * *',false,10);
        INSERT INTO scheduled_test_results(plan_id,status,response_text,latency_ms,started_at,finished_at) VALUES(69001,'success','connectivity only',1,NOW(),NOW())`)
	require.NoError(t, err)
	first, err := svc.ListPelicanHistory(ctx, 0, 2, 0)
	require.NoError(t, err)
	require.Len(t, first.Items, 2)
	require.NotZero(t, first.NextCursor)
	require.Empty(t, first.Items[0].ResponseText, "cards lazy-load response bodies")
	require.Equal(t, "IQ source B", first.Items[0].AccountName)
	second, err := svc.ListPelicanHistory(ctx, first.NextCursor, 2, 0)
	require.NoError(t, err)
	require.Len(t, second.Items, 2)
	require.Zero(t, second.NextCursor)
	require.Less(t, second.Items[0].ID, first.Items[1].ID)
	scoped, err := svc.ListPelicanHistory(ctx, 0, 100, 69001)
	require.NoError(t, err)
	require.Len(t, scoped.Items, 2)
	for _, result := range scoped.Items {
		require.Equal(t, int64(69001), result.AccountID)
	}
	detail, err := svc.GetResult(ctx, stored[0].PlanID, stored[0].ID)
	require.NoError(t, err)
	require.Equal(t, "21", detail.ResponseText)
	require.Equal(t, "candy", detail.PelicanConfig.QuestionKind)
	_, err = svc.GetResult(ctx, stored[2].PlanID, stored[0].ID)
	require.ErrorIs(t, err, sql.ErrNoRows, "a result cannot be read through another plan")
	summaries, err := svc.ListResults(ctx, stored[0].PlanID, 20, false)
	require.NoError(t, err)
	require.Len(t, summaries, 2)
	require.Empty(t, summaries[0].ResponseText)
	full, err := svc.ListResults(ctx, stored[0].PlanID, 20)
	require.NoError(t, err)
	require.Equal(t, "21", full[0].ResponseText, "canonical callers still receive response text by default")
	_, err = db.ExecContext(ctx, `UPDATE accounts SET deleted_at=NOW() WHERE id=69002`)
	require.NoError(t, err)
	afterDelete, err := svc.ListPelicanHistory(ctx, 0, 100, 0)
	require.NoError(t, err)
	require.Len(t, afterDelete.Items, 2)
}
