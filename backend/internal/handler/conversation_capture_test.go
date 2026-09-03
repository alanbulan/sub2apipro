package handler

import (
	"bytes"
	"compress/gzip"
	"context"
	"database/sql"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	pkghttputil "github.com/Wei-Shaw/sub2api/internal/pkg/httputil"
	servermiddleware "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type conversationLogRepositoryStub struct {
	mu        sync.Mutex
	nextID    int64
	created   []*service.ConversationLog
	finalized map[int64]*service.ConversationLog
}

func cloneConversationLog(item *service.ConversationLog) *service.ConversationLog {
	if item == nil {
		return nil
	}
	clone := *item
	return &clone
}

func (r *conversationLogRepositoryStub) Create(_ context.Context, item *service.ConversationLog) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.nextID++
	r.created = append(r.created, cloneConversationLog(item))
	return r.nextID, nil
}

func (r *conversationLogRepositoryStub) Finalize(_ context.Context, id int64, item *service.ConversationLog) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.finalized == nil {
		r.finalized = make(map[int64]*service.ConversationLog)
	}
	r.finalized[id] = cloneConversationLog(item)
	return nil
}

func (r *conversationLogRepositoryStub) List(context.Context, service.ConversationLogFilter) ([]*service.ConversationLog, int64, error) {
	return nil, 0, nil
}

func (r *conversationLogRepositoryStub) GetByID(context.Context, int64) (*service.ConversationLog, error) {
	return nil, sql.ErrNoRows
}

func TestConversationCapturePersistsDecodedRequestAndResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &conversationLogRepositoryStub{}
	h := &GatewayHandler{conversationLogService: service.NewConversationLogService(repo)}
	router := gin.New()
	router.Use(func(c *gin.Context) {
		groupID := int64(9)
		c.Set(string(servermiddleware.ContextKeyAPIKey), &service.APIKey{
			ID: 7, UserID: 11, Name: "analysis-key", GroupID: &groupID,
			User: &service.User{Email: "user@example.com"},
		})
		ctx := context.WithValue(c.Request.Context(), ctxkey.ClientRequestID, "request-1")
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	})
	router.POST("/v1/chat/completions", h.ConversationCapture("openai_chat_completions"), func(c *gin.Context) {
		body, err := pkghttputil.ReadRequestBodyWithPrealloc(c.Request)
		require.NoError(t, err)
		require.JSONEq(t, `{"model":"gpt-test","stream":true,"messages":[{"role":"user","content":"hello"}]}`, string(body))
		c.Data(http.StatusCreated, "application/json", []byte(`{"choices":[{"message":{"role":"assistant","content":"world"}}]}`))
	})

	requestJSON := `{"model":"gpt-test","stream":true,"messages":[{"role":"user","content":"hello"}]}`
	var compressed bytes.Buffer
	zw := gzip.NewWriter(&compressed)
	_, err := io.WriteString(zw, requestJSON)
	require.NoError(t, err)
	require.NoError(t, zw.Close())
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(compressed.Bytes()))
	req.Header.Set("Content-Encoding", "gzip")
	req.Header.Set("session-id", "conversation-1")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	require.Equal(t, http.StatusCreated, recorder.Code)
	require.Len(t, repo.created, 1)
	item := repo.finalized[1]
	require.NotNil(t, item)
	require.Equal(t, "request-1", item.RequestID)
	require.Equal(t, "conversation-1", item.SessionID)
	require.Equal(t, int64(11), item.UserID)
	require.Equal(t, "user@example.com", item.UserEmail)
	require.Equal(t, "gpt-test", item.Model)
	require.True(t, item.Stream)
	require.Equal(t, "completed", item.Status)
	require.Equal(t, http.StatusCreated, item.StatusCode)
	require.JSONEq(t, requestJSON, item.RequestBody)
	require.JSONEq(t, recorder.Body.String(), item.ResponseBody)
	require.Equal(t, int64(len(requestJSON)), item.RequestBytes)
	require.Equal(t, int64(recorder.Body.Len()), item.ResponseBytes)
	require.False(t, item.RequestTruncated)
	require.False(t, item.ResponseTruncated)
}

func TestConversationCaptureRetainsRepeatedRequestIDs(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &conversationLogRepositoryStub{}
	h := &GatewayHandler{conversationLogService: service.NewConversationLogService(repo)}
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(string(servermiddleware.ContextKeyAPIKey), &service.APIKey{ID: 1, UserID: 2})
		ctx := context.WithValue(c.Request.Context(), ctxkey.ClientRequestID, "reused-request-id")
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	})
	router.POST("/v1/messages", h.ConversationCapture("anthropic_messages"), func(c *gin.Context) {
		_, _ = io.Copy(io.Discard, c.Request.Body)
		c.JSON(http.StatusOK, gin.H{"content": "ok"})
	})

	for range 2 {
		recorder := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/v1/messages", bytes.NewBufferString(`{"model":"claude-test"}`))
		router.ServeHTTP(recorder, req)
		require.Equal(t, http.StatusOK, recorder.Code)
	}

	require.Len(t, repo.created, 2)
	require.Len(t, repo.finalized, 2)
	require.Equal(t, "reused-request-id", repo.created[0].RequestID)
	require.Equal(t, "reused-request-id", repo.created[1].RequestID)
}

