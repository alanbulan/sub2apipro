// Package feishu implements the official Feishu login and app registration APIs.
package feishu

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	AccountsURL          = "https://accounts.feishu.cn"
	APIURL               = "https://open.feishu.cn/open-apis"
	AuthorizeURL         = AccountsURL + "/open-apis/authen/v1/authorize"
	TokenURL             = AccountsURL + "/oauth/v3/token"
	RegistrationURL      = AccountsURL + "/oauth/v1/app/registration"
	CallbackPath         = "/api/v1/auth/oauth/feishu/callback"
	FrontendCallbackPath = "/auth/feishu/callback"
)

// Error deliberately excludes response bodies, which can contain credentials.
type Error struct {
	Stage string
	Code  string
}

func (e *Error) Error() string { return fmt.Sprintf("feishu %s failed (%s)", e.Stage, e.Code) }

type Client struct{ HTTP *http.Client }

func NewClient() *Client {
	return &Client{HTTP: &http.Client{
		Timeout:       15 * time.Second,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse },
	}}
}

func (c *Client) request(ctx context.Context, stage, method, endpoint, token string, body io.Reader, contentType string, out any) error {
	req, err := http.NewRequestWithContext(ctx, method, endpoint, body)
	if err != nil {
		return &Error{stage, "invalid_request"}
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	req.Header.Set("Accept", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return &Error{stage, "network_error"}
	}
	defer func() { _ = resp.Body.Close() }()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20+1))
	if err != nil || len(raw) > 1<<20 {
		return &Error{stage, "invalid_response"}
	}
	var envelope struct {
		Code  json.RawMessage `json:"code"`
		Error json.RawMessage `json:"error"`
	}
	if json.Unmarshal(raw, &envelope) != nil {
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return &Error{stage, "http_" + strconv.Itoa(resp.StatusCode)}
		}
		return &Error{stage, "invalid_response"}
	}
	openAPI := strings.HasPrefix(endpoint, APIURL+"/")
	if openAPI && len(envelope.Code) == 0 {
		return &Error{stage, "missing_result_code"}
	}
	// OAuth uses an error string; OpenAPI uses an error object containing
	// diagnostics. Decode them separately so metadata cannot hide the real code.
	var protocolError string
	if len(envelope.Error) > 0 && string(envelope.Error) != "null" {
		if json.Unmarshal(envelope.Error, &protocolError) != nil && (!openAPI || envelope.Error[0] != '{') {
			return &Error{stage, "provider_error"}
		}
	}
	// Registration uses OAuth protocol errors alongside numeric diagnostic
	// codes. A pending poll is a control response, even with code=20094.
	pendingPoll := stage == "registration_poll" && endpoint == RegistrationURL &&
		(protocolError == "authorization_pending" || protocolError == "slow_down")
	if protocolError != "" && !pendingPoll {
		// Only return known protocol errors; never expose arbitrary provider text.
		switch protocolError {
		case "access_denied", "expired_token", "invalid_client", "invalid_grant":
			return &Error{stage, protocolError}
		default:
			return &Error{stage, "provider_error"}
		}
	}
	if len(envelope.Code) > 0 && !pendingPoll {
		code := strings.Trim(string(envelope.Code), "\"")
		if code != "0" {
			if _, err := strconv.Atoi(code); err != nil {
				code = "provider_error"
			}
			return &Error{stage, code}
		}
	}
	if (resp.StatusCode < 200 || resp.StatusCode >= 300) && (!pendingPoll || resp.StatusCode != http.StatusBadRequest) {
		return &Error{stage, "http_" + strconv.Itoa(resp.StatusCode)}
	}
	if json.Unmarshal(raw, out) != nil {
		return &Error{stage, "invalid_response"}
	}
	return nil
}

func (c *Client) jsonRequest(ctx context.Context, stage, method, endpoint, token string, payload, out any) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return &Error{stage, "invalid_request"}
	}
	return c.request(ctx, stage, method, endpoint, token, bytes.NewReader(raw), "application/json; charset=utf-8", out)
}

func BuildAuthorizeURL(appID, redirect, state string, scopes ...string) string {
	q := url.Values{"client_id": {appID}, "response_type": {"code"}, "redirect_uri": {redirect}, "state": {state}}
	if len(scopes) > 0 {
		q.Set("scope", strings.Join(scopes, " "))
	}
	return AuthorizeURL + "?" + q.Encode()
}

func (c *Client) ExchangeCode(ctx context.Context, appID, secret, redirect, code string) (string, error) {
	form := url.Values{"grant_type": {"authorization_code"}, "client_id": {appID}, "client_secret": {secret}, "redirect_uri": {redirect}, "code": {code}}
	var out struct {
		AccessToken string `json:"access_token"`
	}
	if err := c.request(ctx, "token_exchange", http.MethodPost, TokenURL, "", strings.NewReader(form.Encode()), "application/x-www-form-urlencoded", &out); err != nil {
		return "", err
	}
	if strings.TrimSpace(out.AccessToken) == "" {
		return "", &Error{"token_exchange", "missing_token"}
	}
	return out.AccessToken, nil
}

