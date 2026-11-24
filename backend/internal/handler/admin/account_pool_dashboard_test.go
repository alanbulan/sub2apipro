//go:build unit

package admin

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type poolGroupReaderStub struct{ groups map[int64][]service.Group }

func (s *poolGroupReaderStub) GetUserGroupDeniedModels(context.Context, int64) (map[int64][]string, error) {
	return nil, nil
}

func (s *poolGroupReaderStub) PoolSourceGroups(_ context.Context, id int64) ([]service.Group, error) {
	return s.groups[id], nil
}

type poolAccountsStub struct {
	service.AdminService
	queries []int64
}

func (s *poolAccountsStub) ListAccountsForSchedulerScoreFilter(_ context.Context, _, _, _, _ string, groupID int64, _ string) ([]service.Account, error) {
	s.queries = append(s.queries, groupID)
	return []service.Account{{ID: groupID * 10, Name: "Public source name", Credentials: map[string]any{"api_key": "private-key", "email": "private-account@company.com"}, Platform: service.PlatformOpenAI, Status: service.StatusActive, Schedulable: true}}, nil
}
func TestUserPoolAuthorizesBeforeCacheAndIgnoresForgedGroupParameters(t *testing.T) {
	gin.SetMode(gin.TestMode)
	reader := &poolGroupReaderStub{groups: map[int64][]service.Group{1: {{ID: 1, Name: "Mine"}}, 2: {{ID: 2, Name: "Other"}}}}
	accounts := &poolAccountsStub{}
	h := &AccountHandler{adminService: accounts, poolGroupReader: reader}
	request := func(userID int64) *httptest.ResponseRecorder {
		recorder := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(recorder)
		ctx.Request = httptest.NewRequest(http.MethodGet, "/user/pool-dashboard?user_id=2&group_id=2", nil)
		if userID > 0 {
			ctx.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: userID})
		}
		h.SharedPoolDashboard(ctx)
		return recorder
	}
	first := request(1)
	require.Equal(t, 200, first.Code)
	require.Contains(t, first.Body.String(), "Mine")
	require.NotContains(t, first.Body.String(), "Other")
	require.Contains(t, first.Body.String(), "Public source name")
	require.NotContains(t, first.Body.String(), "private-account")
	require.NotContains(t, first.Body.String(), "private-key")
	second := request(2)
	require.Contains(t, second.Body.String(), "Other")
	require.NotContains(t, second.Body.String(), "Mine")
	reader.groups[1] = nil
	revoked := request(1)
	require.Contains(t, revoked.Body.String(), `"groups":[]`)
	require.Equal(t, []int64{1, 2}, accounts.queries)
	require.Equal(t, http.StatusUnauthorized, request(0).Code)
}
