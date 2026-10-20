//go:build unit

package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestFeishuImportedContactCanBeVerifiedAndBoundWithoutAnOldPassword(t *testing.T) {
	cache := &emailBindCacheStub{data: &service.VerificationCodeData{Code: "123456", CreatedAt: time.Now().UTC(), ExpiresAt: time.Now().UTC().Add(10 * time.Minute)}}
	svc, repo, client := newAuthServiceForEmailBind(t, map[string]string{}, cache, nil)
	ctx := context.Background()
	u, err := client.User.Create().SetEmail("staff@company.example").SetPasswordHash("unknown-generated-password").SetSignupSource("feishu").SetRole(service.RoleUser).SetStatus(service.StatusActive).Save(ctx)
	require.NoError(t, err)
	_, _, err = svc.Login(ctx, u.Email, "unknown-generated-password")
	require.Error(t, err)
	bound, err := svc.BindEmailIdentity(ctx, u.ID, u.Email, "123456", "my-new-password")
	require.NoError(t, err)
	require.Equal(t, u.Email, bound.Email)
	_, loggedIn, err := svc.Login(ctx, u.Email, "my-new-password")
	require.NoError(t, err)
	require.Equal(t, u.ID, loggedIn.ID)
	svc.RecordSuccessfulLogin(ctx, u.ID)
	summaries, err := service.NewUserService(repo, nil, nil, nil).GetProfileIdentitySummaries(ctx, u.ID, nil)
	require.NoError(t, err)
	require.True(t, summaries.Email.Bound)
}
