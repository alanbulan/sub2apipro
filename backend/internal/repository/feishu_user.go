package repository

import (
	"context"
	"fmt"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	dbuser "github.com/Wei-Shaw/sub2api/ent/user"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

// FillFeishuContactEmail only replaces a provisioned placeholder. It must never
// overwrite a concurrent email binding, password or email authentication identity.
func (r *userRepository) FillFeishuContactEmail(ctx context.Context, userID int64, email string) (bool, error) {
	tx := dbent.TxFromContext(ctx)
	if tx == nil {
		return false, fmt.Errorf("Feishu contact import requires a transaction")
	}
	client := tx.Client()
	release, err := lockRepositoryScopedKeys(ctx, client, txAwareSQLExecutor(ctx, r.sql, r.client), normalizedEmailUniquenessLockKey(email), emailAliasUniquenessLockKey(email))
	if err != nil {
		return false, err
	}
	defer release()
	owner, exists, err := emailAliasOwnerIDWithClient(ctx, client, email, userID)
	if err != nil {
		return false, err
	}
	if exists && owner != userID {
		return false, service.ErrEmailExists
	}
	count, err := client.User.Update().Where(dbuser.IDEQ(userID), dbuser.SignupSourceEQ("feishu"), dbuser.EmailHasSuffix(service.FeishuConnectSyntheticEmailDomain)).SetEmail(email).Save(ctx)
	return count > 0, translatePersistenceError(err, service.ErrUserNotFound, service.ErrEmailExists)
}
