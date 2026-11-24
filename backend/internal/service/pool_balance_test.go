package service

import (
	"context"
	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestParseUpstreamBalanceResponseModes(t *testing.T) {
	for _, tc := range []struct {
		name string
		body string
		want map[string]any
	}{
		{
			name: "wallet keeps only balance fields",
			body: `{"mode":"unrestricted","isValid":true,"planName":"钱包余额","remaining":12.34,"unit":"USD","balance":12.34,
				"usage":{"today":{"cost":1}},"daily_usage":[],"model_stats":[{"model":"gpt-5"}],"secret":"must-not-persist"}`,
			want: map[string]any{
				"mode": "unrestricted", "is_valid": true, "plan_name": "钱包余额", "unit": "USD",
				"remaining": 12.34, "wallet_balance": 12.34,
			},
		},
		{
			name: "negative wallet balance stays a balance",
			body: `{"mode":"unrestricted","isValid":true,"remaining":-1,"balance":-1,"unit":"USD"}`,
			want: map[string]any{
				"mode": "unrestricted", "is_valid": true, "unit": "USD", "remaining": -1.0, "wallet_balance": -1.0,
			},
		},
		{
			name: "subscription without limits is unlimited",
			body: `{"mode":"unrestricted","isValid":true,"planName":"Claude Max","unit":"USD","remaining":-1,
				"subscription":{"daily_usage_usd":3,"weekly_usage_usd":4,"monthly_usage_usd":5,
				"daily_limit_usd":null,"weekly_limit_usd":0,"monthly_limit_usd":null,
				"expires_at":"2026-08-01T08:00:00+08:00"}}`,
			want: map[string]any{
				"mode": "unrestricted", "is_valid": true, "plan_name": "Claude Max", "unit": "USD",
				"unlimited": true, "expires_at": "2026-08-01T00:00:00Z",
			},
		},
		{
			name: "subscription reports configured windows",
			body: `{"mode":"unrestricted","isValid":true,"planName":"Pro","unit":"USD","remaining":7.5,
				"subscription":{"daily_usage_usd":2.5,"weekly_usage_usd":4,"monthly_usage_usd":30,
				"daily_limit_usd":10,"weekly_limit_usd":null,"monthly_limit_usd":100,
				"expires_at":"2026-08-01T00:00:00Z"}}`,
			want: map[string]any{
				"mode": "unrestricted", "is_valid": true, "plan_name": "Pro", "unit": "USD", "remaining": 7.5,
				"windows": []map[string]any{
					{"window": "daily", "limit": 10.0, "used": 2.5},
					{"window": "monthly", "limit": 100.0, "used": 30.0},
				},
				"expires_at": "2026-08-01T00:00:00Z",
			},
		},
		{
			name: "subscription without an active subscription has no remaining",
			body: `{"mode":"unrestricted","isValid":true,"planName":"Pro","unit":"USD"}`,
			want: map[string]any{"mode": "unrestricted", "is_valid": true, "plan_name": "Pro", "unit": "USD"},
		},
		{
			name: "quota limited key",
			body: `{"mode":"quota_limited","isValid":true,"status":"quota_exhausted",
				"quota":{"limit":100,"used":100,"remaining":0,"unit":"USD"},"remaining":0,"unit":"USD",
				"rate_limits":[
					{"window":"5h","limit":5,"used":1.25,"remaining":3.75,"window_start":"2026-07-13T00:00:00Z","reset_at":"2026-07-13T05:00:00Z"},
					{"window":"7d","limit":20,"used":2,"remaining":18}
				],
				"expires_at":"2026-09-01T00:00:00Z","days_until_expiry":50}`,
			want: map[string]any{
				"mode": "quota_limited", "is_valid": true, "key_status": "quota_exhausted", "unit": "USD",
				"remaining": 0.0, "quota_limit": 100.0, "quota_used": 100.0,
				"windows": []map[string]any{
					{"window": "5h", "limit": 5.0, "used": 1.25, "reset_at": "2026-07-13T05:00:00Z"},
					{"window": "7d", "limit": 20.0, "used": 2.0},
				},
				"expires_at": "2026-09-01T00:00:00Z",
			},
		},
		{
			name: "quota remaining falls back to the quota object",
			body: `{"mode":"quota_limited","isValid":true,"status":"active","quota":{"limit":10,"used":4,"remaining":6}}`,
			want: map[string]any{
				"mode": "quota_limited", "is_valid": true, "key_status": "active",
				"remaining": 6.0, "quota_limit": 10.0, "quota_used": 4.0,
			},
		},
		{
			name: "legacy response without mode",
			body: `{"isValid":true,"planName":"钱包余额","remaining":3,"unit":"USD"}`,
			want: map[string]any{"is_valid": true, "plan_name": "钱包余额", "unit": "USD", "remaining": 3.0},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			data, err := parseUpstreamBalanceResponse([]byte(tc.body))
			require.NoError(t, err)
			require.Equal(t, tc.want, data)
		})
	}
}