type UserInfo struct {
	OpenID          string `json:"open_id"`
	TenantKey       string `json:"tenant_key"`
	Name            string `json:"name"`
	AvatarURL       string `json:"avatar_url"`
	Email           string `json:"email"`
	EnterpriseEmail string `json:"enterprise_email"`
}

func (c *Client) GetUserInfo(ctx context.Context, token string) (*UserInfo, error) {
	var out struct {
		Code *int     `json:"code"`
		Data UserInfo `json:"data"`
	}
	if err := c.request(ctx, "user_info", http.MethodGet, APIURL+"/authen/v1/user_info", token, nil, "", &out); err != nil {
		return nil, err
	}
	if out.Code == nil || *out.Code != 0 || strings.TrimSpace(out.Data.OpenID) == "" || strings.TrimSpace(out.Data.TenantKey) == "" {
		return nil, &Error{"user_info", "missing_identity"}
	}
	out.Data.OpenID = strings.TrimSpace(out.Data.OpenID)
	out.Data.TenantKey = strings.TrimSpace(out.Data.TenantKey)
	return &out.Data, nil
}

type Registration struct {
	DeviceCode      string `json:"device_code"`
	VerificationURL string `json:"verification_uri_complete"`
	Interval        int    `json:"interval"`
	ExpireIn        int    `json:"expire_in"`
}

func (c *Client) BeginRegistration(ctx context.Context, name string) (*Registration, error) {
	form := url.Values{"action": {"begin"}, "archetype": {"PersonalAgent"}, "auth_method": {"client_secret"}, "request_user_info": {"open_id"}}
	var response struct {
		Registration
		ExpiresIn *int `json:"expires_in"`
	}
	if err := c.request(ctx, "registration_begin", http.MethodPost, RegistrationURL, "", strings.NewReader(form.Encode()), "application/x-www-form-urlencoded", &response); err != nil {
		return nil, err
	}
	out := response.Registration
	// The live endpoint uses expires_in; older SDKs use expire_in. Keep the
	// existing internal field and accept both, preferring the current protocol.
	if response.ExpiresIn != nil {
		out.ExpireIn = *response.ExpiresIn
	}
	u, err := url.Parse(out.VerificationURL)
	if err != nil || u.Scheme != "https" || (u.Host != "accounts.feishu.cn" && u.Host != "open.feishu.cn") || u.User != nil || out.DeviceCode == "" {
		return nil, &Error{"registration_begin", "invalid_response"}
	}
	// Match the official scene/registration SDK. The minimal base avoids
	// silently requesting bot/chat/document permissions for a login app.
	addons := map[string]any{"preset": false, "scopes": map[string]any{"tenant": []string{"application:application:patch", "tenant:tenant:readonly"}, "user": []string{"contact:user.email:readonly", "contact:user.employee:readonly"}}}
	raw, err := json.Marshal(addons)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	w := gzip.NewWriter(&buf)
	if _, err := w.Write(raw); err != nil {
		return nil, err
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	q := u.Query()
	q.Set("from", "sdk")
	q.Set("tp", "sdk")
	q.Set("source", "go-sdk/sub2apipro-login")
	q.Set("createOnly", "true")
	q.Set("addons", base64.RawURLEncoding.EncodeToString(buf.Bytes()))
	q.Set("name", name)
	q.Set("desc", "企业成员登录")
	u.RawQuery = q.Encode()
	out.VerificationURL = u.String()
	if out.Interval < 5 {
		out.Interval = 5
	}
	if out.ExpireIn <= 0 || out.ExpireIn > 600 {
		out.ExpireIn = 600
	}
	return &out, nil
}

type RegistrationResult struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	UserInfo     *struct {
		OpenID      string `json:"open_id"`
		TenantBrand string `json:"tenant_brand"`
	} `json:"user_info"`
	Error string `json:"error"`
}

func (c *Client) PollRegistration(ctx context.Context, deviceCode string) (*RegistrationResult, error) {
	form := url.Values{"action": {"poll"}, "device_code": {deviceCode}}
	var out RegistrationResult
	if err := c.request(ctx, "registration_poll", http.MethodPost, RegistrationURL, "", strings.NewReader(form.Encode()), "application/x-www-form-urlencoded", &out); err != nil {
		return nil, err
	}
	if out.UserInfo != nil && out.UserInfo.TenantBrand != "" && out.UserInfo.TenantBrand != "feishu" {
		return nil, &Error{"registration_poll", "unsupported_tenant_brand"}
	}
	if (out.ClientID == "") != (out.ClientSecret == "") {
		return nil, &Error{"registration_poll", "incomplete_credentials"}
	}
	return &out, nil
}

