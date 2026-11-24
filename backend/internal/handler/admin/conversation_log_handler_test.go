package admin

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type conversationLogRepositoryStub struct {
	filter service.ConversationLogFilter
	items  []*service.ConversationLog
}

func (s *conversationLogRepositoryStub) Create(context.Context, *service.ConversationLog) (int64, error) {
	return 0, nil
}

func (s *conversationLogRepositoryStub) Finalize(context.Context, int64, *service.ConversationLog) error {
	return nil
}

func (s *conversationLogRepositoryStub) List(_ context.Context, filter service.ConversationLogFilter) ([]*service.ConversationLog, int64, error) {
	s.filter = filter
	return s.items, int64(len(s.items)), nil
}

func (s *conversationLogRepositoryStub) GetByID(context.Context, int64) (*service.ConversationLog, error) {
	return nil, sql.ErrNoRows
}

func TestConversationLogListDefaultsToAllUsers(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &conversationLogRepositoryStub{items: []*service.ConversationLog{
		{ID: 1, UserID: 11, UserEmail: "first@example.com"},
		{ID: 2, UserID: 12, UserEmail: "second@example.com"},
	}}
	h := NewConversationLogHandler(service.NewConversationLogService(repo))
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/conversation-logs", nil)

	h.List(c)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Zero(t, repo.filter.UserID, "an omitted user_id must keep the admin query unscoped")
	require.Contains(t, recorder.Body.String(), `"user_id":11`)
	require.Contains(t, recorder.Body.String(), `"user_id":12`)
}

func TestConversationLogListRejectsInvalidFilters(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewConversationLogHandler(nil)
	tests := []string{
		"/conversation-logs?status=unknown",
		"/conversation-logs?protocol=unknown",
		"/conversation-logs?user_id=0",
		"/conversation-logs?api_key_id=-1",
		"/conversation-logs?start_time=2026-01-02T00:00:00Z&end_time=2026-01-01T00:00:00Z",
	}
	for _, target := range tests {
		t.Run(target, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(recorder)
			c.Request = httptest.NewRequest(http.MethodGet, target, nil)
			h.List(c)
			require.Equal(t, http.StatusBadRequest, recorder.Code)
		})
	}
}

func TestConversationLogGetRejectsInvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewConversationLogHandler(nil)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Params = gin.Params{{Key: "id", Value: "not-a-number"}}

	h.Get(c)

	require.Equal(t, http.StatusBadRequest, recorder.Code)
}
