package admin

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type poolDiagnosticTarget struct {
	userID, groupID int64
	account         *service.Account
	denied          []string
}

// Recheck live key/group permissions AND account membership for every operation.
// A quota-cache entry or a previously opened modal never grants access.
func (h *AccountHandler) authorizePoolDiagnostic(c *gin.Context) *poolDiagnosticTarget {
	c.Header("Cache-Control", "private, no-store")
	subject, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return nil
	}
	groupID, groupErr := strconv.ParseInt(c.Param("group_id"), 10, 64)
	accountID, accountErr := strconv.ParseInt(c.Param("id"), 10, 64)
	if groupErr != nil || accountErr != nil || groupID <= 0 || accountID <= 0 {
		response.BadRequest(c, "Invalid account or group")
		return nil
	}
	if h.poolGroupReader == nil {
		response.ErrorFrom(c, service.ErrServiceUnavailable)
		return nil
	}
	ctx := c.Request.Context()
	groups, err := h.poolGroupReader.PoolSourceGroups(ctx, subject.UserID)
	if err != nil {
		response.Error(c, http.StatusServiceUnavailable, "Account permissions are unavailable")
		return nil
	}
	permitted := false
	for _, group := range groups {
		if group.ID == groupID {
			permitted = true
			break
		}
	}
	if !permitted {
		response.NotFound(c, "Account not found")
		return nil
	}
	accounts, err := h.adminService.ListAccountsForSchedulerScoreFilter(ctx, "", "", "", "", groupID, "")
	if err != nil {
		response.Error(c, http.StatusServiceUnavailable, "Accounts are unavailable")
		return nil
	}
	permitted = false
	for _, account := range accounts {
		if account.ID == accountID {
			permitted = true
			break
		}
	}
	if !permitted {
		response.NotFound(c, "Account not found")
		return nil
	}
	account, err := h.adminService.GetAccount(ctx, accountID)
	if err != nil || account == nil {
		response.NotFound(c, "Account not found")
		return nil
	}
	denied, err := h.poolGroupReader.GetUserGroupDeniedModels(ctx, subject.UserID)
	if err != nil {
		response.Error(c, http.StatusServiceUnavailable, "Model permissions are unavailable")
		return nil
	}
	return &poolDiagnosticTarget{userID: subject.UserID, groupID: groupID, account: account, denied: denied[groupID]}
}

type poolDiagnosticModel struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
	Type        string `json:"type"`
}

func (target *poolDiagnosticTarget) allowsModel(id string) bool {
	if id == "" || strings.ContainsAny(id, "*\r\n") || !target.account.IsModelSupportedInGroup(&target.groupID, id) {
		return false
	}
	mapped := target.account.GetMappedModel(id)
	if service.FirstUserGroupDeniedModel(target.denied, []string{id, mapped}) != "" {
		return false
	}
	// User diagnostics use the shared TEXT tests only. Media/upload/compact and
	// administrative probes remain on the administrator routes.
	for _, model := range []string{id, mapped} {
		lower := strings.ToLower(model)
		for _, marker := range []string{"image", "imagine", "video", "audio", "tts", "realtime", "whisper", "embedding", "moderation"} {
			if strings.Contains(lower, marker) {
				return false
			}
		}
	}
	return true
}

// Reuse the existing catalog handler so upstream platform/model-discovery fixes
// apply here too. Its response is captured internally, then decoded into an
// independent allowlist DTO; neither raw admin errors nor extra fields escape.
func (h *AccountHandler) poolDiagnosticModels(c *gin.Context, target *poolDiagnosticTarget) ([]poolDiagnosticModel, bool) {
	recorder := httptest.NewRecorder()
	probe, _ := gin.CreateTestContext(recorder)
	probe.Request = c.Request.Clone(c.Request.Context())
	probe.Params = gin.Params{{Key: "id", Value: strconv.FormatInt(target.account.ID, 10)}}
	h.GetAvailableModels(probe)
	var envelope struct {
		Data []poolDiagnosticModel `json:"data"`
	}
	if recorder.Code != http.StatusOK || json.Unmarshal(recorder.Body.Bytes(), &envelope) != nil {
		response.Error(c, http.StatusServiceUnavailable, "Models are unavailable; please retry")
		return nil, false
	}
	models := make([]poolDiagnosticModel, 0, len(envelope.Data))
	seen := map[string]bool{}
	for _, model := range envelope.Data {
		if !seen[model.ID] && target.allowsModel(model.ID) {
			// Do not pass through upstream-provided descriptive text/metadata.
			models = append(models, poolDiagnosticModel{ID: model.ID, DisplayName: model.ID, Type: "model"})
			seen[model.ID] = true
		}
	}
	return models, true
}

func (h *AccountHandler) PoolDiagnosticModels(c *gin.Context) {
	if target := h.authorizePoolDiagnostic(c); target != nil {
		if models, ok := h.poolDiagnosticModels(c, target); ok {
			response.Success(c, models)
		}
	}
}

