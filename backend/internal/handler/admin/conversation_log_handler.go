package admin

import (
	"database/sql"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type ConversationLogHandler struct {
	service *service.ConversationLogService
}

var conversationLogProtocols = map[string]struct{}{
	"anthropic_messages":      {},
	"gemini_generate_content": {},
	"openai_chat_completions": {},
	"openai_responses":        {},
	"openai_responses_ws":     {},
}

var conversationLogStatuses = map[string]struct{}{
	"completed":  {},
	"failed":     {},
	"incomplete": {},
	"pending":    {},
}

func NewConversationLogHandler(service *service.ConversationLogService) *ConversationLogHandler {
	return &ConversationLogHandler{service: service}
}

func parseConversationInt(c *gin.Context, name string) (int64, bool) {
	raw := strings.TrimSpace(c.Query(name))
	if raw == "" {
		return 0, true
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || value < 1 {
		response.BadRequest(c, "Invalid "+name)
		return 0, false
	}
	return value, true
}

func parseConversationTime(c *gin.Context, name string) (*time.Time, bool) {
	raw := strings.TrimSpace(c.Query(name))
	if raw == "" {
		return nil, true
	}
	value, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		response.BadRequest(c, "Invalid "+name+", use RFC3339")
		return nil, false
	}
	return &value, true
}

func parseConversationEnum(c *gin.Context, name string, allowed map[string]struct{}) (string, bool) {
	value := strings.TrimSpace(c.Query(name))
	if value == "" {
		return "", true
	}
	if _, ok := allowed[value]; !ok {
		response.BadRequest(c, "Invalid "+name)
		return "", false
	}
	return value, true
}

func (h *ConversationLogHandler) List(c *gin.Context) {
	page, pageSize := response.ParsePagination(c)
	if pageSize > 200 {
		pageSize = 200
	}
	userID, ok := parseConversationInt(c, "user_id")
	if !ok {
		return
	}
	apiKeyID, ok := parseConversationInt(c, "api_key_id")
	if !ok {
		return
	}
	startTime, ok := parseConversationTime(c, "start_time")
	if !ok {
		return
	}
	endTime, ok := parseConversationTime(c, "end_time")
	if !ok {
		return
	}
	if startTime != nil && endTime != nil && startTime.After(*endTime) {
		response.BadRequest(c, "start_time must not be after end_time")
		return
	}
	protocol, ok := parseConversationEnum(c, "protocol", conversationLogProtocols)
	if !ok {
		return
	}
	status, ok := parseConversationEnum(c, "status", conversationLogStatuses)
	if !ok {
		return
	}
	filter := service.ConversationLogFilter{
		UserID: userID, APIKeyID: apiKeyID, SessionID: strings.TrimSpace(c.Query("session_id")),
		RequestID: strings.TrimSpace(c.Query("request_id")), Model: strings.TrimSpace(c.Query("model")),
		Protocol: protocol, Status: status,
		StartTime: startTime, EndTime: endTime,
		Pagination: pagination.PaginationParams{Page: page, PageSize: pageSize},
	}
	items, total, err := h.service.List(c.Request.Context(), filter)
	if err != nil {
		response.InternalError(c, "Failed to load conversation logs")
		return
	}
	response.Paginated(c, items, total, page, pageSize)
}

func (h *ConversationLogHandler) Get(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id < 1 {
		response.BadRequest(c, "Invalid conversation log id")
		return
	}
	item, err := h.service.GetByID(c.Request.Context(), id)
	if errors.Is(err, sql.ErrNoRows) {
		response.NotFound(c, "Conversation log not found")
		return
	}
	if err != nil {
		response.InternalError(c, "Failed to load conversation log")
		return
	}
	response.Success(c, item)
}
