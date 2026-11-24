package service

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/tlsfingerprint"
)

type poolBalanceCacheEntry struct {
	identity    string
	nextAttempt time.Time
	snapshot    *UpstreamBalanceSnapshot
}

var poolBalances = struct {
	sync.Mutex
	entries map[int64]poolBalanceCacheEntry
	slots   chan struct{}
}{entries: make(map[int64]poolBalanceCacheEntry), slots: make(chan struct{}, 4)}

func poolBalanceIdentity(account *Account) string {
	// Include all credential/header/proxy inputs; never reuse a previous
	// account identity's balance after the administrator changes its key.
	raw, _ := json.Marshal([]any{account.Platform, account.Type, account.Credentials, account.ProxyID, account.Proxy})
	return fmt.Sprintf("%x", sha256.Sum256(raw))
}

func poolBalanceSnapshot(account *Account) *UpstreamBalanceSnapshot {
	poolBalances.Lock()
	entry, ok := poolBalances.entries[account.ID]
	poolBalances.Unlock()
	if ok && entry.identity == poolBalanceIdentity(account) && entry.snapshot != nil {
		return entry.snapshot
	}
	if legacy := decodeUpstreamBillingProbeSnapshot(account.Extra); legacy != nil {
		return legacy.Balance
	}
	return nil
}

// RefreshPoolBalances reads the key-scoped quota endpoint in the background.
// It sends no inference request and never changes canonical billing rates.
func (s *UpstreamBillingProbeService) RefreshPoolBalances(accounts []Account) {
	if s == nil || s.accountTestService == nil || s.accountTestService.httpUpstream == nil {
		return
	}
	for i := range accounts {
		account := accounts[i]
		if !isUpstreamBillingProbeAccount(&account) || !upstreamBillingProbeEnabled(&account) {
			continue
		}
		now, identity := time.Now().UTC(), poolBalanceIdentity(&account)
		poolBalances.Lock()
		entry, exists := poolBalances.entries[account.ID]
		if exists && entry.identity == identity && now.Before(entry.nextAttempt) {
			poolBalances.Unlock()
			continue
		}
		select {
		case poolBalances.slots <- struct{}{}:
		default:
			poolBalances.Unlock()
			continue
		}
		// Bound retained state even if many historical accounts are visited.
		if len(poolBalances.entries) >= 2048 {
			for id, old := range poolBalances.entries {
				if old.nextAttempt.Before(now) {
					delete(poolBalances.entries, id)
				}
			}
			if len(poolBalances.entries) >= 2048 {
				<-poolBalances.slots
				poolBalances.Unlock()
				continue
			}
		}
		if entry.identity != identity {
			entry = poolBalanceCacheEntry{identity: identity}
		}
		entry.nextAttempt = now.Add(time.Minute)
		poolBalances.entries[account.ID] = entry
		poolBalances.Unlock()
		go func(account Account, previous poolBalanceCacheEntry, observed time.Time) {
			defer func() { <-poolBalances.slots }()
			parent := s.parentCtx
			if parent == nil {
				parent = context.Background()
			}
			ctx, cancel := context.WithTimeout(parent, 15*time.Second)
			defer cancel()
			result := s.fetchPoolBalance(ctx, &account, observed)
			snapshot := upstreamBalanceSnapshotFromResult(result, previous.snapshot, 5, observed)
			previous.snapshot = snapshot
			if result.statusCode == http.StatusUnauthorized || result.statusCode == http.StatusForbidden || snapshot.Status == UpstreamBillingProbeStatusUnsupported {
				previous.nextAttempt = observed.Add(time.Hour)
			} else if result.statusCode == http.StatusTooManyRequests {
				previous.nextAttempt = observed.Add(15 * time.Minute)
			}
			poolBalances.Lock()
			if current := poolBalances.entries[account.ID]; current.identity == previous.identity {
				poolBalances.entries[account.ID] = previous
			}
			poolBalances.Unlock()
		}(account, entry, now)
	}
}

func (s *UpstreamBillingProbeService) fetchPoolBalance(ctx context.Context, account *Account, now time.Time) poolBalanceHTTPResult {
	result := poolBalanceHTTPResult{failure: "request_failed"}
	baseURL, apiKey := account.GetCredential("base_url"), account.GetCredential("api_key")
	if account.RoutesProtocolByInbound() && account.IsAdaptiveAPIProtocol() {
		baseURL = account.GetCNProtocolBaseURL(APIProtocolChatCompletions)
	}
	// Only administrator-configured relay URLs are eligible. Provider APIs
	// have no Sub2API quota endpoint and must never receive these extra reads.
	if apiKey == "" || baseURL == "" || upstreamBillingProbeTargetIsOfficialAPI(baseURL) {
		result.failure = "unsupported"
		return result
	}
	normalized, err := s.accountTestService.validateUpstreamBaseURL(baseURL)
	if err != nil {
		result.failure = "invalid_base_url"
		return result
	}
	proxyURL := ""
	if account.ProxyID != nil {
		if account.Proxy == nil || account.Proxy.ID != *account.ProxyID {
			result.failure = "proxy_unavailable"
			return result
		}
		proxyURL = account.Proxy.URL()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, upstreamBalanceProbeURL(normalized, now), nil)
	if err != nil {
		return result
	}
	profile := HTTPUpstreamProfileDefault
	if account.Platform == PlatformOpenAI {
		profile = HTTPUpstreamProfileOpenAI
	}
	req = req.WithContext(WithHTTPUpstreamRedirectsDisabled(WithHTTPUpstreamProfile(ctx, profile)))
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)
	account.ApplyHeaderOverrides(req.Context(), req.Header)
	var tlsProfile *tlsfingerprint.Profile
	if s.accountTestService.tlsFPProfileService != nil {
		tlsProfile = s.accountTestService.tlsFPProfileService.ResolveTLSProfile(account)
	}
	resp, err := s.accountTestService.httpUpstream.DoWithTLS(req, proxyURL, account.ID, account.Concurrency, tlsProfile)
	if err != nil || resp == nil || resp.Body == nil {
		return result
	}
	defer func() { _ = resp.Body.Close() }()
	result.statusCode = resp.StatusCode
	body, err := io.ReadAll(io.LimitReader(resp.Body, upstreamBalanceProbeMaxBodyBytes+1))
	if err != nil || len(body) > upstreamBalanceProbeMaxBodyBytes {
		result.failure = "response_too_large"
		return result
	}
	result.body, result.failure = body, ""
	return result
}
