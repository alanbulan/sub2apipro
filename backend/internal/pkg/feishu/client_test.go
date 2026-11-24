package feishu

import (
	"compress/gzip"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

type transportFunc func(*http.Request) (*http.Response, error)

func (f transportFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func testClient(fn func(*http.Request) string) *Client {
	return &Client{HTTP: &http.Client{Transport: transportFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(fn(r))), Header: make(http.Header)}, nil
	})}}
}
func TestLoginUsesV3ConfidentialTokenAndNoExtraScopes(t *testing.T) {
	callback := "https://example.com" + CallbackPath
	u, err := url.Parse(BuildAuthorizeURL("cli_test", callback, "random-state"))
	require.NoError(t, err)
	require.Equal(t, "accounts.feishu.cn", u.Host)
	require.Equal(t, "random-state", u.Query().Get("state"))
	require.Equal(t, callback, u.Query().Get("redirect_uri"))
	require.Empty(t, u.Query().Get("scope"))
	api := testClient(func(r *http.Request) string {
		require.Equal(t, TokenURL, r.URL.String())
		require.Equal(t, "POST", r.Method)
		require.Empty(t, r.Header.Get("Authorization"))
		require.Equal(t, "application/x-www-form-urlencoded", r.Header.Get("Content-Type"))
		require.NoError(t, r.ParseForm())
		require.Equal(t, "authorization_code", r.Form.Get("grant_type"))
		require.Equal(t, "secret", r.Form.Get("client_secret"))
		require.Equal(t, callback, r.Form.Get("redirect_uri"))
		require.Equal(t, "one-use-code", r.Form.Get("code"))
		return `{"code":0,"access_token":"token"}`
	})
	token, err := api.ExchangeCode(context.Background(), "cli_test", "secret", callback, "one-use-code")
	require.NoError(t, err)
	require.Equal(t, "token", token)
}
func TestRegistrationUsesMinimalTemplate(t *testing.T) {
	api := testClient(func(r *http.Request) string {
		require.Equal(t, RegistrationURL, r.URL.String())
		require.NoError(t, r.ParseForm())
		require.Equal(t, "begin", r.Form.Get("action"))
		require.Equal(t, "PersonalAgent", r.Form.Get("archetype"))
		require.Equal(t, "open_id", r.Form.Get("request_user_info"))
		return `{"device_code":"private-device","user_code":"ABCD-EFGH","verification_uri":"https://open.feishu.cn/page/launcher","verification_uri_complete":"https://open.feishu.cn/page/launcher?user_code=ABCD-EFGH","interval":2,"expires_in":300}`
	})
	registration, err := api.BeginRegistration(context.Background(), "Enterprise login")
	require.NoError(t, err)
	require.Equal(t, 5, registration.Interval)
	require.Equal(t, 300, registration.ExpireIn)
	u, err := url.Parse(registration.VerificationURL)
	require.NoError(t, err)
	require.Equal(t, "open.feishu.cn", u.Host)
	require.Equal(t, "/page/launcher", u.Path)
	require.Equal(t, "ABCD-EFGH", u.Query().Get("user_code"))
	require.Equal(t, "true", u.Query().Get("createOnly"))
	raw, err := base64.RawURLEncoding.DecodeString(u.Query().Get("addons"))
	require.NoError(t, err)
	reader, err := gzip.NewReader(strings.NewReader(string(raw)))
	require.NoError(t, err)
	defer func() { _ = reader.Close() }()
	raw, err = io.ReadAll(reader)
	require.NoError(t, err)
	var addons struct {
		Preset bool `json:"preset"`
		Scopes struct {
			Tenant []string `json:"tenant"`
			User   []string `json:"user"`
		} `json:"scopes"`
	}
	require.NoError(t, json.Unmarshal(raw, &addons))
	require.False(t, addons.Preset)
	require.ElementsMatch(t, []string{"application:application:patch", "tenant:tenant:readonly"}, addons.Scopes.Tenant)
	require.ElementsMatch(t, []string{"contact:user.email:readonly", "contact:user.employee:readonly"}, addons.Scopes.User)
}

