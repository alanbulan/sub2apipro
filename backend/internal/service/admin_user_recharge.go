package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"math"
	"sort"
	"strings"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/redeemcode"
	dbuser "github.com/Wei-Shaw/sub2api/ent/user"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

type BatchUserRechargeResult struct {
	Affected      int     `json:"affected"`
	AmountPerUser float64 `json:"amount_per_user"`
	TotalAmount   float64 `json:"total_amount"`
}

// BatchRechargeUsers commits balance increments and their audit ledger together.
// Incrementing the column in SQL preserves concurrent billing deductions.
func (s *adminServiceImpl) BatchRechargeUsers(ctx context.Context, ids []int64, amount float64, notes string, operationKey string) (*BatchUserRechargeResult, error) {
	if operationKey == "" || len(ids) == 0 || len(ids) > 500 || math.IsNaN(amount) || math.IsInf(amount, 0) || amount < 0.00000001 || amount > 1e9 || len(notes) > 1000 {
		return nil, infraerrors.BadRequest("INVALID_BATCH_RECHARGE", "Select 1–500 users and a positive amount of at most 1,000,000,000")
	}
	amount = math.Round(amount*1e8) / 1e8
	unique := make(map[int64]bool, len(ids))
	cleaned := make([]int64, 0, len(ids))
	for _, id := range ids {
		if id <= 0 {
			return nil, infraerrors.BadRequest("INVALID_USER_ID", "Invalid user ID")
		}
		if !unique[id] {
			unique[id] = true
			cleaned = append(cleaned, id)
		}
	}
	sort.Slice(cleaned, func(i, j int) bool { return cleaned[i] < cleaned[j] })
	if s.entClient == nil {
		return nil, ErrServiceUnavailable
	}
	tx, err := s.entClient.Tx(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()
	txCtx := dbent.NewTxContext(ctx, tx)
	release, err := lockAuthPendingIdentityKeys(txCtx, tx.Client(), "batch-recharge:"+operationKey)
	if err != nil {
		return nil, err
	}
	defer release()
	digest := sha256.Sum256([]byte(operationKey))
	markerCode := "BATCH-" + hex.EncodeToString(digest[:])[:26]
	payload, _ := json.Marshal(struct {
		IDs    []int64
		Amount float64
		Notes  string
	}{cleaned, amount, strings.TrimSpace(notes)})
	fingerprint := sha256.Sum256(payload)
	fingerprintText := hex.EncodeToString(fingerprint[:])
	existing, err := tx.RedeemCode.Query().Where(redeemcode.CodeEQ(markerCode)).Only(ctx)
	if err != nil && !dbent.IsNotFound(err) {
		return nil, err
	}
	result := &BatchUserRechargeResult{Affected: len(cleaned), AmountPerUser: amount, TotalAmount: amount * float64(len(cleaned))}
	if existing != nil {
		if existing.Type != "admin_batch_marker" || (existing.Notes == nil || *existing.Notes != fingerprintText) {
			return nil, ErrIdempotencyKeyConflict
		}
		return result, nil
	}
	// This zero-value internal marker has no used_by, so it never appears in a
	// user's balance history. It makes retries safe even after coordinator TTL
	// expiry or a lost response after the money transaction has committed.
	if _, err := tx.RedeemCode.Create().SetCode(markerCode).SetType("admin_batch_marker").SetValue(0).SetStatus(StatusUsed).SetNotes(fingerprintText).Save(ctx); err != nil {
		return nil, err
	}
	now := time.Now()
	for _, id := range cleaned {
		count, err := tx.User.Update().Where(dbuser.IDEQ(id), dbuser.DeletedAtIsNil()).AddBalance(amount).Save(ctx)
		if err != nil {
			return nil, err
		}
		if count != 1 {
			return nil, ErrUserNotFound
		}
		code, err := GenerateRedeemCode()
		if err != nil {
			return nil, err
		}
		_, err = tx.RedeemCode.Create().SetCode(code).SetType(AdjustmentTypeAdminBalance).SetValue(amount).
			SetStatus(StatusUsed).SetUsedBy(id).SetUsedAt(now).SetNotes(strings.TrimSpace(notes)).Save(ctx)
		if err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	for _, id := range cleaned {
		if s.authCacheInvalidator != nil {
			s.authCacheInvalidator.InvalidateAuthCacheByUserID(ctx, id)
		}
		s.tryAccrueAffiliateRebateForAdminRecharge(ctx, id, "add", amount)
	}
	if s.billingCacheService != nil {
		go func() {
			cacheCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			for _, id := range cleaned {
				_ = s.billingCacheService.InvalidateUserBalance(cacheCtx, id)
			}
		}()
	}
	return result, nil
}
