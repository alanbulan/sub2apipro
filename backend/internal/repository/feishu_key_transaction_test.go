package repository

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/setting"
	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

// Exercise the same APIKeyService.Create call used by Feishu provisioning with
// real canonical repositories and PostgreSQL, not the in-memory service doubles.
func testCanonicalFeishuKeyTransaction(t *testing.T, ctx context.Context, db *sql.DB) {
	t.Helper()
	client := dbent.NewClient(dbent.Driver(entsql.OpenDB(dialect.Postgres, db)))
	users := NewUserRepository(client, db)
	keys := NewAPIKeyRepository(client, db)
	groups := NewGroupRepository(client, db)
	keyService := service.NewAPIKeyService(keys, users, groups, nil, nil, nil, &config.Config{})
	u := client.User.Create().SetEmail("transaction-employee@fixture.invalid").SetPasswordHash("synthetic-only").SetUsername("Fixture employee").SetSignupSource("feishu").SetRole(service.RoleUser).SetStatus(service.StatusActive).SaveX(ctx)
	g := client.Group.Create().SetName("Fixture exclusive pool").SetPlatform(service.PlatformOpenAI).SetStatus(service.StatusActive).SetIsExclusive(true).SaveX(ctx)
	req := service.CreateAPIKeyRequest{Name: "Employee dedicated fixture", GroupID: &g.ID}
	_, err := keyService.Create(ctx, u.ID, req)
	require.ErrorIs(t, err, service.ErrGroupNotAllowed, "normal calls cannot bypass exclusive group grants")
	receiptKey := fmt.Sprintf("feishu_key_issuance:%d", u.ID)
	issue := func(commit bool) int64 {
		tx, err := client.Tx(ctx)
		require.NoError(t, err)
		t.Cleanup(func() { _ = tx.Rollback() })
		txCtx := dbent.NewTxContext(ctx, tx)
		_, err = tx.Client().User.UpdateOneID(u.ID).SetBalance(10).AddAllowedGroupIDs(g.ID).Save(txCtx)
		require.NoError(t, err)
		inside, err := users.GetByID(txCtx, u.ID)
		require.NoError(t, err)
		require.Equal(t, 10.0, inside.Balance)
		require.Contains(t, inside.AllowedGroups, g.ID, "the permission check sees the uncommitted employee group grant")
		outside, err := users.GetByID(ctx, u.ID)
		require.NoError(t, err)
		require.Zero(t, outside.Balance)
		require.NotContains(t, outside.AllowedGroups, g.ID)
		key, err := keyService.Create(txCtx, u.ID, req)
		require.NoError(t, err)
		require.NotZero(t, key.ID)
		inKey, err := keys.GetByID(txCtx, key.ID)
		require.NoError(t, err)
		require.Equal(t, u.ID, inKey.UserID)
		_, err = keys.GetByID(ctx, key.ID)
		require.ErrorIs(t, err, service.ErrAPIKeyNotFound, "the key cannot leak out before the grant and receipt commit")
		_, err = tx.Client().Setting.Create().SetKey(receiptKey).SetValue(fmt.Sprintf(`{"key_id":%d,"user_id":%d,"grant":10}`, key.ID, u.ID)).Save(txCtx)
		require.NoError(t, err)
		if commit {
			require.NoError(t, tx.Commit())
		} else {
			require.NoError(t, tx.Rollback())
		}
		return key.ID
	}
	rolledBack := issue(false)
	_, err = keys.GetByID(ctx, rolledBack)
	require.ErrorIs(t, err, service.ErrAPIKeyNotFound)
	afterRollback, err := users.GetByID(ctx, u.ID)
	require.NoError(t, err)
	require.Zero(t, afterRollback.Balance)
	require.NotContains(t, afterRollback.AllowedGroups, g.ID)
	require.False(t, client.Setting.Query().Where(setting.KeyEQ(receiptKey)).ExistX(ctx))
	committed := issue(true)
	final, err := keys.GetByID(ctx, committed)
	require.NoError(t, err)
	require.Equal(t, g.ID, *final.GroupID)
	afterCommit, err := users.GetByID(ctx, u.ID)
	require.NoError(t, err)
	require.Equal(t, 10.0, afterCommit.Balance)
	require.Contains(t, afterCommit.AllowedGroups, g.ID)
	require.True(t, client.Setting.Query().Where(setting.KeyEQ(receiptKey)).ExistX(ctx))
}
