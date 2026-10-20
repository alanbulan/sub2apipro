package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/mail"
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
	user, _, err := s.ensureFeishuEnterpriseUser(ctx, cfg, info, false)
	return user, err
}

func (s *AuthService) ensureFeishuEnterpriseUser(ctx context.Context, cfg *FeishuConnectSettings, info *feishu.UserInfo, synchronizeProfile bool) (*User, bool, error) {
	if s == nil || s.userRepo == nil || s.entClient == nil || s.settingService == nil {
		return nil, false, ErrServiceUnavailable
	}
	current, err := s.settingService.GetFeishuOAuthConfig(ctx)
	if err != nil {
		return nil, false, err
	}
	if cfg == nil || cfg.Fingerprint() != current.Fingerprint() {
		return nil, false, infraerrors.BadRequest("FEISHU_CONFIG_CHANGED", "Feishu configuration changed; start sign-in again")
	}
	if info == nil || info.TenantKey != current.TenantKey || strings.TrimSpace(info.OpenID) == "" {
		return nil, false, infraerrors.Forbidden("FEISHU_TENANT_MISMATCH", "Only verified members of the configured Feishu enterprise can sign in")
	}
	providerKey := current.IdentityKey()
	grantPlan := s.resolveSignupGrantPlan(ctx, "feishu")
	defaultRPM := s.settingService.GetDefaultUserRPMLimit(ctx)
	// Read the current settings, as the shared login handler does. Provisioning
	// must not rely on the process-wide backend-mode fallback cache.
	settings, err := s.settingService.GetPublicSettings(ctx)
	if err != nil || settings == nil {
		return nil, false, ErrServiceUnavailable
	}
	backendMode := settings.BackendModeEnabled

	tx, err := s.entClient.Tx(ctx)
	if err != nil {
		return nil, false, err
	}
	defer func() { _ = tx.Rollback() }()
	// Keep the request's abuse/rate checks, but do not count verified employees
	// sharing an office IP against the public signup lifetime account limit.
	txCtx := dbent.NewTxContext(authguard.WithoutRegistrationIP(ctx), tx)
	release, err := lockAuthPendingIdentityKeys(txCtx, tx.Client(), "feishu-provision:"+providerKey+":"+info.OpenID)
	if err != nil {
		return nil, false, err
	}
	defer release()
	identity, err := tx.Client().AuthIdentity.Query().Where(
		authidentity.ProviderTypeEQ("feishu"),
		authidentity.ProviderKeyEQ(providerKey),
		authidentity.ProviderSubjectEQ(info.OpenID),
	).Only(txCtx)
	if err != nil && !dbent.IsNotFound(err) {
		return nil, false, err
	}
	if identity != nil {
		user, err := s.userRepo.GetByID(txCtx, identity.UserID)
		if err != nil {
			return nil, false, err
		}
		if !user.IsActive() {
			return nil, false, ErrUserNotActive
		}
		if err := s.syncFeishuContactEmail(txCtx, user, info); err != nil {
			return nil, false, err
		}
		if synchronizeProfile {
			if user.SignupSource == "feishu" {
				name := feishuInitialUsername(info.Name, user.Email[:min(12, len(user.Email))])
				if name != user.Username {
					user.Username = name
					if err := s.userRepo.Update(txCtx, user, UserUpdateFields{Username: true}); err != nil {
						return nil, false, err
					}
				}
				if avatarInput, err := normalizeUserAvatarInput(info.AvatarURL); err == nil && user.AvatarURL != info.AvatarURL {
					avatar, err := s.userRepo.UpsertUserAvatar(txCtx, user.ID, avatarInput)
					if err != nil {
						return nil, false, err
					}
					applyUserAvatar(user, avatar)
				}
			}
			metadata := identity.Metadata
			if metadata == nil {
				metadata = map[string]any{}
			}
			metadata["display_name"] = info.Name
			metadata["avatar_url"] = info.AvatarURL
			if _, err := identity.Update().SetMetadata(metadata).Save(txCtx); err != nil {
				return nil, false, err
			}
		}
		if err := tx.Commit(); err != nil {
			return nil, false, err
		}
		return user, false, nil
	}
	if backendMode {
		return nil, false, infraerrors.Forbidden("BACKEND_MODE_ADMIN_ONLY", "Backend mode is active. Only admin or observer login is allowed.")
	}
	password, err := randomHexString(32)
	if err != nil {
		return nil, false, ErrServiceUnavailable
	}
	hash, err := s.HashPassword(password)
	if err != nil {
		return nil, false, fmt.Errorf("hash Feishu account password: %w", err)
	}
	email := feishuSyntheticEmail(providerKey, info.OpenID)
	name := feishuInitialUsername(info.Name, email[:12])
	user := &User{
		Email: email, Username: name, PasswordHash: hash,
		Role: RoleUser, Status: StatusActive, SignupSource: "feishu",
		Balance: grantPlan.Balance, Concurrency: grantPlan.Concurrency, RPMLimit: defaultRPM,
	}
	if err := s.userRepo.Create(txCtx, user); err != nil {
		return nil, false, err
	}
	if err := s.syncFeishuContactEmail(txCtx, user, info); err != nil {
		return nil, false, err
	}
	if _, err := tx.Client().AuthIdentity.Create().
		SetUserID(user.ID).SetProviderType("feishu").SetProviderKey(providerKey).
		SetProviderSubject(info.OpenID).SetVerifiedAt(time.Now().UTC()).
		SetMetadata(map[string]any{"app_id": current.AppID, "tenant_key": current.TenantKey, "display_name": name, "avatar_url": info.AvatarURL}).
		Save(txCtx); err != nil {
		return nil, false, err
	}
	if avatarInput, err := normalizeUserAvatarInput(info.AvatarURL); err == nil {
		avatar, err := s.userRepo.UpsertUserAvatar(txCtx, user.ID, avatarInput)
		if err != nil {
			return nil, false, err
		}
		applyUserAvatar(user, avatar)
	}
	if err := tx.Commit(); err != nil {
		return nil, false, err
	}
	// Defaults run once, only for the transaction that created this identity.
	s.postAuthUserBootstrap(ctx, user, "feishu", false)
	s.assignSubscriptions(ctx, user.ID, grantPlan.Subscriptions, "auto assigned by signup defaults")
	_ = s.snapshotPlatformQuotaDefaults(ctx, user.ID, &grantPlan)
	s.bindOAuthAffiliate(ctx, user.ID, "")
	return user, true, nil
}

