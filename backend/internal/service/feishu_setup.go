package service

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/feishu"
)

type FeishuSetupStore interface {
	Get(context.Context, string) ([]byte, error)
	Set(context.Context, string, []byte, time.Duration) error
	Lock(context.Context) (func(), error)
}

// Setup credentials stay server-side, encrypted in Redis, and expire with the QR.
type FeishuSetupService struct {
	settings *SettingService
	store    FeishuSetupStore
	api      *feishu.Client
}

func NewFeishuSetupService(s *SettingService, r FeishuSetupStore) *FeishuSetupService {
	return &FeishuSetupService{s, r, feishu.NewClient()}
}

type feishuSetupState struct {
	Owner    int64
	Device   string
	Baseline string
	Config   FeishuConnectSettings
	Stage    int
	Expires  int64
	NextPoll int64
	Interval int
}
type FeishuSetupResult struct {
	ID        string                 `json:"id,omitempty"`
	URL       string                 `json:"verification_url,omitempty"`
	ExpiresAt int64                  `json:"expires_at,omitempty"`
	Interval  int                    `json:"interval"`
	Status    string                 `json:"status"`
	Settings  *FeishuConnectSettings `json:"settings,omitempty"`
}

func (s *FeishuSetupService) crypt() (cipher.AEAD, error) {
	if s.settings.cfg == nil || s.settings.cfg.JWT.Secret == "" {
		return nil, fmt.Errorf("Feishu setup encryption unavailable")
	}
	key := sha256.Sum256([]byte("sub2apipro/feishu-setup/v1/" + s.settings.cfg.JWT.Secret))
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}
func (s *FeishuSetupService) save(ctx context.Context, id string, st *feishuSetupState) error {
	a, err := s.crypt()
	if err != nil {
		return err
	}
	raw, err := json.Marshal(struct {
		State  *feishuSetupState
		Secret string
	}{st, st.Config.AppSecret})
	if err != nil {
		return err
	}
	nonce := make([]byte, a.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return err
	}
	encrypted := a.Seal(nonce, nonce, raw, []byte(id))
	ttl := time.Until(time.Unix(st.Expires, 0))
	if ttl <= 0 {
		return infraerrors.BadRequest("FEISHU_SETUP_EXPIRED", "QR code expired; start again")
	}
	return s.store.Set(ctx, id, encrypted, ttl)
}
func (s *FeishuSetupService) load(ctx context.Context, id string, owner int64) (*feishuSetupState, error) {
	raw, err := s.store.Get(ctx, id)
	if err != nil {
		return nil, infraerrors.NotFound("FEISHU_SETUP_EXPIRED", "QR code expired; start again")
	}
	a, err := s.crypt()
	if err != nil {
		return nil, err
	}
	if len(raw) < a.NonceSize() {
		return nil, fmt.Errorf("invalid setup state")
	}
	raw, err = a.Open(nil, raw[:a.NonceSize()], raw[a.NonceSize():], []byte(id))
	if err != nil {
		return nil, fmt.Errorf("invalid setup state")
	}
	var stored struct {
		State  *feishuSetupState
		Secret string
	}
	if json.Unmarshal(raw, &stored) != nil || stored.State == nil {
		return nil, fmt.Errorf("invalid setup state")
	}
	st := stored.State
	st.Config.AppSecret = stored.Secret
	if st.Owner != owner || st.Expires <= time.Now().Unix() {
		return nil, infraerrors.NotFound("FEISHU_SETUP_EXPIRED", "QR code expired; start again")
	}
	return st, nil
}

