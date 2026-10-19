package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/feishu"
)

const (
	SettingKeyFeishuConnectEnabled     = "feishu_connect_enabled"
	SettingKeyFeishuConnectAppID       = "feishu_connect_app_id"
	SettingKeyFeishuConnectAppSecret   = "feishu_connect_app_secret"
	SettingKeyFeishuConnectTenantKey   = "feishu_connect_tenant_key"
	SettingKeyFeishuConnectTenantName  = "feishu_connect_tenant_name"
	SettingKeyFeishuConnectRedirectURL = "feishu_connect_redirect_url"
)

type FeishuConnectSettings struct {
	Enabled          bool   `json:"enabled"`
	AppID            string `json:"app_id"`
	AppSecret        string `json:"-"`
	SecretConfigured bool   `json:"app_secret_configured"`
	TenantKey        string `json:"tenant_key"`
	TenantName       string `json:"tenant_name"`
	RedirectURL      string `json:"redirect_url"`
}

func (s *SettingService) GetFeishuSettings(ctx context.Context) (*FeishuConnectSettings, error) {
	values, err := s.settingRepo.GetMultiple(ctx, []string{SettingKeyFeishuConnectEnabled, SettingKeyFeishuConnectAppID, SettingKeyFeishuConnectAppSecret, SettingKeyFeishuConnectTenantKey, SettingKeyFeishuConnectTenantName, SettingKeyFeishuConnectRedirectURL})
	if err != nil {
		return nil, fmt.Errorf("read feishu settings: %w", err)
	}
	secret := strings.TrimSpace(values[SettingKeyFeishuConnectAppSecret])
	return &FeishuConnectSettings{Enabled: values[SettingKeyFeishuConnectEnabled] == "true", AppID: strings.TrimSpace(values[SettingKeyFeishuConnectAppID]), AppSecret: secret, SecretConfigured: secret != "", TenantKey: strings.TrimSpace(values[SettingKeyFeishuConnectTenantKey]), TenantName: strings.TrimSpace(values[SettingKeyFeishuConnectTenantName]), RedirectURL: strings.TrimSpace(values[SettingKeyFeishuConnectRedirectURL])}, nil
}

func ValidateFeishuRedirect(raw string) error {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.Path != feishu.CallbackPath || u.RawPath != "" || u.Opaque != "" || u.ForceQuery {
		return infraerrors.BadRequest("FEISHU_REDIRECT_INVALID", "Feishu callback must be an absolute URL ending with "+feishu.CallbackPath)
	}
	secure := u.Scheme == "https"
	loopback := u.Scheme == "http" && (u.Hostname() == "localhost" || u.Hostname() == "127.0.0.1" || u.Hostname() == "::1")
	if !secure && !loopback {
		return infraerrors.BadRequest("FEISHU_REDIRECT_INVALID", "Feishu callback must use HTTPS")
	}
	return nil
}

var feishuIdentifier = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

func (v *FeishuConnectSettings) Validate() error {
	if v == nil || (!strings.HasPrefix(v.AppID, "cli_") || !feishuIdentifier.MatchString(v.AppID) || !feishuIdentifier.MatchString(v.TenantKey)) || strings.TrimSpace(v.AppSecret) == "" || strings.TrimSpace(v.TenantKey) == "" || strings.ContainsAny(v.AppID+v.TenantKey, "\r\n:") {
		return infraerrors.BadRequest("FEISHU_CONFIG_INVALID", "App ID, App Secret and enterprise tenant_key are required")
	}
	return ValidateFeishuRedirect(v.RedirectURL)
}

func (v *FeishuConnectSettings) IdentityKey() string { return "feishu:" + v.AppID + ":" + v.TenantKey }
func (v *FeishuConnectSettings) Fingerprint() string {
	sum := sha256.Sum256([]byte(strconv.FormatBool(v.Enabled) + "\x00" + v.AppID + "\x00" + v.AppSecret + "\x00" + v.TenantKey + "\x00" + v.RedirectURL))
	return hex.EncodeToString(sum[:])
}

func (s *SettingService) GetFeishuOAuthConfig(ctx context.Context) (*FeishuConnectSettings, error) {
	v, err := s.GetFeishuSettings(ctx)
	if err != nil {
		return nil, err
	}
	if !v.Enabled {
		return nil, infraerrors.NotFound("OAUTH_DISABLED", "Feishu login is disabled")
	}
	if err := v.Validate(); err != nil {
		return nil, err
	}
	return v, nil
}

func (s *SettingService) SaveFeishuSettings(ctx context.Context, v *FeishuConnectSettings) error {
	if v.Enabled {
		if err := v.Validate(); err != nil {
			return err
		}
	}
	values := map[string]string{SettingKeyFeishuConnectEnabled: strconv.FormatBool(v.Enabled), SettingKeyFeishuConnectAppID: v.AppID, SettingKeyFeishuConnectTenantKey: v.TenantKey, SettingKeyFeishuConnectTenantName: v.TenantName, SettingKeyFeishuConnectRedirectURL: v.RedirectURL}
	if v.AppSecret != "" {
		values[SettingKeyFeishuConnectAppSecret] = v.AppSecret
	}
	if err := s.settingRepo.SetMultiple(ctx, values); err != nil {
		return fmt.Errorf("save feishu settings: %w", err)
	}
	if s.onUpdate != nil {
		s.onUpdate()
	}
	return nil
}
