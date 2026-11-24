//go:build unit

package service

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/feishu"
	"github.com/stretchr/testify/require"
)

type feishuSettingsRepo struct{ settingWeChatRepoStub }

func (r *feishuSettingsRepo) SetMultiple(_ context.Context, v map[string]string) error {
	for k, value := range v {
		r.values[k] = value
	}
	return nil
}

type feishuMemoryStore struct {
	values map[string][]byte
	locked bool
}

func (s *feishuMemoryStore) Get(_ context.Context, key string) ([]byte, error) {
	v, ok := s.values[key]
	if !ok {
		return nil, errors.New("missing")
	}
	return v, nil
}
func (s *feishuMemoryStore) Set(_ context.Context, key string, v []byte, _ time.Duration) error {
	s.values[key] = v
	return nil
}
func (s *feishuMemoryStore) Lock(context.Context) (func(), error) {
	if s.locked {
		return nil, errors.New("locked")
	}
	s.locked = true
	return func() { s.locked = false }, nil
}

type feishuTransport func(*http.Request) (*http.Response, error)

func (f feishuTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func newFeishuSetupTest(t *testing.T) (*FeishuSetupService, *feishuMemoryStore, *feishuSettingsRepo, *[]string) {
	t.Helper()
	repo := &feishuSettingsRepo{settingWeChatRepoStub{values: map[string]string{SettingKeyFrontendURL: "https://example.com"}}}
	cfg := &config.Config{JWT: config.JWTConfig{Secret: "test-encryption-key"}}
	store := &feishuMemoryStore{values: map[string][]byte{}}
	svc := NewFeishuSetupService(NewSettingService(repo, cfg), store)
	calls := &[]string{}
	svc.api = &feishu.Client{HTTP: &http.Client{Transport: feishuTransport(func(r *http.Request) (*http.Response, error) {
		*calls = append(*calls, r.URL.Path)
		body := `{"code":0}`
		switch r.URL.Path {
		case "/oauth/v1/app/registration":
			require.NoError(t, r.ParseForm())
			if r.Form.Get("action") == "begin" {
				body = `{"device_code":"private-device","verification_uri_complete":"https://open.feishu.cn/page/launcher?user_code=ABCD-EFGH","interval":5,"expires_in":3600}`
			} else {
				body = `{"client_id":"cli_test","client_secret":"private-secret","user_info":{"tenant_brand":"feishu"}}`
			}
		case "/open-apis/auth/v3/tenant_access_token/internal":
			body = `{"code":0,"tenant_access_token":"tenant-token"}`
		case "/open-apis/tenant/v2/tenant/query":
			body = `{"code":0,"data":{"tenant":{"tenant_key":"company_1","name":"Company","tenant_tag":0}}}`
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}, nil
	})}}
	return svc, store, repo, calls
}
func TestFeishuSetupCredentialsStayServerSideAndPollIsRateLimited(t *testing.T) {
	svc, store, _, calls := newFeishuSetupTest(t)
	ctx := context.Background()
	start, err := svc.Start(ctx, 42)
	require.NoError(t, err)
	raw, err := json.Marshal(start)
	require.NoError(t, err)
	require.NotContains(t, string(raw), "private-device")
	_, err = svc.Poll(ctx, start.ID, 99)
	require.Error(t, err)
	result, err := svc.Poll(ctx, start.ID, 42)
	require.NoError(t, err)
	require.Equal(t, "pending", result.Status)
	require.Len(t, *calls, 1)
	state, err := svc.load(ctx, start.ID, 42)
	require.NoError(t, err)
	state.NextPoll = 0
	require.NoError(t, svc.save(ctx, start.ID, state))
	result, err = svc.Poll(ctx, start.ID, 42)
	require.NoError(t, err)
	require.Equal(t, "submitted", result.Status)
	require.Equal(t, "company_1", result.Settings.TenantKey)
	require.True(t, result.Settings.Enabled)
	require.True(t, result.Settings.SecretConfigured)
	raw, err = json.Marshal(result)
	require.NoError(t, err)
	require.NotContains(t, string(raw), "private-secret")
	require.NotContains(t, string(store.values[start.ID]), "private-secret")
	require.NotContains(t, string(store.values[start.ID]), "private-device")
	count := len(*calls)
	_, err = svc.Poll(ctx, start.ID, 42)
	require.NoError(t, err)
	require.Len(t, *calls, count)
	_, err = svc.Start(ctx, 42)
	require.Error(t, err)
}
func TestFeishuSetupNeverReplacesSettingsChangedAfterQR(t *testing.T) {
	svc, _, repo, calls := newFeishuSetupTest(t)
	ctx := context.Background()
	start, err := svc.Start(ctx, 42)
	require.NoError(t, err)
	repo.values[SettingKeyFeishuConnectTenantKey] = "another-company"
	_, err = svc.Poll(ctx, start.ID, 42)
	require.Error(t, err)
	require.Len(t, *calls, 1)
	require.Equal(t, "another-company", repo.values[SettingKeyFeishuConnectTenantKey])
}

func TestFeishuEmailUpgradeRetainsAppCredentialsAndBindings(t *testing.T) {
	svc, _, repo, calls := newFeishuSetupTest(t)
	for key, value := range map[string]string{
		SettingKeyFeishuConnectEnabled: "true", SettingKeyFeishuConnectAppID: "cli_existing", SettingKeyFeishuConnectAppSecret: "existing-secret",
		SettingKeyFeishuConnectTenantKey: "company_1", SettingKeyFeishuConnectRedirectURL: "https://example.com" + feishu.CallbackPath,
	} {
		repo.values[key] = value
	}
	ctx := context.Background()
	before, err := svc.settings.GetFeishuSettings(ctx)
	require.NoError(t, err)
	after, err := svc.EnableEmailSync(ctx)
	require.NoError(t, err)
	require.Equal(t, before.Fingerprint(), after.Fingerprint())
	require.Equal(t, []string{
		"/open-apis/auth/v3/tenant_access_token/internal",
		"/open-apis/application/v7/applications/cli_existing/config",
		"/open-apis/application/v7/applications/cli_existing/publish",
	}, *calls)
	saved, err := svc.settings.GetFeishuSettings(ctx)
	require.NoError(t, err)
	require.Equal(t, before.Fingerprint(), saved.Fingerprint())
}

func TestFeishuSetupKeepsWaitingOnOfficialPendingResponse(t *testing.T) {
	svc, _, repo, _ := newFeishuSetupTest(t)
	ctx := context.Background()
	start, err := svc.Start(ctx, 42)
	require.NoError(t, err)
	require.Contains(t, start.URL, "https://open.feishu.cn/page/launcher?")
	state, err := svc.load(ctx, start.ID, 42)
	require.NoError(t, err)
	state.NextPoll = 0
	require.NoError(t, svc.save(ctx, start.ID, state))
	original := svc.api.HTTP.Transport
	svc.api.HTTP.Transport = feishuTransport(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path == "/oauth/v1/app/registration" {
			return &http.Response{StatusCode: http.StatusBadRequest, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"error":"authorization_pending","error_description":"","code":20094}`))}, nil
		}
		return original.RoundTrip(r)
	})
	result, err := svc.Poll(ctx, start.ID, 42)
	require.NoError(t, err)
	require.Equal(t, "pending", result.Status)
	require.Nil(t, result.Settings)
	require.Empty(t, repo.values[SettingKeyFeishuConnectAppID])
	state, err = svc.load(ctx, start.ID, 42)
	require.NoError(t, err)
	require.Zero(t, state.Stage)
	state.NextPoll = 0
	require.NoError(t, svc.save(ctx, start.ID, state))
	svc.api.HTTP.Transport = original
	result, err = svc.Poll(ctx, start.ID, 42)
	require.NoError(t, err)
	require.Equal(t, "submitted", result.Status)
}

func TestFeishuSetupFailurePreservesOldConfigAndCredentialsForRetry(t *testing.T) {
	svc, _, repo, _ := newFeishuSetupTest(t)
	ctx := context.Background()
	start, err := svc.Start(ctx, 42)
	require.NoError(t, err)
	state, err := svc.load(ctx, start.ID, 42)
	require.NoError(t, err)
	state.NextPoll = 0
	require.NoError(t, svc.save(ctx, start.ID, state))
	original := svc.api.HTTP.Transport
	svc.api.HTTP.Transport = feishuTransport(func(r *http.Request) (*http.Response, error) {
		if strings.HasSuffix(r.URL.Path, "/config") {
			return nil, errors.New("network unavailable")
		}
		return original.RoundTrip(r)
	})
	_, err = svc.Poll(ctx, start.ID, 42)
	require.Error(t, err)
	require.Empty(t, repo.values[SettingKeyFeishuConnectAppID])
	require.Empty(t, repo.values[SettingKeyFeishuConnectEnabled])
	state, err = svc.load(ctx, start.ID, 42)
	require.NoError(t, err)
	require.Equal(t, "private-secret", state.Config.AppSecret)
	require.Equal(t, 1, state.Stage)
	state.NextPoll = 0
	require.NoError(t, svc.save(ctx, start.ID, state))
	svc.api.HTTP.Transport = original
	result, err := svc.Poll(ctx, start.ID, 42)
	require.NoError(t, err)
	require.Equal(t, "submitted", result.Status)
}

func TestFeishuSetupRetriesRejectedWebAppWithoutRegisteringAnotherApp(t *testing.T) {
	svc, _, repo, calls := newFeishuSetupTest(t)
	ctx := context.Background()
	start, err := svc.Start(ctx, 42)
	require.NoError(t, err)
	state, err := svc.load(ctx, start.ID, 42)
	require.NoError(t, err)
	state.NextPoll = 0
	require.NoError(t, svc.save(ctx, start.ID, state))
	original := svc.api.HTTP.Transport
	rejected := false
	svc.api.HTTP.Transport = feishuTransport(func(r *http.Request) (*http.Response, error) {
		if strings.HasSuffix(r.URL.Path, "/ability") && !rejected {
			rejected = true
			return &http.Response{StatusCode: http.StatusBadRequest, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"code":210011,"msg":"invalid request","error":{"log_id":"provider-log","troubleshooter":"do-not-expose"}}`))}, nil
		}
		return original.RoundTrip(r)
	})
	_, err = svc.Poll(ctx, start.ID, 42)
	var providerErr *feishu.Error
	require.ErrorAs(t, err, &providerErr)
	require.Equal(t, "configure_web_app", providerErr.Stage)
	require.Equal(t, "210011", providerErr.Code)
	require.NotContains(t, err.Error(), "do-not-expose")
	require.Empty(t, repo.values[SettingKeyFeishuConnectAppID])
	require.Empty(t, repo.values[SettingKeyFeishuConnectEnabled])
	state, err = svc.load(ctx, start.ID, 42)
	require.NoError(t, err)
	require.Equal(t, 2, state.Stage)
	require.Equal(t, "cli_test", state.Config.AppID)
	require.Equal(t, "private-secret", state.Config.AppSecret)
	state.NextPoll = 0
	require.NoError(t, svc.save(ctx, start.ID, state))
	result, err := svc.Poll(ctx, start.ID, 42)
	require.NoError(t, err)
	require.Equal(t, "submitted", result.Status)
	require.Equal(t, "cli_test", result.Settings.AppID)
	require.Equal(t, "true", repo.values[SettingKeyFeishuConnectEnabled])
	registrationCalls := 0
	for _, path := range *calls {
		if path == "/oauth/v1/app/registration" {
			registrationCalls++
		}
	}
	require.Equal(t, 2, registrationCalls, "retry must reuse the begin/poll credentials")
}

