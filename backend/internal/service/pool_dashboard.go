package service

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
)

// PoolSourceGroups intersects owned keys with current binding permissions.
// Client-supplied group/user IDs are never used to determine visibility.
func (s *APIKeyService) PoolSourceGroups(ctx context.Context, userID int64) ([]Group, error) {
	available, err := s.GetAvailableGroups(ctx, userID)
	if err != nil {
		return nil, err
	}
	used := map[int64]bool{}
	var keys []APIKey
	if repo, ok := s.apiKeyRepo.(apiKeyAllByUserIDLister); ok {
		keys, err = repo.ListAllByUserID(ctx, userID, APIKeyListFilters{})
		if err != nil {
			return nil, err
		}
	} else {
		for page := 1; ; page++ {
			rows, total, listErr := s.apiKeyRepo.ListByUserID(ctx, userID, pagination.PaginationParams{Page: page, PageSize: 200}, APIKeyListFilters{})
			if listErr != nil {
				return nil, listErr
			}
			keys = append(keys, rows...)
			if len(rows) == 0 || total == nil || int64(len(keys)) >= total.Total {
				break
			}
		}
	}
	now := time.Now()
	for _, key := range keys {
		if key.UserID == userID && key.GroupID != nil && key.Status == StatusActive && (key.ExpiresAt == nil || key.ExpiresAt.After(now)) {
			used[*key.GroupID] = true
		}
	}
	out := make([]Group, 0)
	for _, group := range available {
		if group.ID > 0 && used[group.ID] {
			out = append(out, group)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

// These DTOs deliberately do not embed Account or any admin DTO. The account
// name, email, credentials, proxies, group bindings and raw errors stay private.
type PoolQuotaAccount struct {
	ID                int64                   `json:"id"`
	Label             string                  `json:"label"`
	Platform          string                  `json:"platform"`
	Available         bool                    `json:"available"`
	Windows           []AccountOpsUsageWindow `json:"windows"`
	Balance           *float64                `json:"balance"`
	BalanceUnit       string                  `json:"balance_unit"`
	BalanceFreshUntil *time.Time              `json:"balance_fresh_until"`
	UpdatedAt         *time.Time              `json:"updated_at"`
}
type PoolQuotaGroup struct {
	ID       int64              `json:"id"`
	Name     string             `json:"name"`
	Platform string             `json:"platform"`
	Accounts []PoolQuotaAccount `json:"accounts"`
}
type UserPoolDashboard struct {
	GeneratedAt time.Time        `json:"generated_at"`
	Groups      []PoolQuotaGroup `json:"groups"`
}

// BuildPoolQuotaAccounts reads saved observations only. Unknown/expired quota
// stays unknown; an elapsed reset is never interpreted as a fresh 100% grant.
func BuildPoolQuotaAccounts(accounts []Account, parents map[int64]*Account, now time.Time) []PoolQuotaAccount {
	out := make([]PoolQuotaAccount, 0, len(accounts))
	seen := map[int64]bool{}
	for i := range accounts {
		a := &accounts[i]
		if seen[a.ID] {
			continue
		}
		seen[a.ID] = true
		r := PoolQuotaAccount{ID: a.ID, Label: fmt.Sprintf("#%d", a.ID), Platform: a.Platform, Available: a.IsSchedulable(), Windows: []AccountOpsUsageWindow{}}
		source := a
		if a.IsShadow() {
			source = parents[*a.ParentAccountID]
			if source != nil && !source.IsSchedulable() {
				r.Available = false
			}
		}
		// Resolve shared quota server-side without exposing parent identities.
		if source != nil {
			for _, w := range opsExtraUsageWindows(source, now) {
				if w.Window == "any" {
					continue
				}
				if w.UsedPercent != nil && (*w.UsedPercent < 0 || *w.UsedPercent > 100) {
					w.UsedPercent = nil
					w.Status = "unknown"
				}
				if w.ObservedAt != nil && (r.UpdatedAt == nil || w.ObservedAt.After(*r.UpdatedAt)) {
					r.UpdatedAt = w.ObservedAt
				}
				if w.Status == "ok" && w.UsedPercent != nil && *w.UsedPercent >= 100 {
					r.Available = false
				}
				r.Windows = append(r.Windows, w)
			}
			balance := opsBalanceAccount(source, now)
			if balance.BalanceStatus == "ok" && balance.Balance != nil {
				switch balance.Unit {
				case "USD", "CNY", "EUR", "credits", "tokens":
					r.Balance, r.BalanceUnit = balance.Balance, balance.Unit
					r.UpdatedAt = balance.ReceivedAt
					if snap := decodeUpstreamBillingProbeSnapshot(source.Extra); snap != nil && snap.Balance != nil {
						r.BalanceFreshUntil = snap.Balance.FreshUntil
					}
					if *r.Balance <= 0 {
						r.Available = false
					}
				}
			}
		}
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}
