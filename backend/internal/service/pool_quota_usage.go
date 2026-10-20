package service

import (
	"context"
	"sort"
	"strings"
	"time"
)

var poolQuotaRefreshSlots = make(chan struct{}, 4)

// PoolQuotaAccounts reuses the account page's quota cache. Cold Antigravity
// caches refresh in the background, so a slow provider cannot stall the page.
// The refresh reads quota only; it never sends a model inference request.
func (s *AccountUsageService) PoolQuotaAccounts(accounts []Account, parents map[int64]*Account, now time.Time) []PoolQuotaAccount {
	rows := BuildPoolQuotaAccounts(accounts, parents, now)
	if s == nil || s.cache == nil {
		return rows
	}
	sources := make(map[int64]*Account, len(accounts))
	for i := range accounts {
		a := &accounts[i]
		if a.IsShadow() {
			sources[a.ID] = parents[*a.ParentAccountID]
		} else {
			sources[a.ID] = a
		}
	}
	for i := range rows {
		r := &rows[i]
		source := sources[r.ID]
		if source == nil || source.Platform != PlatformAntigravity {
			continue
		}
		var cached *antigravityUsageCache
		if value, ok := s.cache.antigravityCache.Load(source.ID); ok {
			cached, _ = value.(*antigravityUsageCache)
		}
		if cached == nil || now.Sub(cached.timestamp) >= antigravityCacheTTL(cached.usageInfo) {
			s.refreshPoolAntigravityQuota(source, now)
		}
		if cached == nil || cached.usageInfo == nil || cached.usageInfo.IsForbidden || cached.usageInfo.Error != "" || cached.usageInfo.ErrorCode != "" {
			continue
		}
		keys := make([]string, 0, len(cached.usageInfo.AntigravityQuota))
		for key := range cached.usageInfo.AntigravityQuota {
			if len(key) > 80 || strings.ContainsAny(key, "@\r\n") || strings.Contains(key, "://") {
				continue
			}
			keys = append(keys, key)
		}
		sort.Strings(keys)
		if len(keys) > 100 {
			keys = keys[:100]
		}
		allLimited := len(keys) > 0
		for _, key := range keys {
			q := cached.usageInfo.AntigravityQuota[key]
			if q == nil {
				allLimited = false
				continue
			}
			window := opsWindow(key, key, float64(q.Utilization), 1, &cached.timestamp, opsTime(q.ResetTime), now)
			if window.UsedPercent != nil && *window.UsedPercent > 100 {
				window.UsedPercent = nil
				window.Status = "unknown"
			}
			if window.Status != "ok" || window.UsedPercent == nil || *window.UsedPercent < 100 {
				allLimited = false
			}
			r.Windows = append(r.Windows, window)
		}
		if allLimited {
			r.Available = false
		}
		r.UpdatedAt = &cached.timestamp
	}
	return rows
}

func (s *AccountUsageService) refreshPoolAntigravityQuota(account *Account, now time.Time) {
	if s.antigravityQuotaFetcher == nil || !s.antigravityQuotaFetcher.CanFetch(account) {
		return
	}
	select {
	case poolQuotaRefreshSlots <- struct{}{}:
	default:
		return
	}
	stamp := now.Unix()
	previous, exists := s.poolQuotaRefresh.LoadOrStore(account.ID, stamp)
	if exists {
		last, ok := previous.(int64)
		if !ok || stamp-last < 60 || !s.poolQuotaRefresh.CompareAndSwap(account.ID, previous, stamp) {
			<-poolQuotaRefreshSlots
			return
		}
	}
	go func() {
		defer func() { <-poolQuotaRefreshSlots }()
		ctx, cancel := context.WithTimeout(context.Background(), 35*time.Second)
		defer cancel()
		_, _ = s.getAntigravityUsage(ctx, account)
	}()
}
