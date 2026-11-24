package service

import (
	"encoding/json"
	"math"
	"time"
)

type AccountOpsUsageWindow struct {
	Window      string     `json:"window"`
	Label       string     `json:"label"`
	UsedPercent *float64   `json:"used_percent"`
	Status      string     `json:"status"`
	ObservedAt  *time.Time `json:"observed_at,omitempty"`
	ResetsAt    *time.Time `json:"resets_at,omitempty"`
}
type AccountOpsBalanceAccount struct {
	ProviderID       int64                   `json:"provider_id,omitempty"`
	OAuthLifecycle   string                  `json:"oauth_lifecycle,omitempty"`
	AccountID        int64                   `json:"account_id"`
	AccountName      string                  `json:"account_name"`
	GroupID          string                  `json:"group_id"`
	GroupName        string                  `json:"group_name"`
	GroupMode        string                  `json:"group_mode"`
	Platform         string                  `json:"platform"`
	Type             string                  `json:"type"`
	Balance          *float64                `json:"balance"`
	Unit             string                  `json:"unit"`
	BalanceStatus    string                  `json:"balance_status"`
	ProbeEnabled     bool                    `json:"probe_enabled"`
	LastKnownBalance *float64                `json:"last_known_balance,omitempty"`
	ReceivedAt       *time.Time              `json:"received_at"`
	UsageWindows     []AccountOpsUsageWindow `json:"usage_windows"`
}

func opsNumber(v any) (float64, bool) {
	var n float64
	switch x := v.(type) {
	case float64:
		n = x
	case float32:
		n = float64(x)
	case int:
		n = float64(x)
	case int64:
		n = float64(x)
	case json.Number:
		var err error
		n, err = x.Float64()
		if err != nil {
			return 0, false
		}
	default:
		return 0, false
	}
	return n, !math.IsNaN(n) && !math.IsInf(n, 0)
}
func opsTime(v any) *time.Time {
	if t, ok := v.(time.Time); ok {
		return &t
	}
	if t, ok := v.(*time.Time); ok {
		return t
	}
	if s, ok := v.(string); ok {
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			return &t
		}
	}
	if n, ok := opsNumber(v); ok && n > 0 && n < 253402300800 {
		t := time.Unix(int64(n), 0)
		return &t
	}
	return nil
}
func opsFresh(observed, until *time.Time, now time.Time) bool {
	return observed != nil && until != nil && !observed.IsZero() && !observed.After(now) && until.After(*observed) && until.After(now)
}
func opsBalanceAccount(a *Account, now time.Time) AccountOpsBalanceAccount {
	item := AccountOpsBalanceAccount{AccountID: a.ID, AccountName: a.Name, Platform: a.Platform, Type: a.Type, Unit: "USD", BalanceStatus: "unknown", UsageWindows: []AccountOpsUsageWindow{}}
	if a.Type != AccountTypeAPIKey {
		item.Unit = "%"
		return item
	}
	item.ProbeEnabled = upstreamBillingProbeEnabled(a)
	b := poolBalanceSnapshot(a)
	if b == nil {
		return item
	}
	item.BalanceStatus = b.Status
	item.ReceivedAt = b.ReceivedAt
	if unit, ok := b.Data["unit"].(string); ok && unit != "" {
		item.Unit = unit
	}
	if b.Status != "ok" {
		return item
	}
	if unlimited, ok := b.Data["unlimited"].(bool); ok && unlimited {
		item.BalanceStatus = "unsupported"
		return item
	}
	if !opsFresh(b.ReceivedAt, b.FreshUntil, now) {
		if n, ok := opsNumber(b.Data["remaining"]); ok && b.ReceivedAt != nil && !b.ReceivedAt.After(now) {
			item.LastKnownBalance = &n
		}
		item.BalanceStatus = "stale"
		return item
	}
	n, ok := opsNumber(b.Data["remaining"])
	if !ok {
		item.BalanceStatus = "unsupported"
		return item
	}
	item.Balance = &n
	return item
}
func opsWindow(window, label string, value any, scale float64, observed, reset *time.Time, now time.Time) AccountOpsUsageWindow {
	w := AccountOpsUsageWindow{Window: window, Label: label, Status: "unknown", ObservedAt: observed, ResetsAt: reset}
	n, ok := opsNumber(value)
	if !ok || n < 0 {
		return w
	}
	n *= scale
	if math.IsNaN(n) || math.IsInf(n, 0) {
		return w
	}
	if observed == nil || reset == nil {
		return w
	}
	if observed.After(now) || now.Sub(*observed) > 15*time.Minute || !reset.After(now) {
		w.Status = "stale"
		return w
	}
	w.Status = "ok"
	w.UsedPercent = &n
	return w
}
func opsExtraUsageWindows(a *Account, now time.Time) []AccountOpsUsageWindow {
	extra := a.Extra
	windows := []AccountOpsUsageWindow{}
	switch a.Platform {
	case PlatformOpenAI:
		observed := opsTime(extra["codex_usage_updated_at"])
		for _, window := range []string{"5h", "7d"} {
			reset := opsTime(extra["codex_"+window+"_reset_at"])
			if reset == nil && observed != nil {
				if seconds, ok := opsNumber(extra["codex_"+window+"_reset_after_seconds"]); ok && seconds > 0 {
					t := observed.Add(time.Duration(seconds * float64(time.Second)))
					reset = &t
				}
			}
			windows = append(windows, opsWindow(window, window, extra["codex_"+window+"_used_percent"], 1, observed, reset, now))
		}
	case PlatformAnthropic:
		observed := opsTime(extra["passive_usage_sampled_at"])
		windows = append(windows, opsWindow("5h", "5h", extra["session_window_utilization"], 100, observed, a.SessionWindowEnd, now))
		for _, window := range []string{"7d", "7d_oi"} {
			windows = append(windows, opsWindow(window, window, extra["passive_usage_"+window+"_utilization"], 100, observed, opsTime(extra["passive_usage_"+window+"_reset"]), now))
		}
	default:
		windows = append(windows, AccountOpsUsageWindow{Window: "any", Label: "额度", Status: "unsupported"})
	}
	return windows
}
