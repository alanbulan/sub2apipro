package repository

import (
	"context"
	"testing"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/authidentity"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestFeishuContactImportPreservesPasswordAndVerifiedBinding(t *testing.T) {
	repo, client := newUserEntRepo(t)
	ctx := context.Background()
	u := &service.User{Email: "placeholder" + service.FeishuConnectSyntheticEmailDomain, PasswordHash: "original-password", SignupSource: "feishu", Role: service.RoleUser, Status: service.StatusActive}
	require.NoError(t, repo.Create(ctx, u))
	fill := func(email string) bool {
		t.Helper()
		tx, err := client.Tx(ctx)
		require.NoError(t, err)
		defer func() { _ = tx.Rollback() }()
		changed, err := repo.FillFeishuContactEmail(dbent.NewTxContext(ctx, tx), u.ID, email)
		require.NoError(t, err)
		require.NoError(t, tx.Commit())
		return changed
	}
	require.True(t, fill("staff@company.example"))
	u, err := repo.GetByID(ctx, u.ID)
	require.NoError(t, err)
	require.Equal(t, "staff@company.example", u.Email)
	require.Equal(t, "original-password", u.PasswordHash)
	u.Username = "Updated name"
	require.NoError(t, repo.Update(ctx, u, service.UserUpdateFields{Username: true}))
	require.NoError(t, repo.Update(ctx, u, service.UserUpdateFields{Email: true, Username: true}))
	count, err := client.AuthIdentity.Query().Where(authidentity.UserIDEQ(u.ID), authidentity.ProviderTypeEQ("email")).Count(ctx)
	require.NoError(t, err)
	require.Zero(t, count, "profile edits must not backfill an email credential for an imported contact")
	u.Email = "locally-bound@example.com"
	u.PasswordHash = "locally-set-password"
	require.NoError(t, repo.Update(ctx, u, service.UserUpdateFields{Email: true, PasswordHash: true}))
	require.False(t, fill("different@company.example"))
	stored, err := repo.GetByID(ctx, u.ID)
	require.NoError(t, err)
	require.Equal(t, u.Email, stored.Email)
	require.Equal(t, u.PasswordHash, stored.PasswordHash)
}

func TestFeishuContactImportRejectsExistingMailboxAliases(t *testing.T) {
	repo, client := newUserEntRepo(t)
	ctx := context.Background()
	owner := &service.User{Email: "staff@gmail.com", PasswordHash: "owner", Role: service.RoleAdmin, Status: service.StatusActive}
	require.NoError(t, repo.Create(ctx, owner))
	u := &service.User{Email: "placeholder" + service.FeishuConnectSyntheticEmailDomain, PasswordHash: "employee", SignupSource: "feishu", Role: service.RoleUser, Status: service.StatusActive}
	require.NoError(t, repo.Create(ctx, u))
	tx, err := client.Tx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback() }()
	changed, err := repo.FillFeishuContactEmail(dbent.NewTxContext(ctx, tx), u.ID, "staff+contact@gmail.com")
	require.ErrorIs(t, err, service.ErrEmailExists)
	require.False(t, changed)
	require.NoError(t, tx.Commit())
	stored, err := repo.GetByID(ctx, u.ID)
	require.NoError(t, err)
	require.Equal(t, u.Email, stored.Email)
	require.Equal(t, service.RoleUser, stored.Role)
}
