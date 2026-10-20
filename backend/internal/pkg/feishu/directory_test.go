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
