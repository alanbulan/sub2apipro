package service

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/Wei-Shaw/sub2api/internal/pkg/feishu"
	"github.com/stretchr/testify/require"
)

func TestFeishuSyntheticIdentityCannotBecomeAnEmailLogin(t *testing.T) {
	email := feishuSyntheticEmail("feishu:cli_app:tenant", "ou_employee")
	require.True(t, isReservedEmail(email))
	require.True(t, isReservedEmail(strings.ToUpper(email)))
	require.Equal(t, "feishu", inferLegacySignupSource(email))
	require.Equal(t, email, feishuSyntheticEmail("feishu:cli_app:tenant", "ou_employee"))
	require.NotEqual(t, email, feishuSyntheticEmail("feishu:cli_other:tenant", "ou_employee"))
	require.NotEqual(t, email, feishuSyntheticEmail("feishu:cli_app:other", "ou_employee"))
	require.NotEqual(t, email, feishuSyntheticEmail("feishu:cli_app:tenant", "ou_other"))
}

func TestFeishuContactEmailSelection(t *testing.T) {
	for _, tc := range []struct{ name, enterprise, contact, want string }{
		{"enterprise_first", "Staff@Company.Example", "other@example.com", "staff@company.example"},
		{"contact_fallback", "", "  Staff@Company.Example  ", "staff@company.example"},
		{"invalid_enterprise", "not-email", "staff@example.com", "staff@example.com"},
		{"display_name_not_address", "Staff <staff@company.example>", "", ""},
		{"reserved_domain", "staff@feishu-connect.invalid", "", ""},
		{"too_long", strings.Repeat("a", 256) + "@company.example", "", ""},
		{"missing", "", "", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, feishuContactEmail(&feishu.UserInfo{EnterpriseEmail: tc.enterprise, Email: tc.contact}))
		})
	}
	require.Empty(t, feishuContactEmail(nil))
}

func TestFeishuInitialUsernameFitsSharedValidation(t *testing.T) {
	require.Equal(t, "员工张三", feishuInitialUsername("  员工张三  ", "123"))
	name := feishuInitialUsername(strings.Repeat("员工😀", 40), "123")
	require.LessOrEqual(t, len(name), 100)
	require.True(t, utf8.ValidString(name))
	require.Equal(t, "飞书用户-123", feishuInitialUsername("  ", "123"))
}
