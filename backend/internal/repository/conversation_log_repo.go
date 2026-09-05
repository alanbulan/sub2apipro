package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

type ConversationLogRepository struct{ db *sql.DB }

func NewConversationLogRepository(db *sql.DB) service.ConversationLogRepository {
	return &ConversationLogRepository{db: db}
}

func (r *ConversationLogRepository) Create(ctx context.Context, item *service.ConversationLog) (int64, error) {
	var id int64
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO conversation_logs
		(request_id, session_id, user_id, user_email, api_key_id, api_key_name, group_id,
		 protocol, endpoint, model, stream, request_body, request_bytes, request_truncated)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
		RETURNING id`,
		item.RequestID, item.SessionID, item.UserID, item.UserEmail, item.APIKeyID, item.APIKeyName,
		item.GroupID, item.Protocol, item.Endpoint, item.Model, item.Stream, item.RequestBody,
		item.RequestBytes, item.RequestTruncated,
	).Scan(&id)
	return id, err
}

func (r *ConversationLogRepository) Finalize(ctx context.Context, id int64, item *service.ConversationLog) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE conversation_logs SET model=$2, stream=$3, status=$4, status_code=$5,
		content_type=$6, request_body=$7, request_bytes=$8, request_truncated=$9,
		response_body=$10, response_bytes=$11, response_truncated=$12, duration_ms=$13,
		completed_at=NOW() WHERE id=$1`, id, item.Model, item.Stream, item.Status,
		item.StatusCode, item.ContentType, item.RequestBody, item.RequestBytes,
		item.RequestTruncated, item.ResponseBody, item.ResponseBytes,
		item.ResponseTruncated, item.DurationMs)
	return err
}

func conversationLogWhere(filter service.ConversationLogFilter) (string, []any) {
	clauses := []string{"1=1"}
	args := make([]any, 0, 9)
	add := func(expr string, value any) {
		args = append(args, value)
		clauses = append(clauses, fmt.Sprintf(expr, len(args)))
	}
	if filter.UserID > 0 {
		add("user_id=$%d", filter.UserID)
	}
	if filter.APIKeyID > 0 {
		add("api_key_id=$%d", filter.APIKeyID)
	}
	if filter.SessionID != "" {
		add("session_id ILIKE '%%' || $%d || '%%'", filter.SessionID)
	}
	if filter.RequestID != "" {
		add("request_id=$%d", filter.RequestID)
	}
	if filter.Model != "" {
		add("model ILIKE '%%' || $%d || '%%'", filter.Model)
	}
	if filter.Protocol != "" {
		add("protocol=$%d", filter.Protocol)
	}
	if filter.Status != "" {
		add("status=$%d", filter.Status)
	}
	if filter.StartTime != nil {
		add("created_at >= $%d", *filter.StartTime)
	}
	if filter.EndTime != nil {
		add("created_at <= $%d", *filter.EndTime)
	}
	return strings.Join(clauses, " AND "), args
}

func scanConversationSummary(scanner interface{ Scan(...any) error }) (*service.ConversationLog, error) {
	item := &service.ConversationLog{}
	err := scanner.Scan(&item.ID, &item.RequestID, &item.SessionID, &item.UserID, &item.UserEmail,
		&item.APIKeyID, &item.APIKeyName, &item.GroupID, &item.Protocol, &item.Endpoint, &item.Model,
		&item.Stream, &item.Status, &item.StatusCode, &item.ContentType, &item.RequestBytes,
		&item.ResponseBytes, &item.RequestTruncated, &item.ResponseTruncated, &item.DurationMs,
		&item.CreatedAt, &item.CompletedAt)
	return item, err
}

const conversationSummaryColumns = `id,request_id,session_id,user_id,user_email,api_key_id,api_key_name,
	group_id,protocol,endpoint,model,stream,status,status_code,content_type,request_bytes,response_bytes,
	request_truncated,response_truncated,duration_ms,created_at,completed_at`

func (r *ConversationLogRepository) List(ctx context.Context, filter service.ConversationLogFilter) ([]*service.ConversationLog, int64, error) {
	where, args := conversationLogWhere(filter)
	var total int64
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM conversation_logs WHERE "+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	limit, offset := filter.Pagination.Limit(), filter.Pagination.Offset()
	queryArgs := append(append([]any{}, args...), limit, offset)
	rows, err := r.db.QueryContext(ctx, "SELECT "+conversationSummaryColumns+" FROM conversation_logs WHERE "+where+
		fmt.Sprintf(" ORDER BY created_at DESC,id DESC LIMIT $%d OFFSET $%d", len(args)+1, len(args)+2), queryArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = rows.Close() }()
	items := make([]*service.ConversationLog, 0, limit)
	for rows.Next() {
		item, scanErr := scanConversationSummary(rows)
		if scanErr != nil {
			return nil, 0, scanErr
		}
		items = append(items, item)
	}
	return items, total, rows.Err()
}

func (r *ConversationLogRepository) GetByID(ctx context.Context, id int64) (*service.ConversationLog, error) {
	item, err := scanConversationSummary(r.db.QueryRowContext(ctx, "SELECT "+conversationSummaryColumns+" FROM conversation_logs WHERE id=$1", id))
	if err != nil {
		return nil, err
	}
	if err := r.db.QueryRowContext(ctx, "SELECT request_body,response_body FROM conversation_logs WHERE id=$1", id).Scan(&item.RequestBody, &item.ResponseBody); err != nil {
		return nil, err
	}
	return item, nil
}