func TestParseUpstreamBalanceResponseSanitizesUpstreamText(t *testing.T) {
	longName := strings.Repeat("名", upstreamBalanceProbeMaxPlanNameRunes+10)
	data, err := parseUpstreamBalanceResponse([]byte(`{
		"mode":"something_new",
		"isValid":false,
		"status":"not active",
		"unit":"US$",
		"planName":"  Pro\u0000\u202e plan\n  ",
		"remaining":1,
		"rate_limits":[
			{"window":"../5h","limit":5,"used":1},
			{"window":"1d","limit":0,"used":1},
			{"window":"1d","limit":3,"used":1,"reset_at":"not-a-time"}
		]
	}`))
	require.NoError(t, err)
	require.Equal(t, map[string]any{
		"is_valid":  false,
		"plan_name": "Pro plan",
		"remaining": 1.0,
		"windows":   []map[string]any{{"window": "1d", "limit": 3.0, "used": 1.0}},
	}, data)

	data, err = parseUpstreamBalanceResponse([]byte(`{"isValid":true,"planName":"` + longName + `"}`))
	require.NoError(t, err)
	require.Equal(t, strings.Repeat("名", upstreamBalanceProbeMaxPlanNameRunes), data["plan_name"])

	var rateLimits []string
	for range upstreamBalanceProbeMaxWindows + 4 {
		rateLimits = append(rateLimits, `{"window":"5h","limit":1,"used":0}`)
	}
	data, err = parseUpstreamBalanceResponse([]byte(`{"isValid":true,"rate_limits":[` + strings.Join(rateLimits, ",") + `]}`))
	require.NoError(t, err)
	require.Len(t, data["windows"], upstreamBalanceProbeMaxWindows)
}

func TestParseUpstreamBalanceResponseRejectsForeignPayloads(t *testing.T) {
	for _, body := range []string{
		``,
		`<html>not found</html>`,
		`[]`,
		`{"remaining":12.34,"unit":"USD"}`,
		`{"isValid":"true","remaining":1}`,
		`{"isValid":true,"remaining":"12.34"}`,
		`{"isValid":true,"quota":{"limit":"100"}}`,
		`{"error":{"message":"Invalid URL (GET /v1/usage)","type":"invalid_request_error"}}`,
	} {
		_, err := parseUpstreamBalanceResponse([]byte(body))
		require.Error(t, err, body)
	}
}

func TestUpstreamBalanceProbeURLAsksForSmallestStatisticsWindow(t *testing.T) {
	now := time.Date(2026, time.July, 13, 23, 30, 0, 0, time.FixedZone("UTC+8", 8*3600))
	require.Equal(t,
		"https://relay.example/v1/usage?days=1&end_date=2026-07-13&start_date=2026-07-13",
		upstreamBalanceProbeURL("https://relay.example/v1", now))
	require.Equal(t,
		"https://relay.example/prefix/v1/usage?days=1&end_date=2026-07-13&start_date=2026-07-13&tenant=a",
		upstreamBalanceProbeURL("https://relay.example/prefix?tenant=a", now))
}

func TestPoolBalanceFetchKeepsTransportRulesAndRejectsOfficialEndpoints(t *testing.T) {
	upstream := &httpUpstreamRecorder{responses: []*http.Response{{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"isValid":true,"remaining":12.34,"unit":"USD"}`))}}}
	service := NewUpstreamBillingProbeService(nil, &AccountTestService{httpUpstream: upstream, cfg: &config.Config{}}, nil)
	account := &Account{ID: 87123, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Credentials: map[string]any{"api_key": "fixture-only", "base_url": "https://relay.example/v1"}}
	result := service.fetchPoolBalance(context.Background(), account, time.Now())
	require.Empty(t, result.failure)
	require.Equal(t, 200, result.statusCode)
	require.Len(t, upstream.requests, 1)
	require.Equal(t, "/v1/usage", upstream.requests[0].URL.Path)
	require.Equal(t, "Bearer fixture-only", upstream.requests[0].Header.Get("Authorization"))
	identity := poolBalanceIdentity(account)
	account.Credentials["api_key"] = "replacement-fixture"
	require.NotEqual(t, identity, poolBalanceIdentity(account))
	account.Credentials["base_url"] = "https://api.openai.com"
	result = service.fetchPoolBalance(context.Background(), account, time.Now())
	require.Equal(t, "unsupported", result.failure)
	require.Len(t, upstream.requests, 1, "official provider must receive no quota request")
}
