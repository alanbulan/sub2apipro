package repository

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"sync"
	"sync/atomic"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"testing"
	"testing/fstest"
	"time"

	"github.com/Wei-Shaw/sub2api/migrations"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

// The isolated runner exports the exact deployed fork tree. Real production
// credentials and records are never copied into this rehearsal.
func TestCanonicalMigrationUpgradePreservesCustomData(t *testing.T) {
	dsn, directory := os.Getenv("CANONICAL_UPGRADE_DSN"), os.Getenv("CANONICAL_PREVIOUS_MIGRATIONS")
	if dsn == "" || directory == "" {
		t.Skip("isolated PostgreSQL and pinned fork migration archive required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()
	admin, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	t.Cleanup(func() { _ = admin.Close() })
	baseline := fstest.MapFS{}
	production := os.DirFS(directory)
	names, err := fs.Glob(production, "*.sql")
	require.NoError(t, err)
	require.Len(t, names, 461, "pin the exact pre-canonical Pro schema")
	for _, name := range names {
		before, err := fs.ReadFile(production, name)
		require.NoError(t, err)
		// Retired extensions leave the runtime migration list, but their existing
		// ledger entries remain authoritative and their data is never dropped.
		if after, readErr := migrations.FS.ReadFile(name); readErr == nil {
			require.Equal(t, migrationChecksum(string(before)), migrationChecksum(string(after)), name)
		}
		baseline[name] = &fstest.MapFile{Data: before}
	}
	database := fmt.Sprintf("fork_upgrade_%d", time.Now().UnixNano())
	_, err = admin.ExecContext(ctx, "CREATE DATABASE "+database)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, err := admin.Exec("DROP DATABASE " + database + " WITH (FORCE)")
		require.NoError(t, err)
	})
	u, err := url.Parse(dsn)
	require.NoError(t, err)
	u.Path = "/" + database
	db, err := sql.Open("postgres", u.String())
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	require.NoError(t, applyMigrationsFS(ctx, db, baseline))
	var beforeCount int
	require.NoError(t, db.QueryRowContext(ctx, "SELECT COUNT(*) FROM schema_migrations").Scan(&beforeCount))
	require.Equal(t, 461, beforeCount)
	_, err = db.ExecContext(ctx, `
INSERT INTO users(id,email,password_hash,role,balance,signup_source) VALUES(67001,'employee@feishu.invalid','synthetic-only','admin',17.25,'feishu');
INSERT INTO groups(id,name,platform,rate_multiplier,is_exclusive) VALUES(67001,'retained-pool','openai',0.8,true);
INSERT INTO accounts(id,name,platform,type,credentials,extra) VALUES(67001,'retained-upstream','openai','apikey','{"api_key":"synthetic-upstream"}','{"pool_account":true,"custom_marker":"retained"}');
INSERT INTO account_groups(account_id,group_id,priority) VALUES(67001,67001,10);
INSERT INTO api_keys(id,user_id,key,name,group_id) VALUES(67001,67001,'sk-synthetic-upgrade-only','retained-key',67001);
INSERT INTO auth_identities(id,user_id,provider_type,provider_key,provider_subject) VALUES(67001,67001,'feishu','synthetic-app:tenant','synthetic-open-id');
INSERT INTO auth_identity_channels(identity_id,provider_type,provider_key,channel,channel_app_id,channel_subject) VALUES(67001,'feishu','synthetic-app:tenant','web','synthetic-app','synthetic-open-id');
INSERT INTO user_provider_default_grants(user_id,provider_type,grant_reason) VALUES(67001,'feishu','signup');
INSERT INTO conversation_logs(request_id,session_id,user_id,api_key_id,protocol,endpoint,status,request_body,response_body) VALUES('synthetic-request','synthetic-session',67001,67001,'responses','/v1/responses','completed','synthetic request','synthetic response');
INSERT INTO usage_logs(user_id,api_key_id,account_id,request_id,model,input_tokens,output_tokens,total_cost,actual_cost) VALUES(67001,67001,67001,'synthetic-request','synthetic-model',10,2,0.25,0.2);
INSERT INTO scheduled_test_plans(id,account_id,model_id,cron_expression,enabled,max_results,pelican_config) VALUES(67001,67001,'synthetic-model','0 */6 * * *',true,50,'{"question_kind":"candy","prompt":"synthetic question","reasoning_effort":"high","parallel_count":1}');
INSERT INTO scheduled_test_results(plan_id,status,response_text,latency_ms,started_at,finished_at,pelican_config) VALUES(67001,'success','synthetic answer',1,NOW(),NOW(),'{"question_kind":"candy","model_id":"synthetic-model","parallel_count":1}');
INSERT INTO settings(key,value) VALUES('site_name','BeaconChip fixture'),('account_ops_notifications_v1','{"enabled":false,"custom_marker":"retained"}') ON CONFLICT(key) DO UPDATE SET value=EXCLUDED.value;
`)
	require.NoError(t, err)
	// Compare only established fields: additive feature columns are expected.
	// Hash the snapshot so a failed assertion never prints credential/body data.
	snapshot := func() [32]byte {
		var raw string
		require.NoError(t, db.QueryRowContext(ctx, `SELECT jsonb_build_object(
'users',(SELECT jsonb_agg(jsonb_build_array(id,email,password_hash,role,balance,signup_source) ORDER BY id) FROM users),
'groups',(SELECT jsonb_agg(jsonb_build_array(id,name,platform,rate_multiplier,is_exclusive,status) ORDER BY id) FROM groups),
'accounts',(SELECT jsonb_agg(jsonb_build_array(id,name,platform,type,credentials,extra,status) ORDER BY id) FROM accounts),
'keys',(SELECT jsonb_agg(jsonb_build_array(id,user_id,key,name,group_id,status) ORDER BY id) FROM api_keys),
'memberships',(SELECT jsonb_agg(to_jsonb(m) ORDER BY account_id,group_id) FROM account_groups m),
'identities',(SELECT jsonb_agg(to_jsonb(i) ORDER BY id) FROM auth_identities i),
'channels',(SELECT jsonb_agg(to_jsonb(c) ORDER BY id) FROM auth_identity_channels c),
'grants',(SELECT jsonb_agg(to_jsonb(g) ORDER BY id) FROM user_provider_default_grants g),
'conversations',(SELECT jsonb_agg(to_jsonb(c) ORDER BY id) FROM conversation_logs c),
'iq_plans',(SELECT jsonb_agg(to_jsonb(p) ORDER BY id) FROM scheduled_test_plans p),
'iq_results',(SELECT jsonb_agg(to_jsonb(r) ORDER BY id) FROM scheduled_test_results r),
'usage',(SELECT jsonb_agg(jsonb_build_array(id,user_id,api_key_id,account_id,request_id,model,input_tokens,output_tokens,total_cost,actual_cost) ORDER BY id) FROM usage_logs),
'settings',(SELECT jsonb_agg(jsonb_build_array(key,value) ORDER BY key) FROM settings WHERE key IN ('site_name','account_ops_notifications_v1'))
)::text`).Scan(&raw))
		return sha256.Sum256([]byte(raw))
	}
	before := snapshot()
	for range 2 {
		require.NoError(t, ApplyMigrations(ctx, db))
		require.Equal(t, before, snapshot(), "upgrade/replay must retain fork identities, balances, keys and independent conversation logs")
	}
	currentNames, err := fs.Glob(migrations.FS, "*.sql")
	require.NoError(t, err)
	var afterCount int
	require.NoError(t, db.QueryRowContext(ctx, "SELECT COUNT(*) FROM schema_migrations").Scan(&afterCount))
	union := map[string]bool{}
	for _, name := range names {
		union[name] = true
	}
	for _, name := range currentNames {
		union[name] = true
	}
	require.Equal(t, len(union), afterCount)
	for _, name := range names {
		var stored string
		require.NoError(t, db.QueryRowContext(ctx, "SELECT checksum FROM schema_migrations WHERE filename=$1", name).Scan(&stored))
		require.Equal(t, migrationChecksum(string(baseline[name].Data)), stored, name)
	}
	// The baseline migration set can be replayed against the upgraded database.
	// This exercises migration compatibility, not the old application binary.
	require.NoError(t, applyMigrationsFS(ctx, db, baseline))
	require.Equal(t, before, snapshot())
	_, err = db.ExecContext(ctx, `INSERT INTO auth_identities(user_id,provider_type,provider_key,provider_subject) VALUES(67001,'feishu','synthetic-second-app:tenant','second-open-id')`)
	require.NoError(t, err, "Feishu remains an accepted provider after every new migration")
	var longContext bool
	var limit int
	require.NoError(t, db.QueryRowContext(ctx, "SELECT allow_long_context,context_token_limit FROM api_keys WHERE id=67001").Scan(&longContext, &limit))
	require.True(t, longContext)
	require.Zero(t, limit)
	t.Logf("upgraded immutable fork migrations: %d -> %d; custom data, replay and Feishu checks passed", beforeCount, afterCount)
}

