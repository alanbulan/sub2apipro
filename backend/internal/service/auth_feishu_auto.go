package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/authidentity"
	"github.com/Wei-Shaw/sub2api/internal/pkg/authguard"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/feishu"
)

// EnsureFeishuEnterpriseUser accepts only an identity just verified by Feishu.
// Enterprise membership authorizes provisioning independently of public signup.
// Never match accounts by provider email or display name: neither proves ownership.
func (s *AuthService) EnsureFeishuEnterpriseUser(ctx context.Context, cfg *FeishuConnectSettings, info *feishu.UserInfo) (*User, error) {
	if s == nil || s.userRepo == nil || s.entClient == nil || s.settingService == nil {
		return nil, ErrServiceUnavailable
	}
	current, err := s.settingService.GetFeishuOAuthConfig(ctx)
	if err != nil {
		return nil, err
	}
	if cfg == nil || cfg.Fingerprint() != current.Fingerprint() {
		return nil, infraerrors.BadRequest("FEISHU_CONFIG_CHANGED", "Feishu configuration changed; start sign-in again")
	}
	if info == nil || info.TenantKey != current.TenantKey || strings.TrimSpace(info.OpenID) == "" {
		return nil, infraerrors.Forbidden("FEISHU_TENANT_MISMATCH", "Only verified members of the configured Feishu enterprise can sign in")
	}
	providerKey := current.IdentityKey()
	grantPlan := s.resolveSignupGrantPlan(ctx, "feishu")
	defaultRPM := s.settingService.GetDefaultUserRPMLimit(ctx)
	backendMode := s.settingService.IsBackendModeEnabled(ctx)

	tx, err := s.entClient.Tx(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()
	// Keep the request's abuse/rate checks, but do not count verified employees
	// sharing an office IP against the public signup lifetime account limit.
	txCtx := dbent.NewTxContext(authguard.WithoutRegistrationIP(ctx), tx)
	release, err := lockAuthPendingIdentityKeys(txCtx, tx.Client(), "feishu-provision:"+providerKey+":"+info.OpenID)
	if err != nil {
		return nil, err
	}
	defer release()
	identity, err := tx.Client().AuthIdentity.Query().Where(
		authidentity.ProviderTypeEQ("feishu"),
		authidentity.ProviderKeyEQ(providerKey),
		authidentity.ProviderSubjectEQ(info.OpenID),
	).Only(txCtx)
	if err != nil && !dbent.IsNotFound(err) {
		return nil, err
	}
	if identity != nil {
		if err := tx.Commit(); err != nil {
			return nil, err
		}
		user, err := s.userRepo.GetByID(ctx, identity.UserID)
		if err != nil {
			return nil, err
		}
		if !user.IsActive() {
			return nil, ErrUserNotActive
		}
		return user, nil
	}
	if backendMode {
		return nil, infraerrors.Forbidden("BACKEND_MODE_ADMIN_ONLY", "Backend mode is active. Only admin or observer login is allowed.")
	}
	password, err := randomHexString(32)
	if err != nil {
		return nil, ErrServiceUnavailable
	}
	hash, err := s.HashPassword(password)
	if err != nil {
		return nil, fmt.Errorf("hash Feishu account password: %w", err)
	}
	email := feishuSyntheticEmail(providerKey, info.OpenID)
	name := feishuInitialUsername(info.Name, email[:12])
	user := &User{
		Email: email, Username: name, PasswordHash: hash,
		Role: RoleUser, Status: StatusActive, SignupSource: "feishu",
		Balance: grantPlan.Balance, Concurrency: grantPlan.Concurrency, RPMLimit: defaultRPM,
	}
	if err := s.userRepo.Create(txCtx, user); err != nil {
		return nil, err
	}
	if _, err := tx.Client().AuthIdentity.Create().
		SetUserID(user.ID).SetProviderType("feishu").SetProviderKey(providerKey).
		SetProviderSubject(info.OpenID).SetVerifiedAt(time.Now().UTC()).
		SetMetadata(map[string]any{"app_id": current.AppID, "tenant_key": current.TenantKey, "display_name": name, "avatar_url": info.AvatarURL}).
		Save(txCtx); err != nil {
		return nil, err
	}
	if avatarInput, err := normalizeUserAvatarInput(info.AvatarURL); err == nil {
		avatar, err := s.userRepo.UpsertUserAvatar(txCtx, user.ID, avatarInput)
		if err != nil {
			return nil, err
		}
		applyUserAvatar(user, avatar)
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	// Defaults run once, only for the transaction that created this identity.
	s.postAuthUserBootstrap(ctx, user, "feishu", false)
	s.assignSubscriptions(ctx, user.ID, grantPlan.Subscriptions, "auto assigned by signup defaults")
	_ = s.snapshotPlatformQuotaDefaults(ctx, user.ID, &grantPlan)
	s.bindOAuthAffiliate(ctx, user.ID, "")
	return user, nil
}

func feishuSyntheticEmail(providerKey, openID string) string {
	sum := sha256.Sum256([]byte(providerKey + "\x00" + openID))
	return hex.EncodeToString(sum[:]) + FeishuConnectSyntheticEmailDomain
}

func feishuInitialUsername(name, fallback string) string {
	name = strings.TrimSpace(name)
	// Match the shared username validator's 100-byte limit without splitting UTF-8.
	for len(name) > 100 {
		_, size := utf8.DecodeLastRuneInString(name)
		name = name[:len(name)-size]
	}
	if name == "" {
		return "飞书用户-" + fallback
	}
	return name
}
