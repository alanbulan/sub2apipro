//go:build unit

package service

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestPoolAntigravityQuotaUsesOnlyAllowedCachedFields(t *testing.T) {
	now := time.Now().UTC()
	cache := NewUsageCache()
	cache.antigravityCache.Store(int64(1), &antigravityUsageCache{timestamp: now, usageInfo: &UsageInfo{
		ForbiddenReason: "private diagnostic", ValidationURL: "https://private.example.com/verify",
		AntigravityQuota: map[string]*AntigravityModelQuota{"claude-sonnet": {Utilization: 25, ResetTime: now.Add(time.Hour).Format(time.RFC3339)}},
	}})
	svc := &AccountUsageService{cache: cache}
	accounts := []Account{{ID: 1, Platform: PlatformAntigravity, Status: StatusActive, Schedulable: true}}
	rows := svc.PoolQuotaAccounts(accounts, nil, now)
	require.Len(t, rows, 1)
	require.Len(t, rows[0].Windows, 1)
	require.Equal(t, 25.0, *rows[0].Windows[0].UsedPercent)
	raw, err := json.Marshal(rows)
	require.NoError(t, err)
	require.NotContains(t, string(raw), "private diagnostic")
	require.NotContains(t, string(raw), "private.example.com")
	rows = svc.PoolQuotaAccounts(accounts, nil, now.Add(16*time.Minute))
	require.Nil(t, rows[0].Windows[0].UsedPercent)
}
