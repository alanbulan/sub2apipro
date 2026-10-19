package handler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"

	"entgo.io/ent"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/authidentity"
	"github.com/Wei-Shaw/sub2api/internal/pkg/authguard"
	"github.com/Wei-Shaw/sub2api/internal/pkg/feishu"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type feishuHandlerTransport func(*http.Request) (*http.Response, error)

func (f feishuHandlerTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func feishuTestSettings() map[string]string {
	return map[string]string{
		service.SettingKeyFeishuConnectEnabled: "true", service.SettingKeyFeishuConnectAppID: "cli_test", service.SettingKeyFeishuConnectAppSecret: "private-secret", service.SettingKeyFeishuConnectTenantKey: "company_1", service.SettingKeyFeishuConnectRedirectURL: "https://example.com" + feishu.CallbackPath,
	}
}
func feishuTestStart(t *testing.T, h *AuthHandler) []*http.Cookie {
	t.Helper()
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest("GET", "https://example.com/api/v1/auth/oauth/feishu/start", nil)
	h.FeishuOAuthStart(c)
	require.Equal(t, 302, rec.Code)
	require.Contains(t, rec.Header().Get("Location"), "accounts.feishu.cn")
	return rec.Result().Cookies()
}
func feishuTestCallback(t *testing.T, h *AuthHandler, cookies []*http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	state := ""
	for _, cookie := range cookies {
		if cookie.Name == "feishu_oauth_state" {
			var err error
			state, err = decodeCookieValue(cookie.Value)
			require.NoError(t, err)
		}
	}
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest("GET", "https://example.com"+feishu.CallbackPath+"?code=code&state="+url.QueryEscape(state), nil)
	for _, cookie := range cookies {
		c.Request.AddCookie(cookie)
	}
	h.FeishuOAuthCallback(c)
	return rec
}
func feishuMockMember(h *AuthHandler, tenant string) {
	h.feishuClient = &feishu.Client{HTTP: &http.Client{Transport: feishuHandlerTransport(func(r *http.Request) (*http.Response, error) {
		body := `{"code":0,"access_token":"user-token"}`
		if strings.HasSuffix(r.URL.Path, "/user_info") {
			raw, _ := json.Marshal(map[string]any{"code": 0, "data": map[string]string{"open_id": "ou_member", "tenant_key": tenant, "name": "员工张三", "avatar_url": "https://example.com/employee.png", "email": "existing@example.com"}})
			body = string(raw)
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}, nil
	})}}
}
func TestFeishuRejectsOtherEnterpriseBeforeCreatingPendingSession(t *testing.T) {
	h, client := newOAuthPendingFlowTestHandlerWithDependencies(t, oauthPendingFlowTestHandlerOptions{settingValues: feishuTestSettings()})
	feishuMockMember(h, "another_company")
	rec := feishuTestCallback(t, h, feishuTestStart(t, h))
	require.Contains(t, rec.Header().Get("Location"), "tenant_mismatch")
	count, err := client.PendingAuthSession.Query().Count(context.Background())
	require.NoError(t, err)
	require.Zero(t, count)
	count, err = client.User.Query().Count(context.Background())
	require.NoError(t, err)
	require.Zero(t, count)
}
func TestFeishuNeverAutoMergesUnverifiedProviderEmail(t *testing.T) {
	h, client := newOAuthPendingFlowTestHandlerWithDependencies(t, oauthPendingFlowTestHandlerOptions{settingValues: feishuTestSettings()})
	ctx := context.Background()
	existing, err := client.User.Create().SetEmail("existing@example.com").SetUsername("Existing").SetPasswordHash("hash").SetRole(service.RoleAdmin).SetStatus(service.StatusActive).Save(ctx)
	require.NoError(t, err)
	feishuMockMember(h, "company_1")
	rec := feishuTestCallback(t, h, feishuTestStart(t, h))
	require.Equal(t, "/auth/feishu/callback", rec.Header().Get("Location"))
	session, err := client.PendingAuthSession.Query().Only(ctx)
	require.NoError(t, err)
	require.NotNil(t, session.TargetUserID)
	require.NotEqual(t, existing.ID, *session.TargetUserID)
	require.Empty(t, session.ResolvedEmail)
	require.Equal(t, "feishu:cli_test:company_1", session.ProviderKey)
	payload, ok := readCompletionResponse(session.LocalFlowState)
	require.True(t, ok)
	require.Empty(t, payload["step"])
	user, err := client.User.Get(ctx, *session.TargetUserID)
	require.NoError(t, err)
	require.Equal(t, service.RoleUser, user.Role)
	require.Equal(t, "员工张三", user.Username)
	require.Equal(t, "feishu", user.SignupSource)
	require.True(t, strings.HasSuffix(user.Email, service.FeishuConnectSyntheticEmailDomain))
	count, err := client.AuthIdentity.Query().Where(authidentity.ProviderTypeEQ("feishu")).Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, count)
}