func TestGrantedEmailScopesExcludePendingAndUnrelatedPermissions(t *testing.T) {
	api := testClient(func(r *http.Request) string {
		if r.URL.Path == "/open-apis/auth/v3/tenant_access_token/internal" {
			return `{"code":0,"tenant_access_token":"tenant-token"}`
		}
		require.Equal(t, "/open-apis/application/v6/scopes", r.URL.Path)
		require.Equal(t, "Bearer tenant-token", r.Header.Get("Authorization"))
		return `{"code":0,"data":{"scopes":[{"scope_name":"contact:user.email:readonly","grant_status":1},{"scope_name":"contact:user.employee:readonly","grant_status":2},{"scope_name":"contact:user.phone:readonly","grant_status":1}]}}`
	})
	scopes, err := api.GrantedEmailScopes(context.Background(), "cli_test", "secret")
	require.NoError(t, err)
	require.Equal(t, []string{"contact:user.email:readonly"}, scopes)
	u, err := url.Parse(BuildAuthorizeURL("cli_test", "https://example.com"+CallbackPath, "state", scopes...))
	require.NoError(t, err)
	require.Equal(t, "contact:user.email:readonly", u.Query().Get("scope"))
}

func TestEmailScopeUpgradeChangesOnlyCurrentAppUserPermissions(t *testing.T) {
	api := testClient(func(r *http.Request) string {
		require.Equal(t, http.MethodPatch, r.Method)
		require.Equal(t, "/open-apis/application/v7/applications/cli_existing/config", r.URL.Path)
		var payload map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
		require.Equal(t, map[string]any{"scope": map[string]any{"add_scopes": []any{
			map[string]any{"scope_name": "contact:user.email:readonly", "token_type": "user"},
			map[string]any{"scope_name": "contact:user.employee:readonly", "token_type": "user"},
		}}}, payload)
		return `{"code":0}`
	})
	require.NoError(t, api.ConfigureEmailScopes(context.Background(), "tenant-token", "cli_existing"))
}

func TestRegistrationAcceptsCurrentAndLegacyExpiryFields(t *testing.T) {
	for _, tc := range []struct {
		name   string
		uri    string
		expiry string
		want   int
	}{
		{"current", "https://open.feishu.cn/page/launcher?user_code=ABCD-EFGH", `"expires_in":120`, 120},
		{"live_response_capped_to_setup_ttl", "https://open.feishu.cn/page/launcher?user_code=ABCD-EFGH", `"expires_in":3600`, 600},
		{"legacy_sdk", "https://accounts.feishu.cn/oauth/verify?ticket=x", `"expire_in":300`, 300},
		{"current_takes_precedence", "https://open.feishu.cn/page/launcher?user_code=ABCD-EFGH", `"expires_in":90,"expire_in":300`, 90},
	} {
		t.Run(tc.name, func(t *testing.T) {
			api := testClient(func(*http.Request) string {
				return `{"device_code":"private-device","verification_uri_complete":"` + tc.uri + `",` + tc.expiry + `}`
			})
			registration, err := api.BeginRegistration(context.Background(), "Enterprise login")
			require.NoError(t, err)
			require.Equal(t, tc.want, registration.ExpireIn)
		})
	}
}
func TestUserInfoRejectsInactiveMembersAndMissingTenant(t *testing.T) {
	for _, body := range []string{`{"code":20021,"data":{"open_id":"ou_x","tenant_key":"tenant"}}`, `{"code":20022}`, `{"code":0,"data":{"open_id":"ou_x"}}`, `{"data":{"open_id":"ou_x","tenant_key":"tenant"}}`} {
		t.Run(body, func(t *testing.T) {
			api := testClient(func(r *http.Request) string {
				require.Equal(t, "Bearer token", r.Header.Get("Authorization"))
				return body
			})
			_, err := api.GetUserInfo(context.Background(), "token")
			require.Error(t, err)
		})
	}
	api := testClient(func(*http.Request) string {
		return `{"code":0,"data":{"open_id":"ou_x","tenant_key":"tenant","email":"admin-set@example.com","enterprise_email":"member@company.example","name":"Member"}}`
	})
	info, err := api.GetUserInfo(context.Background(), "token")
	require.NoError(t, err)
	require.Equal(t, "tenant", info.TenantKey)
	require.Equal(t, "ou_x", info.OpenID)
	require.Equal(t, "admin-set@example.com", info.Email)
	require.Equal(t, "member@company.example", info.EnterpriseEmail)
}
func TestConfigurationAddsCallbackAndPublishesWebApp(t *testing.T) {
	calls := 0
	api := testClient(func(r *http.Request) string {
		calls++
		require.Equal(t, "Bearer tenant-token", r.Header.Get("Authorization"))
		var body map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		switch calls {
		case 1:
			require.Equal(t, "PATCH", r.Method)
			require.True(t, strings.HasSuffix(r.URL.Path, "/config"))
			require.Equal(t, map[string]any{"add": map[string]any{"redirect_urls": []any{"https://example.com" + CallbackPath}}}, body["security"])
		case 2:
			require.Equal(t, "PATCH", r.Method)
			require.True(t, strings.HasSuffix(r.URL.Path, "/ability"))
			require.Equal(t, map[string]any{"enable": true, "pc_url": "https://example.com/login", "pc_new_page_open_mode": "browser", "mobile_url": "https://example.com/login"}, body["web_app"])
		case 3:
			require.Equal(t, "POST", r.Method)
			require.True(t, strings.HasSuffix(r.URL.Path, "/publish"))
			require.Equal(t, "web_app", body["pc_default_ability"])
			require.NotEmpty(t, body["remark"])
		}
		return `{"code":0}`
	})
	ctx := context.Background()
	require.NoError(t, api.ConfigureLogin(ctx, "tenant-token", "cli_test", "https://example.com"+CallbackPath))
	require.NoError(t, api.ConfigureWebApp(ctx, "tenant-token", "cli_test", "https://example.com/login"))
	require.NoError(t, api.Publish(ctx, "tenant-token", "cli_test"))
	require.Equal(t, 3, calls)
}

