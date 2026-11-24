package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/authidentity"
	dbuser "github.com/Wei-Shaw/sub2api/ent/user"
	"github.com/Wei-Shaw/sub2api/internal/pkg/feishu"
)

// A one-time, app-scoped contact backfill for existing accounts. It never
// provisions employees, changes permissions, issues keys or sends messages.
// A failed/incomplete read leaves the marker absent so a later tick retries.
func (s *FeishuKeyDeliveryService) backfillEmails(ctx context.Context) {
	cfg, err := s.directory.setup.settings.GetFeishuOAuthConfig(ctx)
	if err != nil {
		return
	}
	digest := sha256.Sum256([]byte(cfg.IdentityKey()))
	marker := "feishu_email_backfill_v1:" + hex.EncodeToString(digest[:16])
	var result struct {
		Done                                 bool
		Filled, Missing, Conflict, Protected int
		FinishedAt                           int64
	}
	if ok, err := feishuReadJSON(ctx, s.client, marker, &result); err != nil || ok && result.Done {
		return
	}
	token, err := s.directory.setup.api.TenantToken(ctx, cfg.AppID, cfg.AppSecret)
	if err != nil {
		return
	}
	members, err := s.directory.setup.api.DirectoryMembers(ctx, token)
	if err != nil {
		return
	}
	byID := map[string]feishu.DirectoryMember{}
	for _, m := range members {
		byID[m.OpenID] = m
	}
	identities, err := s.client.AuthIdentity.Query().Where(authidentity.ProviderTypeEQ("feishu"), authidentity.ProviderKeyEQ(cfg.IdentityKey()), authidentity.HasUserWith(dbuser.DeletedAtIsNil())).All(ctx)
	if err != nil {
		return
	}
	for _, identity := range identities {
		m, ok := byID[identity.ProviderSubject]
		if !ok || !m.Active() {
			continue
		}
		current, err := s.directory.setup.settings.GetFeishuOAuthConfig(ctx)
		if err != nil || current.Fingerprint() != cfg.Fingerprint() {
			return
		}
		report := &FeishuEmailSyncResult{}
		err = s.transaction(ctx, func(ctx context.Context, c *dbent.Client) error {
			u, err := s.directory.auth.userRepo.GetByID(ctx, identity.UserID)
			if err != nil {
				return err
			}
			if !u.IsActive() || u.SignupSource != "feishu" || !strings.HasSuffix(strings.ToLower(u.Email), FeishuConnectSyntheticEmailDomain) {
				return nil
			}
			exists, err := c.AuthIdentity.Query().Where(authidentity.IDEQ(identity.ID), authidentity.UserIDEQ(u.ID), authidentity.ProviderKeyEQ(cfg.IdentityKey()), authidentity.ProviderSubjectEQ(m.OpenID)).Exist(ctx)
			if err != nil {
				return err
			}
			if !exists {
				return nil
			}
			return s.directory.auth.syncFeishuContactEmail(ctx, u, &feishu.UserInfo{EnterpriseEmail: m.EnterpriseEmail, Email: m.Email}, report)
		})
		if err != nil {
			return
		}
		switch report.Status {
		case "filled":
			result.Filled++
		case "missing":
			result.Missing++
		case "conflict":
			result.Conflict++
		case "protected":
			result.Protected++
		}
	}
	result.Done, result.FinishedAt = result.Missing == 0, time.Now().Unix()
	_ = feishuWriteJSON(ctx, s.client, marker, &result)
}