// All setup and manual writes share a distributed lock. A stale QR cannot replace
// settings edited after it was generated, including from another replica/admin.
func (s *FeishuSetupService) Lock(ctx context.Context) (func(), error) { return s.store.Lock(ctx) }
func (s *FeishuSetupService) Start(ctx context.Context, owner int64, browserOrigin ...string) (*FeishuSetupResult, error) {
	release, err := s.Lock(ctx)
	if err != nil {
		return nil, err
	}
	defer release()
	last, _ := s.store.Get(ctx, fmt.Sprintf("owner:%d", owner))
	if len(last) > 0 {
		var active struct {
			ID      string
			Created int64
		}
		if json.Unmarshal(last, &active) == nil && time.Now().Unix()-active.Created < 30 {
			return nil, infraerrors.TooManyRequests("FEISHU_SETUP_RATE_LIMIT", "Wait 30 seconds before creating another QR")
		}
	}
	current, err := s.settings.GetFeishuSettings(ctx)
	if err != nil {
		return nil, err
	}
	base := strings.TrimRight(s.settings.GetFrontendURL(ctx), "/")
	if base == "" && len(browserOrigin) > 0 {
		base = strings.TrimRight(strings.TrimSpace(browserOrigin[0]), "/")
	}
	callback := base + feishu.CallbackPath
	if err = ValidateFeishuRedirect(callback); err != nil {
		return nil, infraerrors.BadRequest("FEISHU_SITE_URL_REQUIRED", "A public HTTPS site origin is required for Feishu callbacks")
	}
	name, _ := s.settings.settingRepo.GetValue(ctx, SettingKeySiteName)
	name = strings.TrimSpace(name)
	if name == "" {
		name = "企业登录"
	}
	runes := []rune(name)
	if len(runes) > 40 {
		name = string(runes[:40])
	}
	registration, err := s.api.BeginRegistration(ctx, name)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, 32)
	if _, err = rand.Read(nonce); err != nil {
		return nil, err
	}
	id := base64.RawURLEncoding.EncodeToString(nonce)
	st := &feishuSetupState{Owner: owner, Device: registration.DeviceCode, Baseline: current.Fingerprint(), Config: FeishuConnectSettings{RedirectURL: callback}, Expires: time.Now().Unix() + int64(registration.ExpireIn), Interval: registration.Interval, NextPoll: time.Now().Unix() + int64(registration.Interval)}
	if err = s.save(ctx, id, st); err != nil {
		return nil, err
	}
	active, _ := json.Marshal(struct {
		ID      string
		Created int64
	}{id, time.Now().Unix()})
	if err = s.store.Set(ctx, fmt.Sprintf("owner:%d", owner), active, 10*time.Minute); err != nil {
		return nil, err
	}
	return &FeishuSetupResult{ID: id, URL: registration.VerificationURL, ExpiresAt: st.Expires, Interval: st.Interval, Status: "pending"}, nil
}
func (s *FeishuSetupService) Poll(ctx context.Context, id string, owner int64) (*FeishuSetupResult, error) {
	ctx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()
	release, err := s.Lock(ctx)
	if err != nil {
		return nil, err
	}
	defer release()
	st, err := s.load(ctx, id, owner)
	if err != nil {
		return nil, err
	}
	active, err := s.store.Get(ctx, fmt.Sprintf("owner:%d", owner))
	var latest struct {
		ID      string
		Created int64
	}
	if err != nil || json.Unmarshal(active, &latest) != nil || latest.ID != id {
		return nil, infraerrors.BadRequest("FEISHU_SETUP_REPLACED", "This QR was replaced; use the latest one")
	}
	result := &FeishuSetupResult{Interval: st.Interval, ExpiresAt: st.Expires, Status: "pending"}
	if st.Stage == 5 {
		result.Status = "submitted"
		result.Settings = &st.Config
		return result, nil
	}
	current, err := s.settings.GetFeishuSettings(ctx)
	if err != nil {
		return nil, err
	}
	completedConfig := st.Config
	completedConfig.Enabled = true
	if st.Stage == 4 && current.Fingerprint() == completedConfig.Fingerprint() {
		st.Stage = 5
		result.Status = "submitted"
		result.Settings = current
		return result, s.save(ctx, id, st)
	}
	if current.Fingerprint() != st.Baseline {
		return nil, infraerrors.Conflict("FEISHU_SETUP_CONFIG_CHANGED", "Settings changed; start a new QR setup")
	}
	if time.Now().Unix() < st.NextPoll {
		return result, nil
	}
	st.NextPoll = time.Now().Unix() + int64(st.Interval)
	if err = s.save(ctx, id, st); err != nil {
		return nil, err
	}
	if st.Stage == 0 {
		registration, err := s.api.PollRegistration(ctx, st.Device)
		if err != nil {
			return nil, err
		}
		if registration.Error == "slow_down" {
			st.Interval += 5
			st.NextPoll = time.Now().Unix() + int64(st.Interval)
			result.Interval = st.Interval
			return result, s.save(ctx, id, st)
		}
		if registration.Error == "authorization_pending" {
			return result, nil
		}
		if registration.ClientID == "" && registration.ClientSecret == "" {
			return result, nil
		}
		if registration.ClientID == "" || registration.ClientSecret == "" {
			return nil, fmt.Errorf("Feishu registration returned incomplete credentials")
		}
		st.Config.AppID = registration.ClientID
		st.Config.AppSecret = registration.ClientSecret
		st.Config.SecretConfigured = true
		st.Stage = 1
		if err = s.save(ctx, id, st); err != nil {
			return nil, err
		}
	}
	token, err := s.api.TenantToken(ctx, st.Config.AppID, st.Config.AppSecret)
	if err != nil {
		return nil, err
	}
	if st.Config.TenantKey == "" {
		tenant, err := s.api.GetTenant(ctx, token)
		if err != nil {
			return nil, err
		}
		st.Config.TenantKey = tenant.Key
		st.Config.TenantName = tenant.Name
		if err = st.Config.Validate(); err != nil {
			return nil, err
		}
		if err = s.save(ctx, id, st); err != nil {
			return nil, err
		}
	}
	if st.Stage == 1 {
		if err = s.api.ConfigureLogin(ctx, token, st.Config.AppID, st.Config.RedirectURL); err != nil {
			return nil, err
		}
		st.Stage = 2
		if err = s.save(ctx, id, st); err != nil {
			return nil, err
		}
	}
	if st.Stage == 2 {
		base := strings.TrimSuffix(st.Config.RedirectURL, feishu.CallbackPath)
		if err = s.api.ConfigureWebApp(ctx, token, st.Config.AppID, base+"/login"); err != nil {
			return nil, err
		}
		st.Stage = 3
		if err = s.save(ctx, id, st); err != nil {
			return nil, err
		}
	}
	if st.Stage == 3 {
		if err = s.api.Publish(ctx, token, st.Config.AppID); err != nil {
			return nil, err
		}
		st.Stage = 4
		if err = s.save(ctx, id, st); err != nil {
			return nil, err
		}
	}
	st.Config.Enabled = true
	if err = s.settings.SaveFeishuSettings(ctx, &st.Config); err != nil {
		return nil, err
	}
	st.Stage = 5
	if err = s.save(ctx, id, st); err != nil {
		return nil, err
	}
	result.Status = "submitted"
	result.Settings = &st.Config
	return result, nil
}
