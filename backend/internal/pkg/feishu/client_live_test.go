//go:build integration

package feishu

import (
	"context"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// This opt-in probe creates only a short-lived registration challenge. No app
// can be created without a person opening its private URL and authorizing it.
// Never log that URL, the device code, or credentials.
func TestLiveRegistrationContract(t *testing.T) {
	if os.Getenv("FEISHU_LIVE_REGISTRATION") != "1" {
		t.Skip("live Feishu probe runs only in its dedicated GitHub Actions workflow")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	api := NewClient()
	registration, err := api.BeginRegistration(ctx, "企业登录")
	require.NoError(t, err)
	u, err := url.Parse(registration.VerificationURL)
	require.NoError(t, err)
	require.Contains(t, []string{"open.feishu.cn", "accounts.feishu.cn"}, u.Host)
	require.True(t, registration.DeviceCode != "", "registration must include a private device code")
	require.GreaterOrEqual(t, registration.Interval, 5)
	require.Positive(t, registration.ExpireIn)
	require.LessOrEqual(t, registration.ExpireIn, 600)
	t.Logf("Official registration challenge accepted: host=%s interval=%ds lifetime=%ds", u.Host, registration.Interval, registration.ExpireIn)

	timer := time.NewTimer(time.Duration(registration.Interval) * time.Second)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		t.Fatal("registration challenge timed out before the first poll")
	case <-timer.C:
	}
	result, err := api.PollRegistration(ctx, registration.DeviceCode)
	require.NoError(t, err)
	require.True(t, result.ClientID == "" && result.ClientSecret == "", "an unshared challenge must remain unapproved")
	require.Contains(t, []string{"", "authorization_pending", "slow_down"}, result.Error)
	t.Log("Official unapproved challenge remains pending without a provider error")
}
