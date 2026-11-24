package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/authidentity"
	dbuser "github.com/Wei-Shaw/sub2api/ent/user"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

func (s *FeishuKeyDeliveryService) StartWorker() {
	if !s.available() || s.cancel != nil {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		nextBackfill := time.Time{}
		for {
			if ctx.Err() != nil {
				return
			}
			if time.Now().After(nextBackfill) {
				backfillCtx, stop := context.WithTimeout(ctx, 2*time.Minute)
				s.backfillEmails(backfillCtx)
				stop()
				nextBackfill = time.Now().Add(5 * time.Minute)
			}
			s.workOne(ctx)
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()
}

func (s *FeishuKeyDeliveryService) Stop() {
	if s == nil {
		return
	}
	s.stopOnce.Do(func() {
		if s.cancel != nil {
			s.cancel()
		}
		s.wg.Wait()
	})
}

func (s *FeishuKeyDeliveryService) workOne(parent context.Context) {
	ctx, cancel := context.WithTimeout(parent, 2*time.Minute)
	defer cancel()
	var batch *feishuKeyBatch
	err := s.transaction(ctx, func(ctx context.Context, c *dbent.Client) error {
		var id string
		ok, err := feishuReadJSON(ctx, c, feishuKeyActiveJob, &id)
		if err != nil || !ok {
			return err
		}
		var b feishuKeyBatch
		ok, err = feishuReadJSON(ctx, c, feishuKeyJobPrefix+id, &b)
		if err != nil || !ok || b.Status != "running" {
			return err
		}
		if b.Worker != s.workerID && b.LeaseUntil > time.Now().Unix() {
			return nil
		}
		if b.Worker != s.workerID || b.LeaseUntil <= time.Now().Unix() {
			for i := range b.Rows {
				if b.Rows[i].Status == "sending" {
					b.Rows[i].Status, b.Rows[i].Reason = "uncertain", "smtp_uncertain"
				}
			}
		}
		b.Worker, b.LeaseUntil = s.workerID, time.Now().Add(5*time.Minute).Unix()
		batch = &b
		return feishuWriteJSON(ctx, c, feishuKeyJobPrefix+id, &b)
	})
	if err != nil || batch == nil {
		return
	}
	for i := range batch.Rows {
		if batch.Rows[i].Status == "pending" || batch.Rows[i].Status == "prepared" {
			s.processRecipient(ctx, batch, i)
			return
		}
	}
	_ = s.transaction(ctx, func(ctx context.Context, c *dbent.Client) error {
		var b feishuKeyBatch
		_, err := feishuReadJSON(ctx, c, feishuKeyJobPrefix+batch.ID, &b)
		if err != nil {
			return err
		}
		if b.Worker != s.workerID || b.Status != "running" {
			return nil
		}
		v := b.view()
		if v.Ready > 0 {
			return nil
		}
		b.Status = "completed"
		if v.Failed+v.Uncertain > 0 {
			b.Status = "partial"
		}
		b.Worker, b.LeaseUntil = "", 0
		return feishuWriteJSON(ctx, c, feishuKeyJobPrefix+b.ID, &b)
	})
}

func (s *FeishuKeyDeliveryService) changeRow(ctx context.Context, id string, index int, fn func(context.Context, *dbent.Client, *feishuKeyBatch, *feishuKeyRecipient) error) error {
	return s.transaction(ctx, func(ctx context.Context, c *dbent.Client) error {
		var b feishuKeyBatch
		ok, err := feishuReadJSON(ctx, c, feishuKeyJobPrefix+id, &b)
		if err != nil {
			return err
		}
		if !ok || b.Status != "running" || b.Worker != s.workerID || b.LeaseUntil <= time.Now().Unix() || index < 0 || index >= len(b.Rows) {
			return ErrFeishuKeyStale
		}
		if err = fn(ctx, c, &b, &b.Rows[index]); err != nil {
			return err
		}
		b.LeaseUntil = time.Now().Add(5 * time.Minute).Unix()
		return feishuWriteJSON(ctx, c, feishuKeyJobPrefix+id, &b)
	})
}

func (s *FeishuKeyDeliveryService) markFailed(ctx context.Context, id string, index int, reason string) {
	_ = s.changeRow(ctx, id, index, func(_ context.Context, _ *dbent.Client, _ *feishuKeyBatch, r *feishuKeyRecipient) error {
		if r.Status == "pending" || r.Status == "prepared" {
			r.Status, r.Reason = "failed", reason
		}
		return nil
	})
}

func (s *FeishuKeyDeliveryService) processRecipient(ctx context.Context, b *feishuKeyBatch, index int) {
	r := b.Rows[index]
	cfg, err := s.directory.setup.settings.GetFeishuOAuthConfig(ctx)
	if err != nil || cfg.Fingerprint() != b.Fingerprint {
		s.markFailed(ctx, b.ID, index, "config_changed")
		return
	}
	token, err := s.directory.setup.api.TenantToken(ctx, cfg.AppID, cfg.AppSecret)
	if err != nil {
		s.markFailed(ctx, b.ID, index, "directory_unavailable")
		return
	}
	member, err := s.directory.setup.api.DirectoryUser(ctx, token, r.OpenID)
	if err != nil {
		s.markFailed(ctx, b.ID, index, "directory_unavailable")
		return
	}
	if !member.Active() {
		s.markFailed(ctx, b.ID, index, "inactive_employee")
		return
	}
	if email := feishuRecipientEmail(*member, b.Config.AllowContactEmail); email == "" || email != r.Email {
		s.markFailed(ctx, b.ID, index, "recipient_changed")
		return
	}
	var keyID int64
	err = s.changeRow(ctx, b.ID, index, func(ctx context.Context, c *dbent.Client, b *feishuKeyBatch, r *feishuKeyRecipient) error {
		if r.Status != "pending" && r.Status != "prepared" {
			return ErrFeishuKeyStale
		}
		verified, err := c.AuthIdentity.Query().Where(authidentity.UserIDEQ(r.UserID), authidentity.ProviderTypeEQ("feishu"), authidentity.ProviderKeyEQ(b.ProviderKey), authidentity.ProviderSubjectEQ(r.OpenID)).Exist(ctx)
		if err != nil {
			return err
		}
		if !verified {
			r.Status, r.Reason = "failed", "identity_mismatch"
			return nil
		}
		if reason := s.evaluateRecipient(ctx, c, b, r, *member, true); reason != "" {
			r.Status, r.Reason = "failed", reason
			return nil
		}
		if !r.Reuse {
			// Only append the groups the administrator explicitly selected. Existing
			// permissions, user-owned keys and subscription entitlements survive.
			if _, err = c.User.UpdateOneID(r.UserID).AddAllowedGroupIDs(b.Config.GroupIDs...).Save(ctx); err != nil {
				return err
			}
			req := CreateAPIKeyRequest{Name: feishuInitialUsername(r.Name+" · "+b.Config.NameSuffix, "employee"), GroupIDs: b.Config.GroupIDs, Quota: b.Config.Quota, ConcurrencyLimit: b.Config.Concurrency, RateLimit5h: b.Config.RateLimit5h, RateLimit1d: b.Config.RateLimit1d, RateLimit7d: b.Config.RateLimit7d}
			if b.Config.ExpiresInDays > 0 {
				req.ExpiresInDays = &b.Config.ExpiresInDays
			}
			key, err := s.keys.Create(ctx, r.UserID, req)
			if err != nil {
				return err
			}
			r.KeyID, r.ExpiresAt = key.ID, key.ExpiresAt
			if r.Grant > 0 {
				count, err := c.User.Update().Where(dbuser.IDEQ(r.UserID), dbuser.DeletedAtIsNil(), dbuser.StatusEQ(StatusActive), dbuser.RoleEQ(RoleUser)).AddBalance(r.Grant).Save(ctx)
				if err != nil {
					return err
				}
				if count != 1 {
					return ErrUserNotActive
				}
				code, err := GenerateRedeemCode()
				if err != nil {
					return err
				}
				// This is an audited administrator gift, not a paid recharge. It
				// deliberately does not call the affiliate-rebate recharge path.
				_, err = c.RedeemCode.Create().SetCode(code).SetType(AdjustmentTypeAdminBalance).SetValue(r.Grant).SetStatus(StatusUsed).SetUsedBy(r.UserID).SetUsedAt(time.Now()).SetNotes(fmt.Sprintf("Feishu initial key grant; admin=%d batch=%s", b.ActorID, b.ID)).Save(ctx)
				if err != nil {
					return err
				}
				r.Balance += r.Grant
			}
			receipt := feishuKeyReceipt{KeyID: key.ID, UserID: r.UserID, Grant: r.Grant, IssuedAt: time.Now().Unix(), ActorID: b.ActorID}
			if err = feishuWriteJSON(ctx, c, feishuKeyReceiptKey(r.UserID), &receipt); err != nil {
				return err
			}
		}
		keyID = r.KeyID
		r.Status, r.Reason = "prepared", ""
		return nil
	})
	if err != nil {
		s.markFailed(ctx, b.ID, index, "service_error")
		return
	}
	if keyID == 0 {
		return
	}
	s.keys.InvalidateAuthCacheByUserID(ctx, r.UserID)
	if s.keys.autoCreditBilling != nil {
		if err = s.keys.autoCreditBilling.InvalidateUserBalance(ctx, r.UserID); err != nil {
			s.markFailed(ctx, b.ID, index, "service_error")
			return
		}
	}
	// Re-read committed data. The full key is kept in memory only for SMTP.
	key, err := s.keys.apiKeyRepo.GetByID(ctx, keyID)
	if err != nil || key.UserID != r.UserID {
		s.markFailed(ctx, b.ID, index, "key_unavailable")
		return
	}
	member, err = s.directory.setup.api.DirectoryUser(ctx, token, r.OpenID)
	if err != nil {
		s.markFailed(ctx, b.ID, index, "directory_unavailable")
		return
	}
	if feishuRecipientEmail(*member, b.Config.AllowContactEmail) != r.Email {
		s.markFailed(ctx, b.ID, index, "recipient_changed")
		return
	}
	cfg, err = s.directory.setup.settings.GetFeishuOAuthConfig(ctx)
	if err != nil || cfg.Fingerprint() != b.Fingerprint {
		s.markFailed(ctx, b.ID, index, "config_changed")
		return
	}
	var mailRow FeishuKeyDeliveryRow
	err = s.changeRow(ctx, b.ID, index, func(ctx context.Context, c *dbent.Client, b *feishuKeyBatch, r *feishuKeyRecipient) error {
		if r.Status != "prepared" {
			return ErrFeishuKeyStale
		}
		probe := *r
		if reason := s.evaluateRecipient(ctx, c, b, &probe, *member, false); reason != "" {
			r.Status, r.Reason = "failed", reason
			return nil
		}
		verified, err := c.AuthIdentity.Query().Where(authidentity.UserIDEQ(r.UserID), authidentity.ProviderTypeEQ("feishu"), authidentity.ProviderKeyEQ(b.ProviderKey), authidentity.ProviderSubjectEQ(r.OpenID)).Exist(ctx)
		if err != nil {
			return err
		}
		if !verified {
			r.Status, r.Reason = "failed", "identity_mismatch"
			return nil
		}
		r.Balance, r.GroupNames, r.Quota, r.ExpiresAt = probe.Balance, probe.GroupNames, probe.Quota, probe.ExpiresAt
		r.Status = "sending"
		mailRow = r.FeishuKeyDeliveryRow
		return nil
	})
	if err != nil || mailRow.Status != "sending" {
		return
	}
	body, err := renderFeishuKeyEmail(b.Config, mailRow, key.Key)
	if err == nil {
		err = s.send(ctx, r.Email, b.Config.Subject, body)
	}
	state, reason := "sent", ""
	if err != nil {
		state, reason = "failed", "smtp_failed"
		if ctx.Err() != nil || strings.HasPrefix(err.Error(), "write msg:") || strings.HasPrefix(err.Error(), "close writer:") {
			state, reason = "uncertain", "smtp_uncertain"
		}
	}
	// Preserve the SMTP result even if shutdown cancelled the worker context.
	finishCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = s.changeRow(finishCtx, b.ID, index, func(_ context.Context, _ *dbent.Client, _ *feishuKeyBatch, r *feishuKeyRecipient) error {
		if r.Status != "sending" {
			return infraerrors.Conflict("FEISHU_KEY_CHANGED", "Delivery state changed")
		}
		r.Status, r.Reason = state, reason
		return nil
	})
}