func TestOpenAPIErrorsPreserveCodesWithDiagnosticObjects(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status int
		body   string
		code   string
	}{
		{"new_web_app_missing_open_mode", http.StatusBadRequest, `{"code":210011,"msg":"invalid request","error":{"log_id":"provider-log","troubleshooter":"https://open.feishu.cn/search?secret=do-not-expose"}}`, "210011"},
		{"missing_permission", http.StatusForbidden, `{"code":99991672,"msg":"Access denied","error":{"permission_violations":[{"subject":"application:application:patch"}],"message":"do-not-expose"}}`, "99991672"},
		{"string_code", http.StatusBadRequest, `{"code":"210011","error":{"message":"do-not-expose"}}`, "210011"},
		{"null_diagnostics", http.StatusBadRequest, `{"code":210011,"error":null}`, "210011"},
		{"non_json_http_failure", http.StatusBadGateway, `<html>do-not-expose</html>`, "http_502"},
		{"non_json_success", http.StatusOK, `<html>do-not-expose</html>`, "invalid_response"},
		{"missing_code", http.StatusOK, `{"error":{"message":"do-not-expose"}}`, "missing_result_code"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			api := &Client{HTTP: &http.Client{Transport: transportFunc(func(*http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: tc.status, Body: io.NopCloser(strings.NewReader(tc.body)), Header: make(http.Header)}, nil
			})}}
			err := api.ConfigureWebApp(context.Background(), "tenant-token", "cli_test", "https://example.com/login")
			var providerErr *Error
			require.ErrorAs(t, err, &providerErr)
			require.Equal(t, "configure_web_app", providerErr.Stage)
			require.Equal(t, tc.code, providerErr.Code)
			require.NotContains(t, err.Error(), "do-not-expose")
		})
	}
}

