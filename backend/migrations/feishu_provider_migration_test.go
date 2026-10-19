package migrations

import (
	"regexp"
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

// A clean upstream merge can add a later constraint that removes Feishu. Check
// the final definitions, not just the presence of our historical migration.
func TestLatestAuthProviderConstraintsPreserveFeishu(t *testing.T) {
	entries, err := FS.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	var sql strings.Builder
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".sql") {
			raw, err := FS.ReadFile(entry.Name())
			if err != nil {
				t.Fatal(err)
			}
			sql.Write(raw)
			sql.WriteString("\n")
		}
	}
	for _, name := range []string{"users_signup_source_check", "auth_identities_provider_type_check", "auth_identity_channels_provider_type_check", "pending_auth_sessions_provider_type_check", "user_provider_default_grants_provider_type_check"} {
		pattern := regexp.MustCompile(`(?is)ADD\s+CONSTRAINT\s+` + regexp.QuoteMeta(name) + `\s+CHECK\s*\(([^;]+);`)
		matches := pattern.FindAllString(sql.String(), -1)
		if len(matches) == 0 {
			t.Errorf("missing constraint %s", name)
			continue
		}
		if !strings.Contains(matches[len(matches)-1], "'feishu'") {
			t.Errorf("latest %s removes Feishu; append a new migration preserving upstream providers and Feishu", name)
		}
	}
}