// Fresh installations create only canonical tables and retained Pro extensions.
func TestCanonicalMigrationFreshInstall(t *testing.T) {
	dsn := os.Getenv("CANONICAL_UPGRADE_DSN")
	if dsn == "" {
		t.Skip("isolated Actions PostgreSQL required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()
	admin, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	t.Cleanup(func() { _ = admin.Close() })
	name := fmt.Sprintf("canonical_fresh_%d", time.Now().UnixNano())
	_, err = admin.ExecContext(ctx, "CREATE DATABASE "+name)
	require.NoError(t, err)
	t.Cleanup(func() { _, err := admin.Exec("DROP DATABASE " + name + " WITH (FORCE)"); require.NoError(t, err) })
	u, err := url.Parse(dsn)
	require.NoError(t, err)
	u.Path = "/" + name
	db, err := sql.Open("postgres", u.String())
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	require.NoError(t, ApplyMigrations(ctx, db))
	require.NoError(t, ApplyMigrations(ctx, db))
	var exists bool
	for _, table := range []string{"users", "api_keys", "auth_identities", "conversation_logs", "scheduled_test_plans", "scheduled_test_results"} {
		require.NoError(t, db.QueryRowContext(ctx, "SELECT to_regclass($1) IS NOT NULL", table).Scan(&exists))
		require.True(t, exists, table)
	}
	for _, table := range []string{"auto_credit_config", "quality_probe_batches", "account_quality_states", "api_key_groups"} {
		require.NoError(t, db.QueryRowContext(ctx, "SELECT to_regclass($1) IS NOT NULL", table).Scan(&exists))
		require.False(t, exists, table)
	}
	var n int
	require.NoError(t, db.QueryRowContext(ctx, "SELECT COUNT(*) FROM schema_migrations").Scan(&n))
	files, err := fs.Glob(migrations.FS, "*.sql")
	require.NoError(t, err)
	require.Equal(t, len(files), n)
	testCanonicalPolicyBridge(t, ctx, db)
	testCanonicalIntelligenceLease(t, ctx, db)
	testCanonicalIntelligenceHistory(t, ctx, db)
}

func testCanonicalPolicyBridge(t *testing.T, ctx context.Context, db *sql.DB) {
	t.Helper()
	_, err := db.ExecContext(ctx, `
INSERT INTO users(id,email,password_hash,role) VALUES(68001,'synthetic-policy@example.invalid','fixture','user');
INSERT INTO groups(id,name,platform) VALUES(68001,'fixture-group','openai'),(68002,'other-fixture-group','openai');
INSERT INTO api_keys(id,user_id,key,name) VALUES(68001,68001,'sk-fixture-policy','policy-fixture');
CREATE TABLE api_key_groups(api_key_id BIGINT, group_id BIGINT);
INSERT INTO api_key_groups VALUES(68001,68001),(68001,68002);`)
	require.NoError(t, err)
	bridge, err := migrations.FS.ReadFile("900_pro_canonical_policy_bridge.sql")
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, string(bridge))
	require.ErrorContains(t, err, "active multi-group API keys")
	var assigned sql.NullInt64
	require.NoError(t, db.QueryRowContext(ctx, "SELECT group_id FROM api_keys WHERE id=68001").Scan(&assigned))
	require.False(t, assigned.Valid, "ambiguous legacy permissions must remain untouched")
	_, err = db.ExecContext(ctx, "DELETE FROM api_key_groups WHERE group_id=68002")
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, string(bridge))
	require.NoError(t, err)
	require.NoError(t, db.QueryRowContext(ctx, "SELECT group_id FROM api_keys WHERE id=68001").Scan(&assigned))
	require.Equal(t, int64(68001), assigned.Int64)
	_, err = db.ExecContext(ctx, "UPDATE api_key_groups SET group_id=68002")
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, string(bridge))
	require.ErrorContains(t, err, "conflicting API key group bindings")
	_, err = db.ExecContext(ctx, `DELETE FROM api_key_groups;
ALTER TABLE account_groups ADD COLUMN allowed_models JSONB;
INSERT INTO accounts(id,name,platform,type,credentials) VALUES(68001,'fixture-source','openai','apikey','{}');
INSERT INTO account_groups(account_id,group_id,allowed_models) VALUES(68001,68001,'["restricted-model"]');`)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, string(bridge))
	require.ErrorContains(t, err, "retired per-account-group model restrictions")
	_, err = db.ExecContext(ctx, `UPDATE account_groups SET allowed_models=NULL;
ALTER TABLE user_group_rate_multipliers ADD COLUMN denied_models JSONB;
INSERT INTO user_group_rate_multipliers(user_id,group_id,denied_models) VALUES(68001,68001,'["blocked-model"]');`)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, string(bridge))
	require.ErrorContains(t, err, "retired per-user-group model restrictions")
	_, err = db.ExecContext(ctx, "UPDATE user_group_rate_multipliers SET denied_models=NULL")
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, string(bridge))
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `ALTER TABLE api_keys ADD COLUMN concurrency_limit BIGINT DEFAULT 0;
ALTER TABLE api_keys ADD COLUMN model_mapping JSONB DEFAULT '{}';
ALTER TABLE api_keys ADD COLUMN reasoning_effort VARCHAR(16) DEFAULT '';
ALTER TABLE api_keys ADD COLUMN allow_long_context BOOLEAN DEFAULT true;
ALTER TABLE api_keys ADD COLUMN context_token_limit INTEGER DEFAULT 0;`)
	require.NoError(t, err)
	for _, policy := range []struct{ column, value, reset string }{
		{"concurrency_limit", "2", "0"}, {"model_mapping", `'{"from":"to"}'::jsonb`, `'{}'::jsonb`},
		{"reasoning_effort", "'high'", "''"}, {"allow_long_context", "false", "true"}, {"context_token_limit", "4096", "0"},
	} {
		_, err = db.ExecContext(ctx, "UPDATE api_keys SET "+policy.column+"="+policy.value+" WHERE id=68001")
		require.NoError(t, err)
		_, err = db.ExecContext(ctx, string(bridge))
		require.ErrorContains(t, err, "retired API key "+policy.column+" policy")
		_, err = db.ExecContext(ctx, "UPDATE api_keys SET "+policy.column+"="+policy.reset+" WHERE id=68001")
		require.NoError(t, err)
	}
	_, err = db.ExecContext(ctx, string(bridge))
	require.NoError(t, err)

}