func feishuTestExchange(t *testing.T, h *AuthHandler, startCookies []*http.Cookie, callback *httptest.ResponseRecorder) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest("POST", "https://example.com/api/v1/auth/oauth/pending/exchange", strings.NewReader(`{}`))
	c.Request.Header.Set("Content-Type", "application/json")
	for _, cookie := range startCookies {
		if cookie.Name == oauthPendingBrowserCookieName {
			c.Request.AddCookie(cookie)
		}
	}
	for _, cookie := range callback.Result().Cookies() {
		if cookie.Name == oauthPendingSessionCookieName {
			c.Request.AddCookie(cookie)
		}
	}
	h.ExchangePendingOAuthCompletion(c)
	return rec
}

func TestFeishuEmployeeAutoProvisionAndImmediateLogin(t *testing.T) {
	values := feishuTestSettings()
	values[service.SettingKeyRegistrationEnabled] = "false"
	values[service.SettingKeyDefaultBalance] = "12.5"
	values[service.SettingKeyDefaultConcurrency] = "4"
	values[service.SettingKeyRegistrationEmailSuffixWhitelist] = `["unrelated.example"]`
	h, client := newOAuthPendingFlowTestHandlerWithDependencies(t, oauthPendingFlowTestHandlerOptions{settingValues: values, invitationEnabled: true, emailVerifyEnabled: true})
	feishuMockMember(h, "company_1")
	start := feishuTestStart(t, h)
	callback := feishuTestCallback(t, h, start)
	require.Equal(t, "/auth/feishu/callback", callback.Header().Get("Location"))
	exchange := feishuTestExchange(t, h, start, callback)
	require.Equal(t, http.StatusOK, exchange.Code, exchange.Body.String())
	require.Contains(t, exchange.Body.String(), `"access_token"`)
	require.Contains(t, exchange.Body.String(), `"refresh_token"`)
	require.NotContains(t, exchange.Body.String(), `"adoption_required":true`)
	require.NotContains(t, exchange.Body.String(), `"email_binding_required":true`)
	require.NotContains(t, exchange.Body.String(), oauthPendingChoiceStep)
	ctx := context.Background()
	u, err := client.User.Query().Only(ctx)
	require.NoError(t, err)
	require.Equal(t, "员工张三", u.Username)
	require.Equal(t, service.RoleUser, u.Role)
	require.Equal(t, 12.5, u.Balance)
	require.Equal(t, 4, u.Concurrency)
	profile, err := h.userService.GetByID(ctx, u.ID)
	require.NoError(t, err)
	require.Equal(t, "https://example.com/employee.png", profile.AvatarURL)
	summaries, err := h.userService.GetProfileIdentitySummaries(ctx, u.ID, profile)
	require.NoError(t, err)
	require.False(t, summaries.Email.Bound, "an internal placeholder is not a verified email")
	require.True(t, summaries.Feishu.Bound)
	// A retry or subsequent login must not create another account or overwrite
	// a local profile/role/balance that an administrator has since changed.
	_, err = client.User.UpdateOneID(u.ID).SetUsername("自定义姓名").SetBalance(3).Save(ctx)
	require.NoError(t, err)
	start = feishuTestStart(t, h)
	callback = feishuTestCallback(t, h, start)
	exchange = feishuTestExchange(t, h, start, callback)
	require.Equal(t, http.StatusOK, exchange.Code, exchange.Body.String())
	require.Contains(t, exchange.Body.String(), `"access_token"`)
	u, err = client.User.Query().Only(ctx)
	require.NoError(t, err)
	require.Equal(t, profile.ID, u.ID)
	require.Equal(t, "自定义姓名", u.Username)
	require.Equal(t, float64(3), u.Balance)
}

