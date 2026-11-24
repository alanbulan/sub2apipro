package handler

import (
	"context"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/tidwall/gjson"
	"go.uber.org/zap"
)

type openAIWSConversationTurn struct {
	id            int64
	item          *service.ConversationLog
	response      boundedConversationCapture
	startedAt     time.Time
	result        *service.OpenAIForwardResult
	turnErr       error
	finished      bool
	terminalSeen  bool
	terminalEvent string
}

type openAIWSConversationRecorder struct {
	service    *service.ConversationLogService
	userID     int64
	userEmail  string
	apiKeyID   int64
	apiKeyName string
	groupID    *int64
	sessionID  string
	endpoint   string
	mu         sync.Mutex
	turns      map[int]*openAIWSConversationTurn
}

func newOpenAIWSConversationRecorder(
	conversationLogService *service.ConversationLogService,
	c *gin.Context,
	apiKey *service.APIKey,
) *openAIWSConversationRecorder {
	if conversationLogService == nil || c == nil || c.Request == nil || apiKey == nil {
		return nil
	}
	sessionID := service.ExtractClientSessionID(c)
	if sessionID == "" {
		sessionID, _ = c.Request.Context().Value(ctxkey.ClientRequestID).(string)
		sessionID = strings.TrimSpace(sessionID)
	}
	if sessionID == "" {
		sessionID = uuid.NewString()
	}
	userEmail := ""
	if apiKey.User != nil {
		userEmail = apiKey.User.Email
	}
	return &openAIWSConversationRecorder{
		service: conversationLogService, userID: apiKey.UserID, userEmail: userEmail,
		apiKeyID: apiKey.ID, apiKeyName: apiKey.Name, groupID: apiKey.GroupID,
		sessionID: sessionID, endpoint: c.Request.URL.Path,
		turns: make(map[int]*openAIWSConversationTurn),
	}
}

func (r *openAIWSConversationRecorder) beginTurn(turn int, payload []byte, model string, startedAt time.Time) {
	if r == nil || turn < 1 {
		return
	}
	if startedAt.IsZero() {
		startedAt = time.Now()
	}
	requestBody, requestBytes, requestTruncated := boundedConversationBody(payload)
	model = strings.TrimSpace(model)
	if model == "" {
		model = strings.TrimSpace(gjson.GetBytes(payload, "model").String())
	}
	item := &service.ConversationLog{
		RequestID: uuid.NewString(), SessionID: r.sessionID, UserID: r.userID, UserEmail: r.userEmail,
		APIKeyID: r.apiKeyID, APIKeyName: r.apiKeyName, GroupID: r.groupID,
		Protocol: "openai_responses_ws", Endpoint: r.endpoint, Model: model, Stream: true,
		Status: "pending", RequestBody: requestBody, RequestBytes: requestBytes,
		RequestTruncated: requestTruncated,
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.turns) > 0 {
		return
	}
	persistCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	id, err := r.service.Begin(persistCtx, item)
	cancel()
	if err != nil {
		logger.L().Warn("conversation_log.ws_begin_failed", zap.Int("turn", turn), zap.Error(err))
		return
	}
	r.turns[turn] = &openAIWSConversationTurn{id: id, item: item, startedAt: startedAt}
}

func (r *openAIWSConversationRecorder) turnStateLocked(turn int) (int, *openAIWSConversationTurn) {
	if state := r.turns[turn]; state != nil {
		return turn, state
	}
	if len(r.turns) == 1 {
		for activeTurn, state := range r.turns {
			return activeTurn, state
		}
	}
	return turn, nil
}

func (r *openAIWSConversationRecorder) setTurnStarted(turn int, startedAt time.Time) {
	if r == nil || startedAt.IsZero() {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, state := r.turnStateLocked(turn); state != nil {
		state.startedAt = startedAt
	}
}

func isOpenAIWSConversationTerminalEvent(eventType string) bool {
	switch strings.TrimSpace(eventType) {
	case "error", "response.canceled", "response.cancelled", "response.completed", "response.done", "response.failed", "response.incomplete":
		return true
	default:
		return false
	}
}

func (r *openAIWSConversationRecorder) appendResponseFrame(turn int, payload []byte) {
	if r == nil || turn < 1 {
		return
	}
	r.mu.Lock()
	activeTurn, state := r.turnStateLocked(turn)
	r.mu.Unlock()
	if state == nil {
		return
	}
	frame := make([]byte, len(payload)+1)
	copy(frame, payload)
	frame[len(payload)] = '\n'
	state.response.write(frame)

	eventType := strings.TrimSpace(gjson.GetBytes(payload, "type").String())
	if !isOpenAIWSConversationTerminalEvent(eventType) {
		return
	}
	r.mu.Lock()
	if current := r.turns[activeTurn]; current == state {
		state.terminalSeen = true
		state.terminalEvent = eventType
	}
	ready := state.finished
	r.mu.Unlock()
	if ready {
		r.finalizeTurn(activeTurn, false)
	}
}

func (r *openAIWSConversationRecorder) finishTurn(turn int, result *service.OpenAIForwardResult, turnErr error) {
	if r == nil || turn < 1 {
		return
	}
	r.mu.Lock()
	activeTurn, state := r.turnStateLocked(turn)
	if state == nil {
		r.mu.Unlock()
		return
	}
	state.result = result
	state.turnErr = turnErr
	state.finished = true
	ready := state.terminalSeen
	r.mu.Unlock()
	if ready {
		r.finalizeTurn(activeTurn, false)
	}
}

func (r *openAIWSConversationRecorder) finalizeTurn(turn int, forceIncomplete bool) {
	if r == nil {
		return
	}
	r.mu.Lock()
	activeTurn, state := r.turnStateLocked(turn)
	if state == nil || (!forceIncomplete && (!state.finished || !state.terminalSeen)) {
		r.mu.Unlock()
		return
	}
	delete(r.turns, activeTurn)
	r.mu.Unlock()

	status := "completed"
	switch state.terminalEvent {
	case "error", "response.failed":
		status = "failed"
	case "response.canceled", "response.cancelled", "response.incomplete":
		status = "incomplete"
	}
	if forceIncomplete || state.turnErr != nil && state.terminalEvent == "" {
		status = "incomplete"
	}
	if state.result != nil {
		if model := strings.TrimSpace(state.result.Model); model != "" {
			state.item.Model = model
		}
		state.item.DurationMs = state.result.Duration.Milliseconds()
	}
	if state.item.DurationMs <= 0 {
		state.item.DurationMs = time.Since(state.startedAt).Milliseconds()
	}
	responseBody, responseBytes, responseTruncated := state.response.snapshot()
	state.item.Status = status
	state.item.StatusCode = http.StatusSwitchingProtocols
	state.item.ContentType = "application/x-ndjson"
	state.item.ResponseBody = conversationBodyText(responseBody)
	state.item.ResponseBytes = responseBytes
	state.item.ResponseTruncated = responseTruncated
	finalizeCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	err := r.service.Finalize(finalizeCtx, state.id, state.item)
	cancel()
	if err != nil {
		logger.L().Warn("conversation_log.ws_finalize_failed", zap.Int64("id", state.id), zap.Int("turn", activeTurn), zap.Error(err))
	}
}

func (r *openAIWSConversationRecorder) closePending() {
	if r == nil {
		return
	}
	r.mu.Lock()
	turns := make([]int, 0, len(r.turns))
	for turn := range r.turns {
		turns = append(turns, turn)
	}
	r.mu.Unlock()
	sort.Ints(turns)
	for _, turn := range turns {
		r.finalizeTurn(turn, true)
	}
}