func (h *AccountHandler) validPoolModel(c *gin.Context, target *poolDiagnosticTarget, modelID string) bool {
	models, ok := h.poolDiagnosticModels(c, target)
	if !ok {
		return false
	}
	for _, model := range models {
		if model.ID == modelID {
			return true
		}
	}
	response.BadRequest(c, "Please select an available model")
	return false
}

func (h *AccountHandler) PoolDiagnosticReasoning(c *gin.Context) {
	target := h.authorizePoolDiagnostic(c)
	if target == nil || !h.validPoolModel(c, target, c.Query("model_id")) {
		return
	}
	if h.accountTestService == nil {
		response.ErrorFrom(c, service.ErrServiceUnavailable)
		return
	}
	response.Success(c, h.accountTestService.GetAccountTestReasoning(c.Request.Context(), target.account, c.Query("model_id")))
}

type poolDiagnosticRequest struct {
	ModelID         string `json:"model_id"`
	QuestionKind    string `json:"question_kind"`
	ReasoningEffort string `json:"reasoning_effort"`
}

type poolDiagnosticUserLimit struct {
	window time.Time
	count  int
	active bool
}
type poolDiagnosticLimits struct {
	users    map[int64]poolDiagnosticUserLimit
	accounts map[int64]time.Time
	active   int
}

// Bound user-triggered inference separately from dashboard refreshes. Account
// entries stay reserved while running, even when a slow test exceeds cooldown.
func (h *AccountHandler) beginPoolTest(userID, accountID int64, now time.Time) (func(), bool) {
	h.poolDashboard.mu.Lock()
	defer h.poolDashboard.mu.Unlock()
	limits := &h.poolDashboard.tests
	if limits.users == nil {
		limits.users = map[int64]poolDiagnosticUserLimit{}
		limits.accounts = map[int64]time.Time{}
	}
	for id, user := range limits.users {
		if !user.active && now.Sub(user.window) >= time.Minute {
			delete(limits.users, id)
		}
	}
	for id, until := range limits.accounts {
		if !until.IsZero() && !now.Before(until) {
			delete(limits.accounts, id)
		}
	}
	user := limits.users[userID]
	if user.active || user.count >= 4 || limits.active >= 4 {
		return nil, false
	}
	if _, busy := limits.accounts[accountID]; busy {
		return nil, false
	}
	if user.window.IsZero() {
		user.window = now
	}
	user.count++
	user.active = true
	limits.users[userID] = user
	limits.accounts[accountID] = time.Time{}
	limits.active++
	return func() {
		h.poolDashboard.mu.Lock()
		defer h.poolDashboard.mu.Unlock()
		user := limits.users[userID]
		user.active = false
		limits.users[userID] = user
		limits.accounts[accountID] = time.Now().Add(10 * time.Second)
		limits.active--
	}, true
}

func (h *AccountHandler) PoolConnectionTest(c *gin.Context)   { h.runPoolTest(c, false) }
func (h *AccountHandler) PoolIntelligenceTest(c *gin.Context) { h.runPoolTest(c, true) }

func (h *AccountHandler) runPoolTest(c *gin.Context, intelligence bool) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Minute)
	defer cancel()
	c.Request = c.Request.WithContext(ctx)
	target := h.authorizePoolDiagnostic(c)
	if target == nil {
		return
	}
	var req poolDiagnosticRequest
	decoder := json.NewDecoder(http.MaxBytesReader(c.Writer, c.Request.Body, 4096))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		response.BadRequest(c, "Invalid test request")
		return
	}
	var trailing any
	if decoder.Decode(&trailing) != io.EOF {
		response.BadRequest(c, "Invalid test request")
		return
	}
	if (!intelligence && (req.QuestionKind != "" || req.ReasoningEffort != "")) || (intelligence && req.QuestionKind != "candy" && req.QuestionKind != "pelican") {
		response.BadRequest(c, "Please select a test question")
		return
	}
	if h.accountTestService == nil {
		response.ErrorFrom(c, service.ErrServiceUnavailable)
		return
	}
	if !h.validPoolModel(c, target, req.ModelID) {
		return
	}
	if intelligence && req.ReasoningEffort != "" {
		levels := h.accountTestService.GetAccountTestReasoning(ctx, target.account, req.ModelID).SupportedReasoningLevels
		valid := false
		for _, level := range levels {
			if level == req.ReasoningEffort {
				valid = true
			}
		}
		if !valid {
			response.BadRequest(c, "Please select an available reasoning level")
			return
		}
	}
	release, ok := h.beginPoolTest(target.userID, target.account.ID, time.Now())
	if !ok {
		c.Header("Retry-After", "10")
		response.Error(c, http.StatusTooManyRequests, "测试请求较多，请稍后再试")
		return
	}
	defer release()
	// No administrative recovery/reset is invoked by this user endpoint.
	_ = h.accountTestService.TestPoolAccountConnection(c, target.account, req.ModelID, req.QuestionKind, req.ReasoningEffort)
}
