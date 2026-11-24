//go:build unit

package service

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/feishu"
	"github.com/stretchr/testify/require"
)

type directoryTestStore struct{ *feishuMemoryStore }

func (s *directoryTestStore) Get(ctx context.Context, key string) ([]byte, error) {
	if _, ok := s.values[key]; !ok {
		return nil, ErrFeishuSetupStateNotFound
	}
	return s.feishuMemoryStore.Get(ctx, key)
}
func (s *directoryTestStore) SaveDirectoryProgress(ctx context.Context, id string, value []byte, ttl time.Duration) (bool, error) {
	var old FeishuDirectoryJob
	if json.Unmarshal(s.values[feishuDirectoryJobKey], &old) != nil || old.ID != id {
		return false, nil
	}
	return true, s.Set(ctx, feishuDirectoryJobKey, value, ttl)
}

func TestFeishuDirectoryUsesExistingAppAndSkipsInactiveMembers(t *testing.T) {
	setup, store, repo, _ := newFeishuSetupTest(t)
	setup.store = &directoryTestStore{store}
	for key, value := range map[string]string{
		SettingKeyFeishuConnectEnabled: "true", SettingKeyFeishuConnectAppID: "cli_existing", SettingKeyFeishuConnectAppSecret: "existing-secret",
		SettingKeyFeishuConnectTenantKey: "company_1", SettingKeyFeishuConnectRedirectURL: "https://example.com" + feishu.CallbackPath,
	} {
		repo.values[key] = value
	}
	setup.api = &feishu.Client{HTTP: &http.Client{Transport: feishuTransport(func(r *http.Request) (*http.Response, error) {
		body := `{"code":0}`
		switch r.URL.Path {
		case "/open-apis/auth/v3/tenant_access_token/internal":
			body = `{"code":0,"tenant_access_token":"tenant-token"}`
		case "/open-apis/application/v7/applications/cli_existing/config":
			var patch map[string]any
			require.NoError(t, json.NewDecoder(r.Body).Decode(&patch))
			require.Equal(t, map[string]any{"contacts_range_type": "all"}, patch["contacts"])
			require.NotContains(t, patch, "web_app")
		case "/open-apis/application/v7/applications/cli_existing/publish":
		case "/open-apis/contact/v3/departments/0/children":
			body = `{"code":0,"data":{"items":[],"has_more":false}}`
		case "/open-apis/contact/v3/users/find_by_department":
			body = `{"code":0,"data":{"items":[{"open_id":"former","name":"Former","status":{"is_activated":true,"is_resigned":true}}],"has_more":false}}`
		default:
			t.Fatal(r.URL.Path)
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}, nil
	})}}
	svc := NewFeishuDirectoryService(nil, setup)
	ctx := context.Background()
	current, err := svc.Current(ctx)
	require.NoError(t, err)
	require.Nil(t, current)
	job := &FeishuDirectoryJob{ID: "task", Status: "preparing", Deadline: time.Now().Add(time.Minute).Unix()}
	require.NoError(t, svc.save(ctx, job))
	require.NoError(t, svc.execute(ctx, job))
	require.Equal(t, "succeeded", job.Status)
	require.Equal(t, 1, job.Skipped)
	require.Zero(t, job.Created)
	// A worker from before the current task cannot write new progress.
	require.Error(t, svc.persist(ctx, &FeishuDirectoryJob{ID: "old"}))
	job.Status = "syncing"
	job.Deadline = time.Now().Add(-time.Second).Unix()
	require.NoError(t, svc.save(ctx, job))
	current, err = svc.Current(ctx)
	require.NoError(t, err)
	require.Equal(t, "failed", current.Status)
	require.Equal(t, "expired", current.ErrorCode)
}