func testCanonicalIntelligenceLease(t *testing.T, ctx context.Context, db *sql.DB) {
	t.Helper()
	repo := &scheduledTestPlanRepository{db: db}
	now := time.Now().UTC()
	due := now.Add(-time.Minute)
	plan, err := repo.Create(ctx, &service.ScheduledTestPlan{AccountID: 68001, ModelID: "fixture-model", CronExpression: "0 */6 * * *", Enabled: true, MaxResults: 50, NextRunAt: &due, PelicanConfig: &service.PelicanTestConfig{QuestionKind: "candy", ParallelCount: 1}})
	require.NoError(t, err)
	var winners atomic.Int64
	var wg sync.WaitGroup
	failures := make(chan error, 24)
	for range 24 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ok, err := repo.ClaimRun(ctx, plan.ID, now, now.Add(10*time.Minute))
			if err != nil {
				failures <- err
			}
			if ok {
				winners.Add(1)
			}
		}()
	}
	wg.Wait()
	close(failures)
	for err := range failures {
		require.NoError(t, err)
	}
	require.Equal(t, int64(1), winners.Load(), "replicas must claim a charged IQ schedule only once")
	duePlans, err := repo.ListDue(ctx, now)
	require.NoError(t, err)
	require.Empty(t, duePlans)
	next := now.Add(time.Hour)
	require.NoError(t, repo.UpdateAfterRun(ctx, plan.ID, now, next))
	loaded, err := repo.GetByID(ctx, plan.ID)
	require.NoError(t, err)
	require.Equal(t, "candy", loaded.PelicanConfig.QuestionKind)
	acquired, err := repo.ClaimRun(ctx, plan.ID, now, now.Add(time.Minute))
	require.NoError(t, err)
	require.False(t, acquired)
	_, err = db.ExecContext(ctx, "UPDATE scheduled_test_plans SET next_run_at=$2 WHERE id=$1", plan.ID, due)
	require.NoError(t, err)
	acquired, err = repo.ClaimRun(ctx, plan.ID, now, now.Add(time.Minute))
	require.NoError(t, err)
	require.True(t, acquired, "completed plan releases its old lease")
}
