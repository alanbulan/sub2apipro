package service

import (
	"context"
	"encoding/json"
	"math"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/authidentity"
	"github.com/Wei-Shaw/sub2api/ent/setting"
	dbuser "github.com/Wei-Shaw/sub2api/ent/user"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/feishu"
)

const feishuKeyConfigKey = "feishu_key_delivery_config"
const feishuKeyActiveJob = "feishu_key_delivery_active"
const feishuKeyJobPrefix = "feishu_key_job:"

var ErrFeishuKeyInvalid = infraerrors.BadRequest("FEISHU_KEY_INVALID", "Check request parameters, delivery settings and active groups")
var ErrFeishuKeyStale = infraerrors.Conflict("FEISHU_KEY_STALE", "Configuration or recipient changed; generate a new preview")

type FeishuKeyDeliveryConfig struct {
	Version           int64   `json:"version"`
	GroupIDs          []int64 `json:"group_ids"`
	InitialBalance    float64 `json:"initial_balance"`
	Quota             float64 `json:"quota"`
	ExpiresInDays     int     `json:"expires_in_days"`
	Concurrency       int     `json:"concurrency"`
	RateLimit5h       float64 `json:"rate_limit_5h"`
	RateLimit1d       float64 `json:"rate_limit_1d"`
	RateLimit7d       float64 `json:"rate_limit_7d"`
	AllowContactEmail bool    `json:"allow_contact_email"`
	NameSuffix        string  `json:"name_suffix"`
	Subject           string  `json:"subject"`
	Introduction      string  `json:"introduction"`
	APIURL            string  `json:"api_url"`
}

type FeishuKeyDeliverySettings struct {
	Config         FeishuKeyDeliveryConfig `json:"config"`
	SMTPConfigured bool                    `json:"smtp_configured"`
}

