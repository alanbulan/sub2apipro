package feishu

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDirectoryPaginationDeduplicationAndInactiveMembers(t *testing.T) {
	calls := 0
	api := testClient(func(r *http.Request) string {
		calls++
		require.Equal(t, "Bearer tenant-token", r.Header.Get("Authorization"))
		switch r.URL.Path {
		case "/open-apis/contact/v3/departments/0/children":
			if r.URL.Query().Get("page_token") == "" {
				return `{"code":0,"data":{"items":[{"open_department_id":"dept"}],"has_more":true,"page_token":"next"}}`
			}
			return `{"code":0,"data":{"items":[],"has_more":false}}`
		case "/open-apis/contact/v3/users/find_by_department":
			return `{"code":0,"data":{"items":[{"open_id":"member","name":"Employee","status":{"is_activated":true}},{"open_id":"inactive","name":"Former employee","status":{"is_activated":true,"is_resigned":true}}],"has_more":false}}`
		default:
			t.Fatal(r.URL.Path)
			return ""
		}
	})
	members, err := api.DirectoryMembers(context.Background(), "tenant-token")
	require.NoError(t, err)
	require.Len(t, members, 2)
	require.Equal(t, 4, calls)
	require.True(t, members[0].Active())
	require.False(t, members[1].Active())
}

func TestDirectoryRejectsIncompleteSnapshots(t *testing.T) {
	for _, body := range []string{`{"code":0}`, `{"code":0,"data":{"items":[],"has_more":true}}`, `{"code":99991672,"msg":"permission denied"}`} {
		t.Run(body, func(t *testing.T) {
			api := testClient(func(*http.Request) string { return body })
			members, err := api.DirectoryMembers(context.Background(), "token")
			require.Error(t, err)
			require.Nil(t, members)
		})
	}
}

func TestFeishuDirectoryEnrichesEnterpriseEmailWhenOAuthOmitsIt(t *testing.T) {
	api := testClient(func(r *http.Request) string {
		switch r.URL.Path {
		case "/open-apis/auth/v3/tenant_access_token/internal":
			return `{"code":0,"tenant_access_token":"tenant-token"}`
		case "/open-apis/contact/v3/users/ou_member":
			require.Equal(t, "open_id", r.URL.Query().Get("user_id_type"))
			require.Equal(t, "Bearer tenant-token", r.Header.Get("Authorization"))
			return `{"code":0,"data":{"user":{"open_id":"ou_member","enterprise_email":"employee@company.example","email":"contact@company.example","status":{"is_activated":true}}}}`
		default:
			t.Fatal(r.URL.Path)
			return ""
		}
	})
	info := &UserInfo{OpenID: "ou_member", TenantKey: "company"}
	active, err := api.EnrichContact(context.Background(), "cli_app", "secret", info)
	require.NoError(t, err)
	require.True(t, active)
	require.Equal(t, "employee@company.example", info.EnterpriseEmail)
}

func TestFeishuDirectoryRejectsDifferentIdentityAndInactiveEmployee(t *testing.T) {
	api := testClient(func(*http.Request) string {
		return `{"code":0,"data":{"user":{"open_id":"other","status":{"is_activated":true}}}}`
	})
	_, err := api.DirectoryUser(context.Background(), "token", "ou_member")
	require.Error(t, err)
	api = testClient(func(r *http.Request) string {
		if r.URL.Path == "/open-apis/auth/v3/tenant_access_token/internal" {
			return `{"code":0,"tenant_access_token":"tenant-token"}`
		}
		return `{"code":0,"data":{"user":{"open_id":"ou_member","status":{"is_activated":true,"is_resigned":true}}}}`
	})
	active, err := api.EnrichContact(context.Background(), "cli_app", "secret", &UserInfo{OpenID: "ou_member"})
	require.NoError(t, err)
	require.False(t, active)
}
