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
		return `{"device_code":"private-device","verification_uri_complete":"https://accounts.feishu.cn/oauth/verify?ticket=x","interval":2,"expire_in":300}`
	})
	registration, err := api.BeginRegistration(context.Background(), "Enterprise login")
	require.NoError(t, err)
	require.Equal(t, 5, registration.Interval)
	require.Equal(t, 300, registration.ExpireIn)
	u, err := url.Parse(registration.VerificationURL)
	require.NoError(t, err)
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
		} `json:"scopes"`
	}
	require.NoError(t, json.Unmarshal(raw, &addons))
	require.False(t, addons.Preset)
	require.ElementsMatch(t, []string{"application:application:patch", "tenant:tenant:readonly"}, addons.Scopes.Tenant)
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
		return `{"code":0,"data":{"open_id":"ou_x","tenant_key":"tenant","email":"admin-set@example.com","name":"Member"}}`
	})
	info, err := api.GetUserInfo(context.Background(), "token")
	require.NoError(t, err)
	require.Equal(t, "tenant", info.TenantKey)
	require.Equal(t, "ou_x", info.OpenID)
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
			require.True(t, strings.HasSuffix(r.URL.Path, "/ability"))
			require.Equal(t, map[string]any{"enable": true, "pc_url": "https://example.com/login", "mobile_url": "https://example.com/login"}, body["web_app"])
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
	for _, uri := range []string{"http://accounts.feishu.cn/x", "https://evil.example/x", "https://accounts.feishu.cn.evil.example/x", "https://user@accounts.feishu.cn/x"} {
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
