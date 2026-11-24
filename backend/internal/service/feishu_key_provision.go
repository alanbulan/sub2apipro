package service

import (
	"context"
	"fmt"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	dbuser "github.com/Wei-Shaw/sub2api/ent/user"
)

// Called only inside transaction after identity, group and balance checks.
// The durable receipt makes API retries and subsequent email delivery reuse
// one key and one initial gift, including requests on different replicas.
func (s *FeishuKeyDeliveryService) provisionKey(ctx context.Context, c *dbent.Client, b *feishuKeyBatch, r *feishuKeyRecipient) error {
	if !r.Reuse {
		// Only append the groups the administrator explicitly selected. Existing
		// permissions, user-owned keys and subscription entitlements survive.
		if _, err := c.User.UpdateOneID(r.UserID).AddAllowedGroupIDs(b.Config.GroupIDs...).Save(ctx); err != nil {
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
		if err := feishuWriteJSON(ctx, c, feishuKeyReceiptKey(r.UserID), &receipt); err != nil {
			return err
		}
	}
	return nil
}