func (c *Client) TenantToken(ctx context.Context, appID, secret string) (string, error) {
	var out struct {
		TenantToken string `json:"tenant_access_token"`
	}
	if err := c.jsonRequest(ctx, "tenant_token", http.MethodPost, APIURL+"/auth/v3/tenant_access_token/internal", "", map[string]string{"app_id": appID, "app_secret": secret}, &out); err != nil {
		return "", err
	}
	if out.TenantToken == "" {
		return "", &Error{"tenant_token", "missing_token"}
	}
	return out.TenantToken, nil
}

type Tenant struct {
	Key  string `json:"tenant_key"`
	Name string `json:"name"`
	Tag  *int   `json:"tenant_tag"`
}

func (c *Client) GetTenant(ctx context.Context, token string) (*Tenant, error) {
	var out struct {
		Data struct {
			Tenant Tenant `json:"tenant"`
		} `json:"data"`
	}
	if err := c.request(ctx, "tenant_info", http.MethodGet, APIURL+"/tenant/v2/tenant/query", token, nil, "", &out); err != nil {
		return nil, err
	}
	if strings.TrimSpace(out.Data.Tenant.Key) == "" || out.Data.Tenant.Tag == nil {
		return nil, &Error{"tenant_info", "missing_tenant"}
	}
	if *out.Data.Tenant.Tag != 0 {
		return nil, &Error{"tenant_info", "enterprise_tenant_required"}
	}
	return &out.Data.Tenant, nil
}

// ConfigureLogin only adds this redirect URL; it never replaces other URLs.
func (c *Client) ConfigureLogin(ctx context.Context, token, appID, redirect string) error {
	var out struct{}
	return c.jsonRequest(ctx, "configure_redirect", http.MethodPatch, APIURL+"/application/v7/applications/"+url.PathEscape(appID)+"/config", token,
		map[string]any{"security": map[string]any{"add": map[string]any{"redirect_urls": []string{redirect}}}, "visibility": map[string]any{"is_visible_to_all": true}}, &out)
}

func (c *Client) ConfigureWebApp(ctx context.Context, token, appID, homepage string) error {
	var out struct{}
	// Explicitly set the opening mode: newly registered apps return 210011
	// when it is omitted, despite the documentation marking it optional.
	return c.jsonRequest(ctx, "configure_web_app", http.MethodPatch, APIURL+"/application/v7/applications/"+url.PathEscape(appID)+"/ability", token,
		map[string]any{"web_app": map[string]any{"enable": true, "pc_url": homepage, "pc_new_page_open_mode": "browser", "mobile_url": homepage}}, &out)
}

// ConfigureEmailScopes upgrades the same app; existing open_id bindings survive.
func (c *Client) ConfigureEmailScopes(ctx context.Context, token, appID string) error {
	var out struct{}
	return c.jsonRequest(ctx, "configure_email_scopes", http.MethodPatch, APIURL+"/application/v7/applications/"+url.PathEscape(appID)+"/config", token,
		map[string]any{"scope": map[string]any{"add_scopes": []map[string]string{
			{"scope_name": "contact:user.email:readonly", "token_type": "user"},
			{"scope_name": "contact:user.employee:readonly", "token_type": "user"},
		}}}, &out)
}

// Only request approved permissions. Adding an unapproved scope to the OAuth
// URL breaks even basic sign-in with Feishu error 20027.
func (c *Client) GrantedEmailScopes(ctx context.Context, appID, secret string) ([]string, error) {
	token, err := c.TenantToken(ctx, appID, secret)
	if err != nil {
		return nil, err
	}
	var out struct {
		Data struct {
			Scopes []struct {
				Name   string `json:"scope_name"`
				Status int    `json:"grant_status"`
			} `json:"scopes"`
		} `json:"data"`
	}
	if err := c.request(ctx, "email_scopes", http.MethodGet, APIURL+"/application/v6/scopes", token, nil, "", &out); err != nil {
		return nil, err
	}
	var scopes []string
	seen := make(map[string]bool)
	for _, scope := range out.Data.Scopes {
		if scope.Status == 1 && !seen[scope.Name] && (scope.Name == "contact:user.email:readonly" || scope.Name == "contact:user.employee:readonly") {
			scopes = append(scopes, scope.Name)
			seen[scope.Name] = true
		}
	}
	return scopes, nil
}

func (c *Client) Publish(ctx context.Context, token, appID string) error {
	var out struct{}
	return c.jsonRequest(ctx, "publish", http.MethodPost, APIURL+"/application/v7/applications/"+url.PathEscape(appID)+"/publish", token,
		map[string]string{"pc_default_ability": "web_app", "mobile_default_ability": "web_app", "remark": "启用本企业成员登录", "changelog": "配置企业登录及 OAuth 回调地址"}, &out)
}