func TestFeishuRedirectAndIdentityIsolation(t *testing.T) {
	valid := &FeishuConnectSettings{Enabled: true, AppID: "cli_test", AppSecret: "secret", TenantKey: "company", RedirectURL: "https://example.com" + feishu.CallbackPath}
	require.NoError(t, valid.Validate())
	original := valid.Fingerprint()
	key := valid.IdentityKey()
	for _, callback := range []string{"http://example.com" + feishu.CallbackPath, "//example.com" + feishu.CallbackPath, "https://user:pass@example.com" + feishu.CallbackPath, "https://example.com/wrong", "https://example.com" + feishu.CallbackPath + "?next=evil", "https://example.com" + feishu.CallbackPath + "#x"} {
		require.Error(t, ValidateFeishuRedirect(callback))
	}
	require.NoError(t, ValidateFeishuRedirect("http://localhost:8080"+feishu.CallbackPath))
	valid.AppID = "cli_other"
	require.NotEqual(t, key, valid.IdentityKey())
	require.NotEqual(t, original, valid.Fingerprint())
	valid.AppID = "cli_../x"
	require.Error(t, valid.Validate())
}

func TestFeishuSetupDerivesCallbackWithoutManualSiteConfiguration(t *testing.T) {
	svc, _, repo, _ := newFeishuSetupTest(t)
	repo.values[SettingKeyFrontendURL] = ""
	result, err := svc.Start(context.Background(), 42, "https://public.example")
	require.NoError(t, err)
	state, err := svc.load(context.Background(), result.ID, 42)
	require.NoError(t, err)
	require.Equal(t, "https://public.example"+feishu.CallbackPath, state.Config.RedirectURL)
}
