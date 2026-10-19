package handler

import (
	"context"
	"crypto/subtle"
	"net/http"
	"strings"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/feishu"
	"github.com/Wei-Shaw/sub2api/internal/pkg/oauth"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

const feishuCookiePath = "/api/v1/auth/oauth/feishu"

func feishuCookie(c *gin.Context, name, value string, age int) {
	http.SetCookie(c.Writer, &http.Cookie{Name: "feishu_oauth_" + name, Value: encodeCookieValue(value), Path: feishuCookiePath, MaxAge: age, HttpOnly: true, Secure: isRequestHTTPS(c), SameSite: http.SameSiteLaxMode})
}
func readFeishuCookie(c *gin.Context, name string) string {
	v, _ := readCookieDecoded(c, "feishu_oauth_"+name)
	return v
}
func (h *AuthHandler) FeishuOAuthStart(c *gin.Context) {
	if !h.requireActionCaptchaForOAuthLoginStart(c) {
		return
	}
	cfg, err := h.settingSvc.GetFeishuOAuthConfig(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	state, err := oauth.GenerateState()
	if err != nil {
		response.InternalError(c, "Failed to create OAuth state")
		return
	}
	browser, err := generateOAuthPendingBrowserSession()
	if err != nil {
		response.InternalError(c, "Failed to create OAuth session")
		return
	}
	intent := normalizeOAuthIntent(c.Query("intent"))
	redirect := sanitizeFrontendRedirectPath(c.Query("redirect"))
	if redirect == "" {
		redirect = "/dashboard"
	}
	bind := ""
	if intent == oauthIntentBindCurrentUser {
		bind, err = h.buildOAuthBindUserCookieFromContext(c)
		if err != nil {
			response.ErrorFrom(c, err)
			return
		}
	}
	for k, v := range map[string]string{"state": state, "config": cfg.Fingerprint(), "redirect": redirect, "intent": intent, "bind_user": bind} {
		feishuCookie(c, k, v, 600)
	}
	setOAuthPendingBrowserCookie(c, browser, isRequestHTTPS(c))
	clearOAuthPendingSessionCookie(c, isRequestHTTPS(c))
	captureOAuthPromoCode(c, isRequestHTTPS(c))
	respondOAuthStart(c, feishu.BuildAuthorizeURL(cfg.AppID, cfg.RedirectURL, state))
}
func (h *AuthHandler) FeishuOAuthCallback(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	c.Header("Referrer-Policy", "no-referrer")
	fail := func(code, message string) { redirectOAuthError(c, feishu.FrontendCallbackPath, code, message, "") }
	defer func() {
		for _, name := range []string{"state", "config", "redirect", "intent", "bind_user"} {
			feishuCookie(c, name, "", -1)
		}
		clearOAuthPromoCodeCookie(c, isRequestHTTPS(c))
	}()
	expected := readFeishuCookie(c, "state")
	if expected == "" || subtle.ConstantTimeCompare([]byte(expected), []byte(c.Query("state"))) != 1 {
		fail("invalid_state", "Invalid OAuth state; start again")
		return
	}
	if c.Query("error") != "" {
		fail("access_denied", "Feishu authorization was not completed")
		return
	}
	cfg, err := h.settingSvc.GetFeishuOAuthConfig(c.Request.Context())
	if err != nil {
		fail("config_error", "Feishu login unavailable")
		return
	}
	if readFeishuCookie(c, "config") != cfg.Fingerprint() {
		fail("config_changed", "Feishu configuration changed; start again")
		return
	}
	browser, _ := readOAuthPendingBrowserCookie(c)
	code := strings.TrimSpace(c.Query("code"))
	if browser == "" || code == "" {
		fail("missing_params", "Missing OAuth browser session or code")
		return
	}
	api := h.feishuClient
	if api == nil {
		api = feishu.NewClient()
	}
	token, err := api.ExchangeCode(c.Request.Context(), cfg.AppID, cfg.AppSecret, cfg.RedirectURL, code)
	if err != nil {
		fail("token_exchange_failed", "Failed to exchange Feishu authorization code")
		return
	}
	info, err := api.GetUserInfo(c.Request.Context(), token)
	if err != nil {
		fail("userinfo_failed", "Failed to verify Feishu member identity")
		return
	}
	if info.TenantKey != cfg.TenantKey {
		fail("tenant_mismatch", "Only members of the configured Feishu enterprise can sign in")
		return
	}
	identity := service.PendingAuthIdentityKey{ProviderType: "feishu", ProviderKey: cfg.IdentityKey(), ProviderSubject: info.OpenID}
	redirect := sanitizeFrontendRedirectPath(readFeishuCookie(c, "redirect"))
	if redirect == "" {
		redirect = "/dashboard"
	}
	intent := normalizeOAuthIntent(readFeishuCookie(c, "intent"))
	var target *int64
	completion := map[string]any{"redirect": redirect}
	if intent == oauthIntentBindCurrentUser {
		uid, err := h.readOAuthBindUserIDFromCookie(c, "feishu_oauth_bind_user")
		if err != nil {
			fail("invalid_state", "Invalid account binding session")
			return
		}
		target = &uid
	} else {
		user, err := h.findOAuthIdentityUser(c.Request.Context(), identity)
		if err != nil {
			fail("session_error", "Failed to resolve Feishu identity")
			return
		}
		if user != nil {
			target = &user.ID
		} else {
			completion["step"] = oauthPendingChoiceStep
			completion["create_account_allowed"] = h.authService.IsRegistrationEnabled(c.Request.Context())
			completion["force_email_on_signup"] = true
			completion["email_binding_required"] = true
		}
	}
	claims := map[string]any{"app_id": cfg.AppID, "tenant_key": cfg.TenantKey, "config_fingerprint": cfg.Fingerprint(), "suggested_display_name": info.Name, "suggested_avatar_url": info.AvatarURL}
	if err = h.createOAuthPendingSession(c, oauthPendingSessionPayload{Intent: intent, Identity: identity, TargetUserID: target, RedirectTo: redirect, BrowserSessionKey: browser, UpstreamIdentityClaims: claims, CompletionResponse: completion}); err != nil {
		fail("session_error", "Failed to continue Feishu login")
		return
	}
	redirectToFrontendCallback(c, feishu.FrontendCallbackPath)
}

// Revalidate on every pending step, including email completion and TOTP. Admin
// changes revoke in-flight Feishu sessions, while ordinary site sessions persist.
func (h *AuthHandler) validateFeishuPending(ctx context.Context, session *dbent.PendingAuthSession) error {
	if session == nil || session.ProviderType != "feishu" {
		return nil
	}
	cfg, err := h.settingSvc.GetFeishuOAuthConfig(ctx)
	if err != nil {
		return err
	}
	if session.ProviderKey != cfg.IdentityKey() || pendingSessionStringValue(session.UpstreamIdentityClaims, "config_fingerprint") != cfg.Fingerprint() || pendingSessionStringValue(session.UpstreamIdentityClaims, "tenant_key") != cfg.TenantKey {
		return infraerrors.BadRequest("FEISHU_CONFIG_CHANGED", "Feishu configuration changed; start sign-in again")
	}
	return nil
}
