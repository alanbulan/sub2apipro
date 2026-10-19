package migrations

import (
	"strings"
	"testing"
)

func TestFeishuProviderMigrationExtendsEveryConstraint(t *testing.T) {
	raw, err := FS.ReadFile("291_add_feishu_provider.sql")
	if err != nil {
		t.Fatal(err)
	}
	sql := string(raw)
	for _, table := range []string{"users", "auth_identities", "auth_identity_channels", "pending_auth_sessions", "user_provider_default_grants"} {
		start := strings.Index(sql, "ALTER TABLE "+table)
		if start < 0 {
			t.Fatalf("missing table %s", table)
		}
		remaining := sql[start:]
		check := strings.Index(remaining, "CHECK (")
		if check < 0 {
			t.Fatalf("missing check for %s", table)
		}
		statement := remaining[check : strings.Index(remaining[check:], ";")+check]
		for _, provider := range []string{"email", "linuxdo", "wechat", "oidc", "github", "google", "dingtalk", "feishu"} {
			if !strings.Contains(statement, "'"+provider+"'") {
				t.Errorf("%s constraint lost provider %s", table, provider)
			}
		}
	}
}