func TestProviderErrorsNeverLeakResponseCredentials(t *testing.T) {
	for _, body := range []string{`{"error":"secret-value"}`, `{"code":"secret-value","client_secret":"do-not-expose"}`, `{"code":999,"msg":"do-not-expose"}`} {
		api := testClient(func(*http.Request) string { return body })
		_, err := api.ExchangeCode(context.Background(), "cli_x", "secret", "https://example.com"+CallbackPath, "code")
		require.Error(t, err)
		require.NotContains(t, err.Error(), "secret-value")
		require.NotContains(t, err.Error(), "do-not-expose")
	}
}
func TestRegistrationRejectsForeignVerificationHostsAndLark(t *testing.T) {
	for _, uri := range []string{"http://accounts.feishu.cn/x", "https://evil.example/x", "https://accounts.feishu.cn.evil.example/x", "https://user@accounts.feishu.cn/x", "http://open.feishu.cn/page/launcher", "https://open.feishu.cn.evil.example/page/launcher", "https://user@open.feishu.cn/page/launcher", "https://open.feishu.cn:8443/page/launcher"} {
		api := testClient(func(*http.Request) string {
			raw, _ := json.Marshal(map[string]any{"device_code": "secret", "verification_uri_complete": uri})
			return string(raw)
		})
		_, err := api.BeginRegistration(context.Background(), "login")
		require.Error(t, err)
	}
	api := testClient(func(*http.Request) string {
		return `{"client_id":"cli_x","client_secret":"secret","user_info":{"tenant_brand":"lark"}}`
	})
	_, err := api.PollRegistration(context.Background(), "device")
	require.Error(t, err)
}

func TestTenantQueryRequiresEnterpriseAndNeverUsesDisplayID(t *testing.T) {
	for _, body := range []string{`{"code":0,"data":{"tenant":{"display_id":"F123","tenant_tag":0}}}`, `{"code":0,"data":{"tenant":{"tenant_key":"personal","tenant_tag":2}}}`, `{"code":0,"data":{"tenant":{"tenant_key":"unknown"}}}`} {
		api := testClient(func(*http.Request) string { return body })
		_, err := api.GetTenant(context.Background(), "token")
		require.Error(t, err)
	}
	api := testClient(func(*http.Request) string {
		return `{"code":0,"data":{"tenant":{"display_id":"F123","tenant_key":"company","tenant_tag":0}}}`
	})
	tenant, err := api.GetTenant(context.Background(), "token")
	require.NoError(t, err)
	require.Equal(t, "company", tenant.Key)
}
func TestDevicePollingHandlesProtocolPendingOverHTTP400(t *testing.T) {
	for _, protocolError := range []string{"authorization_pending", "slow_down"} {
		for _, numericCode := range []string{"", `,"code":20094`, `,"code":"20094"`} {
			for _, status := range []int{http.StatusOK, http.StatusBadRequest} {
				api := &Client{HTTP: &http.Client{Transport: transportFunc(func(*http.Request) (*http.Response, error) {
					return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(`{"error":"` + protocolError + `"` + numericCode + `}`)), Header: make(http.Header)}, nil
				})}}
				result, err := api.PollRegistration(context.Background(), "device")
				require.NoError(t, err)
				require.Equal(t, protocolError, result.Error)
			}
		}
	}
}

func TestDevicePollingPreservesTerminalErrors(t *testing.T) {
	for _, protocolError := range []string{"access_denied", "expired_token"} {
		api := testClient(func(*http.Request) string {
			return `{"error":"` + protocolError + `","code":20094,"error_description":"private-provider-details"}`
		})
		_, err := api.PollRegistration(context.Background(), "device")
		require.EqualError(t, err, "feishu registration_poll failed ("+protocolError+")")
	}
	api := &Client{HTTP: &http.Client{Transport: transportFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusServiceUnavailable, Body: io.NopCloser(strings.NewReader(`{"error":"authorization_pending","code":20094}`)), Header: make(http.Header)}, nil
	})}}
	_, err := api.PollRegistration(context.Background(), "device")
	require.EqualError(t, err, "feishu registration_poll failed (http_503)")

	api = testClient(func(*http.Request) string { return `{"code":20094}` })
	_, err = api.PollRegistration(context.Background(), "device")
	require.EqualError(t, err, "feishu registration_poll failed (20094)")

	api = testClient(func(*http.Request) string { return `{"error":"authorization_pending","code":20094}` })
	_, err = api.BeginRegistration(context.Background(), "Enterprise login")
	require.EqualError(t, err, "feishu registration_begin failed (provider_error)")
}
