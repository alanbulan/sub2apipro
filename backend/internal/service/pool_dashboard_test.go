//go:build unit

package service

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type poolKeysRepo struct {
	APIKeyRepository
	keys []APIKey
}

func (r *poolKeysRepo) ListAllByUserID(context.Context, int64, APIKeyListFilters) ([]APIKey, error) {
	return r.keys, nil
}

func TestPoolSourceGroupsIntersectOwnerKeysWithCurrentPermissions(t *testing.T) {
	group1, group2, group3 := int64(1), int64(2), int64(3)
	expired := time.Now().Add(-time.Hour)
	svc := &APIKeyService{
		userRepo:    &visibilityUserRepo{user: &User{ID: 1, AllowedGroups: []int64{1}, RestrictPublicGroups: true}},
		userSubRepo: &visibilitySubRepo{},
		groupRepo:   &visibilityGroupRepo{groups: []Group{{ID: 1}, {ID: 2}, {ID: 3}}},
		apiKeyRepo: &poolKeysRepo{keys: []APIKey{
			{UserID: 1, GroupID: &group1, Status: StatusActive}, {UserID: 1, GroupID: &group1, Status: StatusActive},
			{UserID: 1, GroupID: &group2, Status: StatusActive}, {UserID: 2, GroupID: &group3, Status: StatusActive},
			{UserID: 1, GroupID: &group3, Status: StatusActive, ExpiresAt: &expired},
		}},
	}
	groups, err := svc.PoolSourceGroups(context.Background(), 1)
	require.NoError(t, err)
	require.Len(t, groups, 1)
	require.Equal(t, int64(1), groups[0].ID)
}

func TestPoolQuotaAccountsRedactSecretsAndRejectExpiredObservations(t *testing.T) {
	now := time.Now().UTC()
	accounts := []Account{{ID: 1, Name: "secret@example.com", Platform: PlatformOpenAI, Type: AccountTypeOAuth, Status: StatusActive, Schedulable: true,
		Credentials: map[string]any{"access_token": "private-token", "email": "secret@example.com"},
		Extra:       map[string]any{"codex_usage_updated_at": now.Add(-time.Minute).Format(time.RFC3339), "codex_5h_used_percent": 20.0, "codex_5h_reset_at": now.Add(time.Hour).Format(time.RFC3339)},
	}}
	rows := BuildPoolQuotaAccounts(accounts, nil, now)
	require.Len(t, rows, 1)
	require.Equal(t, "#1", rows[0].Label)
	require.NotNil(t, rows[0].Windows[0].UsedPercent)
	raw, err := json.Marshal(rows)
	require.NoError(t, err)
	require.NotContains(t, string(raw), "secret@example.com")
	require.NotContains(t, string(raw), "private-token")
	require.NotContains(t, string(raw), "credentials")
	var response []map[string]any
	require.NoError(t, json.Unmarshal(raw, &response))
	require.Len(t, response[0], 9)
	window := response[0]["windows"].([]any)[0].(map[string]any)
	require.Len(t, window, 6)
	for _, key := range []string{"window", "label", "used_percent", "status", "observed_at", "resets_at"} {
		require.Contains(t, window, key)
	}
	accounts[0].Extra["codex_5h_reset_at"] = now.Add(-time.Minute).Format(time.RFC3339)
	rows = BuildPoolQuotaAccounts(accounts, nil, now)
	require.Nil(t, rows[0].Windows[0].UsedPercent)
	require.Equal(t, "stale", rows[0].Windows[0].Status)
	// Shared credentials retain separate global/Spark quotas. A globally
	// exhausted parent must not override a usable Spark quota or its status.
	accounts[0].Extra["codex_5h_reset_at"] = now.Add(time.Hour).Format(time.RFC3339)
	parent := accounts[0]
	parent.Extra = map[string]any{"codex_usage_updated_at": now.Format(time.RFC3339), "codex_5h_used_percent": 100.0, "codex_5h_reset_at": now.Add(time.Hour).Format(time.RFC3339)}
	parent.Schedulable = false
	parentID := parent.ID
	shadow := Account{ID: 2, ParentAccountID: &parentID, Platform: PlatformOpenAI, Status: StatusActive, Schedulable: true, Extra: accounts[0].Extra}
	rows = BuildPoolQuotaAccounts([]Account{shadow}, map[int64]*Account{parentID: &parent}, now)
	require.Len(t, rows, 1)
	require.Equal(t, int64(2), rows[0].ID)
	require.NotNil(t, rows[0].Windows[0].UsedPercent)
	require.Equal(t, 20.0, *rows[0].Windows[0].UsedPercent)
	require.True(t, rows[0].Available)
	rows = BuildPoolQuotaAccounts([]Account{shadow}, nil, now)
	require.False(t, rows[0].Available)
	shadow.Extra = nil
	rows = BuildPoolQuotaAccounts([]Account{shadow}, map[int64]*Account{parentID: &parent}, now)
	require.Nil(t, rows[0].Windows[0].UsedPercent)
}
