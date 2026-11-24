//go:build unit

package admin

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type poolDiagnosticAdminStub struct {
	service.AdminService
	account *service.Account
	reads   int
}

func (s *poolDiagnosticAdminStub) ListAccountsForSchedulerScoreFilter(_ context.Context, _, _, _, _ string, groupID int64, _ string) ([]service.Account, error) {
	if groupID == 1 {
		return []service.Account{*s.account}, nil
	}
	return nil, nil
}
func (s *poolDiagnosticAdminStub) GetAccount(context.Context, int64) (*service.Account, error) {
	s.reads++
	return s.account, nil
}

func poolDiagnosticContext(userID int64, groupID, accountID, body string) (*gin.Context, *httptest.ResponseRecorder) {
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(body))
	c.Params = gin.Params{{Key: "group_id", Value: groupID}, {Key: "id", Value: accountID}}
	if userID > 0 {
		c.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: userID})
	}
	return c, recorder
}

func TestPoolDiagnosticLiveAuthorizationAndCatalogAllowlist(t *testing.T) {
	gin.SetMode(gin.TestMode)
	reader := &poolGroupReaderStub{groups: map[int64][]service.Group{1: {{ID: 1}}}}
	admin := &poolDiagnosticAdminStub{account: &service.Account{ID: 10, Name: "Public name", Platform: service.PlatformOpenAI, Type: service.AccountTypeAPIKey,
		Credentials: map[string]any{"api_key": "private-key", "email": "private@company.com", "model_mapping": map[string]any{"public-model": "private-target", "gpt-image-1": "gpt-image-1"}},
	}}
	h := &AccountHandler{adminService: admin, poolGroupReader: reader}
	c, recorder := poolDiagnosticContext(1, "1", "10", "")
	h.PoolDiagnosticModels(c)
	require.Equal(t, http.StatusOK, recorder.Code)
	require.Contains(t, recorder.Body.String(), "public-model")
	for _, secret := range []string{"private-key", "private@company.com", "private-target", "gpt-image-1", "credentials", "Public name"} {
		require.NotContains(t, recorder.Body.String(), secret)
	}
	for _, test := range []struct {
		userID         int64
		group, account string
		status         int
	}{{0, "1", "10", 401}, {2, "1", "10", 404}, {1, "2", "10", 404}, {1, "1", "999", 404}, {1, "-1", "10", 400}} {
		before := admin.reads
		c, recorder = poolDiagnosticContext(test.userID, test.group, test.account, "")
		h.PoolDiagnosticModels(c)
		require.Equal(t, test.status, recorder.Code)
		require.Equal(t, before, admin.reads)
	}
	reader.groups[1] = nil
	c, recorder = poolDiagnosticContext(1, "1", "10", "")
	h.PoolConnectionTest(c)
	require.Equal(t, http.StatusNotFound, recorder.Code)
}

func TestPoolDiagnosticModelGroupAndUserRestrictions(t *testing.T) {
	target := &poolDiagnosticTarget{groupID: 1, account: &service.Account{Platform: service.PlatformOpenAI, Type: service.AccountTypeAPIKey,
		Credentials:   map[string]any{"model_mapping": map[string]any{"alias": "blocked-target", "allowed": "allowed", "other": "other"}},
		AccountGroups: []service.AccountGroup{{GroupID: 1, AllowedModels: []string{"alias", "allowed"}}},
	}, denied: []string{"blocked-*"}}
	require.True(t, target.allowsModel("allowed"))
	require.False(t, target.allowsModel("alias"))
	require.False(t, target.allowsModel("other"))
	require.False(t, target.allowsModel("*"))
	require.False(t, target.allowsModel(""))
}

func TestPoolDiagnosticRejectsCustomPromptsAndExtraJSONBeforeInference(t *testing.T) {
	h := &AccountHandler{adminService: &poolDiagnosticAdminStub{account: &service.Account{ID: 10}}, poolGroupReader: &poolGroupReaderStub{groups: map[int64][]service.Group{1: {{ID: 1}}}}}
	for _, body := range []string{`{"model_id":"x","prompt":"arbitrary"}`, `{"model_id":"x","mode":"image"}`, `{"model_id":"x"} {}`, `{"model_id":"x","question_kind":"custom"}`} {
		c, recorder := poolDiagnosticContext(1, "1", "10", body)
		h.PoolIntelligenceTest(c)
		require.Equal(t, http.StatusBadRequest, recorder.Code)
	}
}

func TestPoolDiagnosticLimitsReserveActiveAccountAndRelease(t *testing.T) {
	h := &AccountHandler{}
	now := time.Now()
	release, ok := h.beginPoolTest(1, 10, now)
	require.True(t, ok)
	_, ok = h.beginPoolTest(1, 11, now)
	require.False(t, ok)
	_, ok = h.beginPoolTest(2, 10, now.Add(2*time.Minute))
	require.False(t, ok, "an active account stays reserved beyond cooldown")
	release()
	_, ok = h.beginPoolTest(2, 10, time.Now())
	require.False(t, ok)
	release, ok = h.beginPoolTest(2, 10, time.Now().Add(11*time.Second))
	require.True(t, ok)
	release()
}

func TestPoolDiagnosticLimitsBoundUsersAndGlobalConcurrency(t *testing.T) {
	h := &AccountHandler{}
	now := time.Now()
	for id := int64(1); id <= 4; id++ {
		release, ok := h.beginPoolTest(1, id, now)
		require.True(t, ok)
		release()
	}
	_, ok := h.beginPoolTest(1, 5, now)
	require.False(t, ok, "user minute budget is exhausted")
	releases := []func(){}
	for id := int64(2); id <= 5; id++ {
		release, ok := h.beginPoolTest(id, id+10, now)
		require.True(t, ok)
		releases = append(releases, release)
	}
	_, ok = h.beginPoolTest(6, 20, now)
	require.False(t, ok, "global concurrency is bounded")
	for _, release := range releases {
		release()
	}
	release, ok := h.beginPoolTest(1, 20, now.Add(time.Minute))
	require.True(t, ok)
	release()
}