func TestFeishuProvisioningIsAtomic(t *testing.T) {
	h, client := newOAuthPendingFlowTestHandlerWithDependencies(t, oauthPendingFlowTestHandlerOptions{settingValues: feishuTestSettings()})
	client.AuthIdentity.Use(func(next ent.Mutator) ent.Mutator {
		return ent.MutateFunc(func(ctx context.Context, mutation ent.Mutation) (ent.Value, error) {
			if mutation.Op().Is(ent.OpCreate) {
				return nil, errors.New("identity storage unavailable")
			}
			return next.Mutate(ctx, mutation)
		})
	})
	feishuMockMember(h, "company_1")
	callback := feishuTestCallback(t, h, feishuTestStart(t, h))
	require.Contains(t, callback.Header().Get("Location"), "account_unavailable")
	ctx := context.Background()
	count, err := client.User.Query().Count(ctx)
	require.NoError(t, err)
	require.Zero(t, count, "failed identity creation must roll back its user")
	count, err = client.PendingAuthSession.Query().Count(ctx)
	require.NoError(t, err)
	require.Zero(t, count)
}

func TestFeishuConcurrentProvisioningAndIdentityIsolation(t *testing.T) {
	h, client := newOAuthPendingFlowTestHandlerWithDependencies(t, oauthPendingFlowTestHandlerOptions{settingValues: feishuTestSettings()})
	ctx := authguard.WithRegistrationIP(context.Background(), "192.0.2.10")
	cfg, err := h.settingSvc.GetFeishuOAuthConfig(ctx)
	require.NoError(t, err)
	info := &feishu.UserInfo{OpenID: "ou_employee", TenantKey: "company_1", Name: "同名员工"}
	const requests = 4
	var wg sync.WaitGroup
	results := make(chan *service.User, requests)
	errCh := make(chan error, requests)
	for i := 0; i < requests; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			user, err := h.authService.EnsureFeishuEnterpriseUser(ctx, cfg, info)
			results <- user
			errCh <- err
		}()
	}
	wg.Wait()
	close(results)
	close(errCh)
	for err := range errCh {
		require.NoError(t, err)
	}
	var userID int64
	for user := range results {
		require.NotNil(t, user)
		if userID == 0 {
			userID = user.ID
		}
		require.Equal(t, userID, user.ID)
	}
	other, err := h.authService.EnsureFeishuEnterpriseUser(ctx, cfg, &feishu.UserInfo{OpenID: "ou_other", TenantKey: info.TenantKey, Name: info.Name})
	require.NoError(t, err)
	require.NotEqual(t, userID, other.ID, "identical names are not identities")
	count, err := client.User.Query().Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 2, count)
}

func TestFeishuBackendModeDoesNotProvisionEmployees(t *testing.T) {
	values := feishuTestSettings()
	values[service.SettingKeyBackendModeEnabled] = "true"
	h, client := newOAuthPendingFlowTestHandlerWithDependencies(t, oauthPendingFlowTestHandlerOptions{settingValues: values})
	feishuMockMember(h, "company_1")
	callback := feishuTestCallback(t, h, feishuTestStart(t, h))
	require.Contains(t, callback.Header().Get("Location"), "account_unavailable")
	count, err := client.User.Query().Count(context.Background())
	require.NoError(t, err)
	require.Zero(t, count)
}

