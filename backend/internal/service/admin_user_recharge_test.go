//go:build unit

package service

import (
	"context"
	"math"
	"testing"

	"github.com/Wei-Shaw/sub2api/ent/redeemcode"
	"github.com/stretchr/testify/require"
)

func TestBatchRechargeAtomicAndDurableRetry(t *testing.T) {
	client := newAdminServiceAuthIdentityBindingTestClient(t)
	ctx := context.Background()
	a, err := client.User.Create().SetEmail("recharge-a@example.com").SetPasswordHash("hash").SetBalance(5).Save(ctx)
	require.NoError(t, err)
	b, err := client.User.Create().SetEmail("recharge-b@example.com").SetPasswordHash("hash").SetBalance(2).Save(ctx)
	require.NoError(t, err)
	svc := &adminServiceImpl{entClient: client}
	// A late missing user rolls back earlier increments AND ledger entries.
	_, err = svc.BatchRechargeUsers(ctx, []int64{a.ID, b.ID, 999999}, 3, "grant", "failed-operation")
	require.Error(t, err)
	current, err := client.User.Get(ctx, a.ID)
	require.NoError(t, err)
	require.Equal(t, 5.0, current.Balance)
	count, err := client.RedeemCode.Query().Count(ctx)
	require.NoError(t, err)
	require.Zero(t, count)
	result, err := svc.BatchRechargeUsers(ctx, []int64{b.ID, a.ID, a.ID}, 3, "grant", "operation")
	require.NoError(t, err)
	require.Equal(t, 2, result.Affected)
	// Replay stays safe independently of the HTTP coordinator's lifetime.
	_, err = svc.BatchRechargeUsers(ctx, []int64{a.ID, b.ID}, 3, "grant", "operation")
	require.NoError(t, err)
	current, err = client.User.Get(ctx, a.ID)
	require.NoError(t, err)
	require.Equal(t, 8.0, current.Balance)
	count, err = client.RedeemCode.Query().Where(redeemcode.TypeEQ(AdjustmentTypeAdminBalance)).Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 2, count)
	_, err = svc.BatchRechargeUsers(ctx, []int64{a.ID, b.ID}, 4, "grant", "operation")
	require.ErrorIs(t, err, ErrIdempotencyKeyConflict)
	_, err = svc.BatchRechargeUsers(ctx, []int64{a.ID}, math.NaN(), "", "invalid")
	require.Error(t, err)
}