type FeishuKeyDeliveryRow struct {
	UserID     int64      `json:"user_id"`
	Name       string     `json:"name"`
	Email      string     `json:"email"`
	Status     string     `json:"status"`
	Reason     string     `json:"reason,omitempty"`
	KeyID      int64      `json:"key_id,omitempty"`
	Reuse      bool       `json:"reuse"`
	Grant      float64    `json:"grant"`
	Balance    float64    `json:"balance"`
	GroupNames []string   `json:"group_names"`
	Quota      float64    `json:"quota"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
}

type FeishuKeyDeliveryJob struct {
	ID         string                 `json:"id"`
	Status     string                 `json:"status"`
	Rows       []FeishuKeyDeliveryRow `json:"rows"`
	Ready      int                    `json:"ready"`
	Sent       int                    `json:"sent"`
	Failed     int                    `json:"failed"`
	Skipped    int                    `json:"skipped"`
	Uncertain  int                    `json:"uncertain"`
	TotalGrant float64                `json:"total_grant"`
	CreatedAt  int64                  `json:"created_at"`
	ExpiresAt  int64                  `json:"expires_at"`
}

// Internal batch data never leaves the administrator API as raw JSON. It stores
// identities and key IDs, never full keys, SMTP credentials or email bodies.
type feishuKeyRecipient struct {
	FeishuKeyDeliveryRow
	OpenID string `json:"open_id"`
}
type feishuKeyBatch struct {
	ID          string
	Status      string
	Rows        []feishuKeyRecipient
	Config      FeishuKeyDeliveryConfig
	Fingerprint string
	ProviderKey string
	ActorID     int64
	CreatedAt   int64
	ExpiresAt   int64
	Worker      string
	LeaseUntil  int64
}
type feishuKeyReceipt struct {
	KeyID    int64
	UserID   int64
	Grant    float64
	IssuedAt int64
	ActorID  int64
}

type FeishuKeyDeliveryService struct {
	billingCache *BillingCacheService
	directory    *FeishuDirectoryService
	keys         *APIKeyService
	email        *EmailService
	client       *dbent.Client
	cancel       context.CancelFunc
	wg           sync.WaitGroup
	stopOnce     sync.Once
	workerID     string
	// Injectable for SMTP contract tests. Production uses the shared mail service.
	send func(context.Context, string, string, string) error
}

func NewFeishuKeyDeliveryService(directory *FeishuDirectoryService, keys *APIKeyService, email *EmailService, caches ...*BillingCacheService) *FeishuKeyDeliveryService {
	s := &FeishuKeyDeliveryService{directory: directory, keys: keys, email: email}
	if len(caches) > 0 {
		s.billingCache = caches[0]
	}
	if directory != nil && directory.auth != nil {
		s.client = directory.auth.entClient
	}
	if email != nil {
		s.send = email.SendEmailContext
	}
	s.workerID, _ = randomHexString(16)
	return s
}

func (s *FeishuKeyDeliveryService) available() bool {
	return s != nil && s.client != nil && s.directory != nil && s.directory.setup != nil && s.keys != nil && s.email != nil && s.workerID != ""
}

func feishuReadJSON(ctx context.Context, client *dbent.Client, key string, dst any) (bool, error) {
	v, err := client.Setting.Query().Where(setting.KeyEQ(key)).Only(ctx)
	if dbent.IsNotFound(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if err := json.Unmarshal([]byte(v.Value), dst); err != nil {
		return false, ErrServiceUnavailable
	}
	return true, nil
}

func feishuWriteJSON(ctx context.Context, client *dbent.Client, key string, value any) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return client.Setting.Create().SetKey(key).SetValue(string(raw)).OnConflictColumns(setting.FieldKey).UpdateNewValues().Exec(ctx)
}

// Transactions serialize all batch transitions across replicas, and serialize
// the per-user receipt with key creation and the initial balance grant.
func (s *FeishuKeyDeliveryService) transaction(ctx context.Context, fn func(context.Context, *dbent.Client) error) error {
	tx, err := s.client.Tx(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	txCtx := dbent.NewTxContext(ctx, tx)
	release, err := lockAuthPendingIdentityKeys(txCtx, tx.Client(), "feishu-key-delivery")
	if err != nil {
		return err
	}
	defer release()
	if err := fn(txCtx, tx.Client()); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *FeishuKeyDeliveryService) Config(ctx context.Context) (*FeishuKeyDeliverySettings, error) {
	if !s.available() {
		return nil, ErrServiceUnavailable
	}
	cfg := FeishuKeyDeliveryConfig{GroupIDs: []int64{}, NameSuffix: "企业发放", Subject: "您的企业 API Key", Introduction: "以下是您的专属 API Key，请妥善保管，不要转发。"}
	_, err := feishuReadJSON(ctx, s.client, feishuKeyConfigKey, &cfg)
	if err != nil {
		return nil, err
	}
	smtp, smtpErr := s.email.GetSMTPConfig(ctx)
	configured := smtpErr == nil && validFeishuContactEmail(smtp.From) != "" && smtp.Port > 0 && smtp.Port <= 65535 && smtp.Username != "" && smtp.Password != ""
	return &FeishuKeyDeliverySettings{Config: cfg, SMTPConfigured: configured}, nil
}

func validateFeishuKeyProvisionConfig(v *FeishuKeyDeliveryConfig) error {
	if v == nil || len(v.GroupIDs) != 1 {
		return ErrFeishuKeyInvalid
	}
	if v.GroupIDs[0] <= 0 {
		return ErrFeishuKeyInvalid
	}
	for _, n := range []float64{v.InitialBalance, v.Quota, v.RateLimit5h, v.RateLimit1d, v.RateLimit7d} {
		if math.IsNaN(n) || math.IsInf(n, 0) || n < 0 || n > 1e9 {
			return ErrFeishuKeyInvalid
		}
	}
	if v.InitialBalance != 0 && v.InitialBalance < 1e-8 {
		return ErrFeishuKeyInvalid
	}
	v.InitialBalance = math.Round(v.InitialBalance*1e8) / 1e8
	v.NameSuffix = strings.TrimSpace(v.NameSuffix)
	if v.NameSuffix == "" || len(v.NameSuffix) > 100 || v.ExpiresInDays < 0 || v.ExpiresInDays > 3650 || v.Concurrency != 0 {
		return ErrFeishuKeyInvalid
	}
	return nil
}

func validateFeishuKeyConfig(v *FeishuKeyDeliveryConfig) error {
	if err := validateFeishuKeyProvisionConfig(v); err != nil {
		return err
	}
	u, err := url.Parse(strings.TrimSpace(v.APIURL))
	if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || strings.ContainsAny(v.APIURL, "\r\n") {
		return ErrFeishuKeyInvalid
	}
	v.APIURL = strings.TrimRight(u.String(), "/")
	v.Subject, v.NameSuffix = strings.TrimSpace(v.Subject), strings.TrimSpace(v.NameSuffix)
	if v.Subject == "" || len(v.Subject) > 200 || strings.ContainsAny(v.Subject, "\r\n") || len(v.Introduction) > 4000 {
		return ErrFeishuKeyInvalid
	}
	return nil
}

func (s *FeishuKeyDeliveryService) SaveConfig(ctx context.Context, cfg FeishuKeyDeliveryConfig) (*FeishuKeyDeliverySettings, error) {
	if !s.available() {
		return nil, ErrServiceUnavailable
	}
	if err := validateFeishuKeyConfig(&cfg); err != nil {
		return nil, err
	}
	for _, id := range cfg.GroupIDs {
		g, err := s.keys.groupRepo.GetByID(ctx, id)
		if err != nil || !g.IsActive() {
			return nil, ErrFeishuKeyInvalid
		}
	}
	err := s.transaction(ctx, func(ctx context.Context, c *dbent.Client) error {
		var old FeishuKeyDeliveryConfig
		if _, err := feishuReadJSON(ctx, c, feishuKeyConfigKey, &old); err != nil {
			return err
		}
		if old.Version != cfg.Version {
			return ErrFeishuKeyStale
		}
		cfg.Version++
		return feishuWriteJSON(ctx, c, feishuKeyConfigKey, &cfg)
	})
	if err != nil {
		return nil, err
	}
	return s.Config(ctx)
}

func (b *feishuKeyBatch) view() *FeishuKeyDeliveryJob {
	v := &FeishuKeyDeliveryJob{ID: b.ID, Status: b.Status, CreatedAt: b.CreatedAt, ExpiresAt: b.ExpiresAt, Rows: make([]FeishuKeyDeliveryRow, 0, len(b.Rows))}
	for _, r := range b.Rows {
		v.Rows = append(v.Rows, r.FeishuKeyDeliveryRow)
		switch r.Status {
		case "ready", "pending", "prepared", "sending":
			v.Ready++
		case "sent":
			v.Sent++
		case "failed":
			v.Failed++
		case "uncertain":
			v.Uncertain++
		default:
			v.Skipped++
		}
		if r.Status != "skipped" {
			v.TotalGrant += r.Grant
		}
	}
	v.TotalGrant = math.Round(v.TotalGrant*1e8) / 1e8
	return v
}

func feishuRecipientEmail(m feishu.DirectoryMember, allowContact bool) string {
	if email := validFeishuContactEmail(m.EnterpriseEmail); email != "" {
		return email
	}
	if allowContact {
		return validFeishuContactEmail(m.Email)
	}
	return ""
}

func feishuKeyReceiptKey(userID int64) string {
	return "feishu_key_issuance:" + strconv.FormatInt(userID, 10)
}

func (s *FeishuKeyDeliveryService) Preview(ctx context.Context, actor int64, selected []int64, all bool) (*FeishuKeyDeliveryJob, error) {
	if !s.available() {
		return nil, ErrServiceUnavailable
	}
	if (!all && len(selected) == 0) || len(selected) > 500 || (all && len(selected) > 0) {
		return nil, ErrFeishuKeyInvalid
	}
	wanted := map[int64]bool{}
	for _, id := range selected {
		if id <= 0 {
			return nil, ErrFeishuKeyInvalid
		}
		wanted[id] = true
	}
	settings, err := s.Config(ctx)
	if err != nil {
		return nil, err
	}
	if err = validateFeishuKeyConfig(&settings.Config); err != nil {
		return nil, err
	}
	cfg, err := s.directory.setup.settings.GetFeishuOAuthConfig(ctx)
	if err != nil {
		return nil, err
	}
	token, err := s.directory.setup.api.TenantToken(ctx, cfg.AppID, cfg.AppSecret)
	if err != nil {
		return nil, err
	}
	members, err := s.directory.setup.api.DirectoryMembers(ctx, token)
	if err != nil {
		return nil, err
	}
	identities, err := s.client.AuthIdentity.Query().Where(authidentity.ProviderTypeEQ("feishu"), authidentity.ProviderKeyEQ(cfg.IdentityKey())).All(ctx)
	if err != nil {
		return nil, err
	}
	owners := map[string]int64{}
	ownerCount := map[int64]int{}
	for _, i := range identities {
		owners[i.ProviderSubject] = i.UserID
		ownerCount[i.UserID]++
	}
	id, err := randomHexString(20)
	if err != nil {
		return nil, err
	}
	b := &feishuKeyBatch{ID: id, Status: "preview", Config: settings.Config, Fingerprint: cfg.Fingerprint(), ProviderKey: cfg.IdentityKey(), ActorID: actor, CreatedAt: time.Now().Unix(), ExpiresAt: time.Now().Add(15 * time.Minute).Unix()}
	for _, m := range members {
		uid := owners[m.OpenID]
		if !all && !wanted[uid] {
			continue
		}
		delete(wanted, uid)
		r := feishuKeyRecipient{OpenID: m.OpenID, FeishuKeyDeliveryRow: FeishuKeyDeliveryRow{UserID: uid, Name: m.Name, Email: feishuRecipientEmail(m, b.Config.AllowContactEmail), Status: "ready"}}
		r.Reason = s.evaluateRecipient(ctx, s.client, b, &r, m, true)
		if ownerCount[uid] > 1 {
			r.Reason = "identity_mismatch"
		}
		if r.Reason != "" {
			r.Status = "skipped"
		}
		b.Rows = append(b.Rows, r)
	}
	for uid := range wanted {
		b.Rows = append(b.Rows, feishuKeyRecipient{FeishuKeyDeliveryRow: FeishuKeyDeliveryRow{UserID: uid, Status: "skipped", Reason: "identity_mismatch"}})
	}
	if len(b.Rows) > 500 {
		return nil, infraerrors.BadRequest("FEISHU_KEY_BATCH_LIMIT", "Select at most 500 employees per batch")
	}
	// Duplicate company mailboxes cannot establish a unique recipient.
	mailCount := map[string]int{}
	for _, m := range members {
		if m.Active() {
			if email := feishuRecipientEmail(m, b.Config.AllowContactEmail); email != "" {
				mailCount[email]++
			}
		}
	}
	for i := range b.Rows {
		if mailCount[b.Rows[i].Email] > 1 {
			b.Rows[i].Status, b.Rows[i].Reason = "skipped", "email_conflict"
		}
	}
	sort.Slice(b.Rows, func(i, j int) bool { return b.Rows[i].UserID < b.Rows[j].UserID })
	current, err := s.directory.setup.settings.GetFeishuOAuthConfig(ctx)
	if err != nil || current.Fingerprint() != b.Fingerprint {
		return nil, ErrFeishuKeyStale
	}
	if err = feishuWriteJSON(ctx, s.client, feishuKeyJobPrefix+id, b); err != nil {
		return nil, err
	}
	return b.view(), nil
}

// All checks are repeated immediately before provisioning and SMTP. New keys
// may add explicitly selected standard groups; resends never alter old keys.
func (s *FeishuKeyDeliveryService) evaluateRecipient(ctx context.Context, c *dbent.Client, b *feishuKeyBatch, r *feishuKeyRecipient, m feishu.DirectoryMember, allowNew bool) string {
	if !m.Active() {
		return "inactive_employee"
	}
	if r.UserID == 0 {
		return "sync_required"
	}
	if r.Email == "" {
		return "email_missing"
	}
	conflict, err := c.User.Query().Where(dbuser.EmailEQ(r.Email), dbuser.IDNEQ(r.UserID), dbuser.DeletedAtIsNil()).Exist(ctx)
	if err != nil {
		return "service_error"
	}
	if conflict {
		return "email_conflict"
	}
	conflict, err = c.AuthIdentity.Query().Where(authidentity.ProviderTypeEQ("email"), authidentity.ProviderSubjectEQ(r.Email), authidentity.UserIDNEQ(r.UserID), authidentity.VerifiedAtNotNil()).Exist(ctx)
	if err != nil {
		return "service_error"
	}
	if conflict {
		return "email_conflict"
	}
	return s.evaluateEmployee(ctx, c, b, r, m, allowNew)
}

// Key provisioning uses current Feishu membership, never an email as identity.
// API-only issuance deliberately does not require a deliverable mailbox.
func (s *FeishuKeyDeliveryService) evaluateEmployee(ctx context.Context, c *dbent.Client, b *feishuKeyBatch, r *feishuKeyRecipient, m feishu.DirectoryMember, allowNew bool) string {
	if !m.Active() {
		return "inactive_employee"
	}
	if r.UserID == 0 {
		return "sync_required"
	}
	u, err := s.keys.userRepo.GetByID(ctx, r.UserID)
	if err != nil || !u.IsActive() || u.Role != RoleUser {
		return "inactive_user"
	}
	r.Balance = u.Balance
	var receipt feishuKeyReceipt
	ok, err := feishuReadJSON(ctx, c, feishuKeyReceiptKey(u.ID), &receipt)
	if err != nil {
		return "service_error"
	}
	ids := b.Config.GroupIDs
	r.Grant = b.Config.InitialBalance
	r.Quota = b.Config.Quota
	if ok {
		k, err := s.keys.apiKeyRepo.GetByID(ctx, receipt.KeyID)
		if err != nil || k.UserID != u.ID || !k.IsActive() || k.IsExpired() || k.IsQuotaExhausted() {
			return "key_unavailable"
		}
		if k.RateLimit5h > 0 && k.EffectiveUsage5h() >= k.RateLimit5h || k.RateLimit1d > 0 && k.EffectiveUsage1d() >= k.RateLimit1d || k.RateLimit7d > 0 && k.EffectiveUsage7d() >= k.RateLimit7d {
			return "key_rate_limited"
		}
		if len(k.IPWhitelist) > 0 || len(k.IPBlacklist) > 0 {
			return "key_ip_restricted"
		}
		r.Reuse, r.KeyID, r.Grant, r.Quota, r.ExpiresAt = true, k.ID, 0, k.Quota, k.ExpiresAt
		ids = k.ConfiguredGroupIDs()
	} else if !allowNew {
		return "key_unavailable"
	}
	if len(ids) == 0 {
		return "group_unavailable"
	}
	r.GroupNames = []string{}
	standard := false
	for _, id := range ids {
		g, err := s.keys.groupRepo.GetByID(ctx, id)
		if err != nil || !g.IsActive() {
			return "group_unavailable"
		}
		if (g.IsSubscriptionType() || r.Reuse) && !s.keys.canUserBindGroup(ctx, u, g) {
			return "group_not_allowed"
		}
		if !g.IsSubscriptionType() {
			standard = true
		}
		r.GroupNames = append(r.GroupNames, g.Name)
	}
	minimum := 0.0
	if s.keys.cfg != nil {
		minimum = s.keys.cfg.Billing.MinimumBalanceReserve
	}
	if standard && u.Balance+r.Grant <= minimum {
		return "balance_insufficient"
	}
	return ""
}

func (s *FeishuKeyDeliveryService) Current(ctx context.Context) (*FeishuKeyDeliveryJob, error) {
	if !s.available() {
		return nil, ErrServiceUnavailable
	}
	var id string
	ok, err := feishuReadJSON(ctx, s.client, feishuKeyActiveJob, &id)
	if err != nil || !ok {
		return nil, err
	}
	var b feishuKeyBatch
	ok, err = feishuReadJSON(ctx, s.client, feishuKeyJobPrefix+id, &b)
	if err != nil || !ok {
		return nil, err
	}
	return b.view(), nil
}

func (s *FeishuKeyDeliveryService) Start(ctx context.Context, actor int64, id string) (*FeishuKeyDeliveryJob, error) {
	if !s.available() {
		return nil, ErrServiceUnavailable
	}
	settings, err := s.Config(ctx)
	if err != nil {
		return nil, err
	}
	if !settings.SMTPConfigured {
		return nil, ErrEmailNotConfigured
	}
	var b feishuKeyBatch
	err = s.transaction(ctx, func(ctx context.Context, c *dbent.Client) error {
		ok, err := feishuReadJSON(ctx, c, feishuKeyJobPrefix+id, &b)
		if err != nil {
			return err
		}
		if !ok || b.ActorID != actor {
			return ErrFeishuKeyStale
		}
		if b.Status != "preview" {
			return nil
		} // Retry after a lost HTTP response is the same job.
		var currentConfig FeishuKeyDeliveryConfig
		if _, err = feishuReadJSON(ctx, c, feishuKeyConfigKey, &currentConfig); err != nil {
			return err
		}
		if b.ExpiresAt < time.Now().Unix() || b.Config.Version != currentConfig.Version {
			return ErrFeishuKeyStale
		}
		cfg, err := s.directory.setup.settings.GetFeishuOAuthConfig(ctx)
		if err != nil || cfg.Fingerprint() != b.Fingerprint {
			return ErrFeishuKeyStale
		}
		var active string
		exists, err := feishuReadJSON(ctx, c, feishuKeyActiveJob, &active)
		if err != nil {
			return err
		}
		if exists {
			var old feishuKeyBatch
			_, err = feishuReadJSON(ctx, c, feishuKeyJobPrefix+active, &old)
			if err != nil {
				return err
			}
			if old.Status == "running" {
				return infraerrors.Conflict("FEISHU_KEY_BUSY", "A delivery batch is already running")
			}
		}
		if b.view().Ready == 0 {
			return ErrFeishuKeyInvalid
		}
		for i := range b.Rows {
			if b.Rows[i].Status == "ready" {
				b.Rows[i].Status = "pending"
			}
		}
		b.Status = "running"
		if err = feishuWriteJSON(ctx, c, feishuKeyJobPrefix+b.ID, &b); err != nil {
			return err
		}
		return feishuWriteJSON(ctx, c, feishuKeyActiveJob, b.ID)
	})
	if err != nil {
		return nil, err
	}
	return b.view(), nil
}

func (s *FeishuKeyDeliveryService) Retry(ctx context.Context, actor int64, id string, includeUncertain bool) (*FeishuKeyDeliveryJob, error) {
	if !s.available() {
		return nil, ErrServiceUnavailable
	}
	settings, err := s.Config(ctx)
	if err != nil {
		return nil, err
	}
	if !settings.SMTPConfigured {
		return nil, ErrEmailNotConfigured
	}
	var b feishuKeyBatch
	err = s.transaction(ctx, func(ctx context.Context, c *dbent.Client) error {
		var active string
		_, err := feishuReadJSON(ctx, c, feishuKeyActiveJob, &active)
		if err != nil {
			return err
		}
		if active != id {
			return ErrFeishuKeyStale
		}
		ok, err := feishuReadJSON(ctx, c, feishuKeyJobPrefix+id, &b)
		if err != nil {
			return err
		}
		if !ok {
			return ErrFeishuKeyStale
		}
		// Retrying after a lost response observes the batch already resumed by
		// the first request, without resetting any recipient a second time.
		if b.Status == "running" {
			return nil
		}
		cfg, err := s.directory.setup.settings.GetFeishuOAuthConfig(ctx)
		if err != nil || cfg.Fingerprint() != b.Fingerprint {
			return ErrFeishuKeyStale
		}
		retrying := 0
		for i := range b.Rows {
			r := &b.Rows[i]
			if r.Status == "failed" || includeUncertain && r.Status == "uncertain" {
				r.Status, r.Reason = "pending", ""
				retrying++
			}
		}
		if retrying == 0 {
			return ErrFeishuKeyInvalid
		}
		b.ActorID, b.Status, b.Worker, b.LeaseUntil = actor, "running", "", 0
		return feishuWriteJSON(ctx, c, feishuKeyJobPrefix+id, &b)
	})
	if err != nil {
		return nil, err
	}
	return b.view(), nil
}
