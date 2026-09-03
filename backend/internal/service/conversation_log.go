package service

import (
	"context"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
)

type ConversationLog struct {
	ID                int64      `json:"id"`
	RequestID         string     `json:"request_id"`
	SessionID         string     `json:"session_id"`
	UserID            int64      `json:"user_id"`
	UserEmail         string     `json:"user_email"`
	APIKeyID          int64      `json:"api_key_id"`
	APIKeyName        string     `json:"api_key_name"`
	GroupID           *int64     `json:"group_id,omitempty"`
	Protocol          string     `json:"protocol"`
	Endpoint          string     `json:"endpoint"`
	Model             string     `json:"model"`
	Stream            bool       `json:"stream"`
	Status            string     `json:"status"`
	StatusCode        int        `json:"status_code"`
	ContentType       string     `json:"content_type"`
	RequestBody       string     `json:"request_body,omitempty"`
	ResponseBody      string     `json:"response_body,omitempty"`
	RequestBytes      int64      `json:"request_bytes"`
	ResponseBytes     int64      `json:"response_bytes"`
	RequestTruncated  bool       `json:"request_truncated"`
	ResponseTruncated bool       `json:"response_truncated"`
	DurationMs        int64      `json:"duration_ms"`
	CreatedAt         time.Time  `json:"created_at"`
	CompletedAt       *time.Time `json:"completed_at,omitempty"`
}

type ConversationLogFilter struct {
	UserID     int64
	APIKeyID   int64
	SessionID  string
	RequestID  string
	Model      string
	Protocol   string
	Status     string
	StartTime  *time.Time
	EndTime    *time.Time
	Pagination pagination.PaginationParams
}

type ConversationLogRepository interface {
	Create(context.Context, *ConversationLog) (int64, error)
	Finalize(context.Context, int64, *ConversationLog) error
	List(context.Context, ConversationLogFilter) ([]*ConversationLog, int64, error)
	GetByID(context.Context, int64) (*ConversationLog, error)
}

type ConversationLogService struct{ repo ConversationLogRepository }

func NewConversationLogService(repo ConversationLogRepository) *ConversationLogService {
	return &ConversationLogService{repo: repo}
}

func (s *ConversationLogService) Begin(ctx context.Context, log *ConversationLog) (int64, error) {
	return s.repo.Create(ctx, log)
}

func (s *ConversationLogService) Finalize(ctx context.Context, id int64, log *ConversationLog) error {
	return s.repo.Finalize(ctx, id, log)
}

func (s *ConversationLogService) List(ctx context.Context, filter ConversationLogFilter) ([]*ConversationLog, int64, error) {
	return s.repo.List(ctx, filter)
}

func (s *ConversationLogService) GetByID(ctx context.Context, id int64) (*ConversationLog, error) {
	return s.repo.GetByID(ctx, id)
}