// Feishu's email field is an administrator-supplied contact, not proof of
// mailbox ownership. Store it as profile data without creating an email login.
func (s *AuthService) syncFeishuContactEmail(ctx context.Context, user *User, info *feishu.UserInfo) error {
	if user.SignupSource != "feishu" || !strings.HasSuffix(strings.ToLower(user.Email), FeishuConnectSyntheticEmailDomain) {
		return nil
	}
	email := feishuContactEmail(info)
	if email == "" {
		return nil
	}
	guard, ok := s.userRepo.(interface {
		FillFeishuContactEmail(context.Context, int64, string) (bool, error)
	})
	if !ok {
		return ErrServiceUnavailable
	}
	updated, err := guard.FillFeishuContactEmail(ctx, user.ID, email)
	if err != nil {
		if errors.Is(err, ErrEmailExists) {
			return nil
		}
		return err
	}
	if updated {
		user.Email = email
	}
	return nil
}

func feishuContactEmail(info *feishu.UserInfo) string {
	if info == nil {
		return ""
	}
	for _, raw := range []string{info.EnterpriseEmail, info.Email} {
		email := strings.ToLower(strings.TrimSpace(raw))
		address, err := mail.ParseAddress(email)
		if err == nil && address.Address == email && len(email) <= 255 && !strings.HasSuffix(email, ".invalid") {
			return email
		}
	}
	return ""
}

// Imported contact details must not become login, recovery or automatic account
// adoption credentials. An authenticated member can verify/bind email normally.
func (s *AuthService) emailAuthenticationAllowed(ctx context.Context, user *User) (bool, error) {
	if user == nil {
		return false, ErrUserNotFound
	}
	if user.SignupSource != "feishu" {
		return true, nil
	}
	if isReservedEmail(user.Email) {
		return false, nil
	}
	if s.entClient == nil {
		return false, ErrServiceUnavailable
	}
	client := s.entClient
	if tx := dbent.TxFromContext(ctx); tx != nil {
		client = tx.Client()
	}
	return client.AuthIdentity.Query().Where(
		authidentity.UserIDEQ(user.ID), authidentity.ProviderTypeEQ("email"),
		authidentity.ProviderKeyEQ("email"), authidentity.ProviderSubjectEQ(strings.ToLower(strings.TrimSpace(user.Email))),
		authidentity.VerifiedAtNotNil(),
	).Exist(ctx)
}

func (s *AuthService) requireEmailAuthentication(ctx context.Context, user *User) error {
	allowed, err := s.emailAuthenticationAllowed(ctx, user)
	if err != nil {
		return ErrServiceUnavailable
	}
	if !allowed {
		return ErrInvalidCredentials
	}
	return nil
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