func TestShouldCaptureConversationSkipsNonConversationOperations(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses/input_tokens", nil)
	require.False(t, shouldCaptureConversation(c, "openai_responses"))

	c.Params = gin.Params{{Key: "modelAction", Value: "/gemini-test:countTokens"}}
	require.False(t, shouldCaptureConversation(c, "gemini_generate_content"))
	c.Params = gin.Params{{Key: "modelAction", Value: "/gemini-test:generateContent"}}
	require.True(t, shouldCaptureConversation(c, "gemini_generate_content"))
	c.Params = gin.Params{{Key: "modelAction", Value: "/gemini-test:streamGenerateContent"}}
	require.True(t, shouldCaptureConversation(c, "gemini_generate_content"))
}

func TestBoundedConversationCaptureIsConcurrentAndBounded(t *testing.T) {
	var capture boundedConversationCapture
	chunk := bytes.Repeat([]byte("x"), conversationCaptureBodyLimit/2)
	var wg sync.WaitGroup
	for range 3 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			capture.write(chunk)
		}()
	}
	wg.Wait()
	body, total, truncated := capture.snapshot()
	require.Len(t, body, conversationCaptureBodyLimit)
	require.Equal(t, int64(len(chunk)*3), total)
	require.True(t, truncated)
}

func TestBoundedConversationBodyKeepsTruncatedTextValidUTF8(t *testing.T) {
	body := append(bytes.Repeat([]byte("x"), conversationCaptureBodyLimit-1), []byte("\u20ac")...)

	stored, total, truncated := boundedConversationBody(body)

	require.True(t, truncated)
	require.Equal(t, int64(len(body)), total)
	require.True(t, utf8.ValidString(stored))
	require.Contains(t, stored, "?")
}

func TestOpenAIWSConversationRecorderPersistsClientVisibleTurn(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &conversationLogRepositoryStub{}
	conversationService := service.NewConversationLogService(repo)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodGet, "/v1/responses", nil)
	c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), ctxkey.ClientRequestID, "connection-1"))
	c.Request.Header.Set("session-id", "ws-session-1")
	apiKey := &service.APIKey{ID: 7, UserID: 11, Name: "analysis-key", User: &service.User{Email: "user@example.com"}}
	recorder := newOpenAIWSConversationRecorder(conversationService, c, apiKey)
	requestBody := []byte(`{"type":"response.create","model":"gpt-test","input":"hello"}`)
	recorder.beginTurn(1, requestBody, "gpt-test", time.Now())

	result := &service.OpenAIForwardResult{
		Model: "gpt-test", OpenAIWSMode: true, UpstreamTerminalEvent: "response.completed",
		Duration: 42 * time.Millisecond,
	}
	recorder.finishTurn(1, result, nil)
	require.Empty(t, repo.finalized)
	recorder.appendResponseFrame(1, []byte(`{"type":"response.output_text.delta","delta":"world"}`))
	require.Empty(t, repo.finalized)
	recorder.appendResponseFrame(1, []byte(`{"type":"response.completed","response":{"id":"resp_1"}}`))

	item := repo.finalized[1]
	require.NotNil(t, item)
	require.Equal(t, "openai_responses_ws", item.Protocol)
	require.Equal(t, "ws-session-1", item.SessionID)
	require.Equal(t, "gpt-test", item.Model)
	require.Equal(t, "completed", item.Status)
	require.Equal(t, http.StatusSwitchingProtocols, item.StatusCode)
	require.Equal(t, int64(42), item.DurationMs)
	require.JSONEq(t, string(requestBody), item.RequestBody)
	require.Contains(t, item.ResponseBody, "response.output_text.delta")
	require.Contains(t, item.ResponseBody, "response.completed")
}

func TestOpenAIWSConversationRecorderMarksUnfinishedTurnIncomplete(t *testing.T) {
	repo := &conversationLogRepositoryStub{}
	conversationService := service.NewConversationLogService(repo)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodGet, "/responses", nil)
	apiKey := &service.APIKey{ID: 7, UserID: 11}
	recorder := newOpenAIWSConversationRecorder(conversationService, c, apiKey)
	recorder.beginTurn(1, []byte(`{"type":"response.create","model":"gpt-test"}`), "gpt-test", time.Now())
	recorder.appendResponseFrame(1, []byte(`{"type":"response.output_text.delta","delta":"partial"}`))

	recorder.closePending()

	item := repo.finalized[1]
	require.NotNil(t, item)
	require.Equal(t, "incomplete", item.Status)
	require.Contains(t, item.ResponseBody, "partial")
}

func TestOpenAIWSConversationRecorderMapsRetryTurnToActiveRequest(t *testing.T) {
	repo := &conversationLogRepositoryStub{}
	conversationService := service.NewConversationLogService(repo)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodGet, "/responses", nil)
	recorder := newOpenAIWSConversationRecorder(conversationService, c, &service.APIKey{ID: 7, UserID: 11})
	recorder.beginTurn(2, []byte(`{"type":"response.create","model":"gpt-test","input":"retry me"}`), "gpt-test", time.Now())

	// A new upstream attempt restarts its internal turn counter at one.
	recorder.finishTurn(1, &service.OpenAIForwardResult{
		Model: "gpt-test", OpenAIWSMode: true, UpstreamTerminalEvent: "response.completed",
	}, nil)
	recorder.appendResponseFrame(1, []byte(`{"type":"response.completed","response":{"id":"resp_retry"}}`))

	require.Len(t, repo.created, 1)
	item := repo.finalized[1]
	require.NotNil(t, item)
	require.Equal(t, "completed", item.Status)
	require.Contains(t, item.RequestBody, "retry me")
	require.Contains(t, item.ResponseBody, "resp_retry")
}
