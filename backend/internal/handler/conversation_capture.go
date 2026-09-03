package handler

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	pkghttputil "github.com/Wei-Shaw/sub2api/internal/pkg/httputil"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	servermiddleware "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
	"go.uber.org/zap"
)

const conversationCaptureBodyLimit = 4 << 20

type boundedConversationCapture struct {
	mu        sync.Mutex
	buf       bytes.Buffer
	total     int64
	truncated bool
}

func (b *boundedConversationCapture) write(p []byte) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.total += int64(len(p))
	remaining := conversationCaptureBodyLimit - b.buf.Len()
	if remaining <= 0 {
		b.truncated = true
		return
	}
	if len(p) > remaining {
		p = p[:remaining]
		b.truncated = true
	}
	_, _ = b.buf.Write(p)
}

func (b *boundedConversationCapture) snapshot() ([]byte, int64, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	body := append([]byte(nil), b.buf.Bytes()...)
	return body, b.total, b.truncated
}

type conversationCaptureReadCloser struct {
	io.ReadCloser
	capture boundedConversationCapture
}

func (r *conversationCaptureReadCloser) Read(p []byte) (int, error) {
	n, err := r.ReadCloser.Read(p)
	if n > 0 {
		r.capture.write(p[:n])
	}
	return n, err
}

type conversationCaptureWriter struct {
	gin.ResponseWriter
	capture boundedConversationCapture
}

func (w *conversationCaptureWriter) Write(p []byte) (int, error) {
	w.capture.write(p)
	return w.ResponseWriter.Write(p)
}

func (w *conversationCaptureWriter) WriteString(s string) (int, error) {
	w.capture.write([]byte(s))
	return w.ResponseWriter.WriteString(s)
}

func boundedConversationBody(body []byte) (string, int64, bool) {
	if len(body) <= conversationCaptureBodyLimit {
		return conversationBodyText(body), int64(len(body)), false
	}
	return conversationBodyText(body[:conversationCaptureBodyLimit]), int64(len(body)), true
}

func conversationBodyText(body []byte) string {
	return strings.ToValidUTF8(string(body), "?")
}

func shouldCaptureConversation(c *gin.Context, protocol string) bool {
	switch protocol {
	case "openai_responses":
		return !service.IsOpenAIResponsesInputTokensRequestPath(c)
	case "gemini_generate_content":
		_, action, err := parseGeminiModelAction(strings.TrimPrefix(c.Param("modelAction"), "/"))
		return err == nil && (action == "generateContent" || action == "streamGenerateContent")
	default:
		return true
	}
}

func capturedConversationRequestBody(capture *boundedConversationCapture, contentEncoding string) (string, int64, bool) {
	body, total, truncated := capture.snapshot()
	if truncated || strings.TrimSpace(contentEncoding) == "" || strings.EqualFold(strings.TrimSpace(contentEncoding), "identity") {
		return conversationBodyText(body), total, truncated
	}

	req := &http.Request{
		Body:          io.NopCloser(bytes.NewReader(body)),
		ContentLength: total,
		Header:        make(http.Header),
	}
	req.Header.Set("Content-Encoding", contentEncoding)
	decoded, err := pkghttputil.ReadRequestBodyWithPrealloc(req)
	if err != nil {
		return conversationBodyText(body), total, truncated
	}
	return boundedConversationBody(decoded)
}

func populateConversationRequestMetadata(c *gin.Context, item *service.ConversationLog) {
	body := []byte(item.RequestBody)
	item.Model = strings.TrimSpace(gjson.GetBytes(body, "model").String())
	if item.Model == "" && item.Protocol == "gemini_generate_content" {
		model, _, err := parseGeminiModelAction(strings.TrimPrefix(c.Param("modelAction"), "/"))
		if err == nil {
			item.Model = strings.TrimSpace(model)
		}
	}
	if item.Model == "" {
		item.Model = strings.TrimSpace(c.Param("model"))
	}
	item.Stream = gjson.GetBytes(body, "stream").Bool() ||
		strings.Contains(strings.ToLower(c.Param("modelAction")), "streamgeneratecontent")
}

// ConversationCapture persists one row for a client-visible conversational HTTP exchange.
// It must run after API-key authentication and before the endpoint handler.
func (h *GatewayHandler) ConversationCapture(protocol string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if h == nil || h.conversationLogService == nil || c.Request == nil || c.Request.Body == nil ||
			!shouldCaptureConversation(c, protocol) {
			c.Next()
			return
		}
		apiKey, ok := servermiddleware.GetAPIKeyFromContext(c)
		if !ok || apiKey == nil {
			c.Next()
			return
		}
		requestID, _ := c.Request.Context().Value(ctxkey.ClientRequestID).(string)
		requestID = strings.TrimSpace(requestID)
		if requestID == "" {
			requestID = c.GetHeader("X-Request-ID")
		}
		if requestID == "" {
			c.Next()
			return
		}
		sessionID := service.ExtractClientSessionID(c)
		if sessionID == "" {
			sessionID = requestID
		}
		userEmail := ""
		if apiKey.User != nil {
			userEmail = apiKey.User.Email
		}
		item := &service.ConversationLog{
			RequestID: requestID, SessionID: sessionID, UserID: apiKey.UserID, UserEmail: userEmail,
			APIKeyID: apiKey.ID, APIKeyName: apiKey.Name, GroupID: apiKey.GroupID,
			Protocol: protocol, Endpoint: c.Request.URL.Path, Status: "pending",
		}
		persistCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		id, err := h.conversationLogService.Begin(persistCtx, item)
		cancel()
		if err != nil {
			logger.L().Warn("conversation_log.begin_failed", zap.String("request_id", requestID), zap.Error(err))
			c.Next()
			return
		}

		started := time.Now()
		contentEncoding := c.GetHeader("Content-Encoding")
		requestReader := &conversationCaptureReadCloser{ReadCloser: c.Request.Body}
		c.Request.Body = requestReader
		originalWriter := c.Writer
		writer := &conversationCaptureWriter{ResponseWriter: c.Writer}
		c.Writer = writer
		defer func() {
			panicValue := recover()
			item.RequestBody, item.RequestBytes, item.RequestTruncated = capturedConversationRequestBody(&requestReader.capture, contentEncoding)
			populateConversationRequestMetadata(c, item)
			statusCode := writer.Status()
			if statusCode == 0 {
				statusCode = http.StatusOK
			}
			if panicValue != nil {
				statusCode = http.StatusInternalServerError
			}
			status := "completed"
			if statusCode >= 400 {
				status = "failed"
			}
			if panicValue != nil || c.Request.Context().Err() != nil {
				status = "incomplete"
			}
			item.Status, item.StatusCode = status, statusCode
			item.ContentType = writer.Header().Get("Content-Type")
			responseBody, responseBytes, responseTruncated := writer.capture.snapshot()
			item.ResponseBody, item.ResponseBytes, item.ResponseTruncated = conversationBodyText(responseBody), responseBytes, responseTruncated
			item.DurationMs = time.Since(started).Milliseconds()
			finalizeCtx, finalizeCancel := context.WithTimeout(context.Background(), 5*time.Second)
			finalizeErr := h.conversationLogService.Finalize(finalizeCtx, id, item)
			finalizeCancel()
			if finalizeErr != nil {
				logger.L().Warn("conversation_log.finalize_failed", zap.Int64("id", id), zap.Error(finalizeErr))
			}
			c.Writer = originalWriter
			if panicValue != nil {
				panic(panicValue)
			}
		}()
		c.Next()
	}
}