func TestFeishuDisabledAccountCannotBeRecreated(t *testing.T) {
	h, client := newOAuthPendingFlowTestHandlerWithDependencies(t, oauthPendingFlowTestHandlerOptions{settingValues: feishuTestSettings()})
	ctx := context.Background()
	u, err := client.User.Create().SetEmail("disabled@example.com").SetPasswordHash("hash").SetRole(service.RoleUser).SetStatus("disabled").Save(ctx)
	require.NoError(t, err)
	_, err = client.AuthIdentity.Create().SetUserID(u.ID).SetProviderType("feishu").SetProviderKey("feishu:cli_test:company_1").SetProviderSubject("ou_member").Save(ctx)
	require.NoError(t, err)
	feishuMockMember(h, "company_1")
	callback := feishuTestCallback(t, h, feishuTestStart(t, h))
	require.Contains(t, callback.Header().Get("Location"), "account_unavailable")
	count, err := client.User.Query().Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, count)
}
func TestFeishuStateAndConfigurationChangesInvalidateLogin(t *testing.T) {
	h, _ := newOAuthPendingFlowTestHandlerWithDependencies(t, oauthPendingFlowTestHandlerOptions{settingValues: feishuTestSettings()})
	cookies := feishuTestStart(t, h)
	for _, cookie := range cookies {
		if cookie.Name == "feishu_oauth_state" {
			cookie.Value = ""
		}
	}
	rec := feishuTestCallback(t, h, cookies)
	require.Contains(t, rec.Header().Get("Location"), "invalid_state")
	cfg, err := h.settingSvc.GetFeishuOAuthConfig(context.Background())
	require.NoError(t, err)
	pending := &dbent.PendingAuthSession{ProviderType: "feishu", ProviderKey: cfg.IdentityKey(), UpstreamIdentityClaims: map[string]any{"tenant_key": cfg.TenantKey, "config_fingerprint": cfg.Fingerprint()}}
	require.NoError(t, h.validateFeishuPending(context.Background(), pending))
	pending.UpstreamIdentityClaims["config_fingerprint"] = "previous-config"
	require.Error(t, h.validateFeishuPending(context.Background(), pending))
}
func TestFeishuExistingAccountRequiresTOTPBeforeIssuingTokens(t *testing.T) {
	values := feishuTestSettings()
	values[service.SettingKeyTotpEnabled] = "true"
	cache := &oauthPendingFlowTotpCacheStub{}
	h, client := newOAuthPendingFlowTestHandlerWithDependencies(t, oauthPendingFlowTestHandlerOptions{settingValues: values, totpCache: cache})
	ctx := context.Background()
	u, err := client.User.Create().SetEmail("member@example.com").SetPasswordHash("hash").SetRole(service.RoleUser).SetStatus(service.StatusActive).SetTotpEnabled(true).Save(ctx)
	require.NoError(t, err)
	_, err = client.AuthIdentity.Create().SetUserID(u.ID).SetProviderType("feishu").SetProviderKey("feishu:cli_test:company_1").SetProviderSubject("ou_member").Save(ctx)
	require.NoError(t, err)
	feishuMockMember(h, "company_1")
	startCookies := feishuTestStart(t, h)
	callback := feishuTestCallback(t, h, startCookies)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest("POST", "https://example.com/api/v1/auth/oauth/pending/exchange", strings.NewReader(`{}`))
	c.Request.Header.Set("Content-Type", "application/json")
	for _, cookie := range startCookies {
		if cookie.Name == oauthPendingBrowserCookieName {
			c.Request.AddCookie(cookie)
		}
	}
	for _, cookie := range callback.Result().Cookies() {
		if cookie.Name == oauthPendingSessionCookieName {
			c.Request.AddCookie(cookie)
		}
	}
	h.ExchangePendingOAuthCompletion(c)
	require.Equal(t, 200, rec.Code)
	require.Contains(t, rec.Body.String(), `"requires_2fa":true`)
	require.Contains(t, rec.Body.String(), `"temp_token"`)
	require.NotContains(t, rec.Body.String(), `"access_token"`)
}
