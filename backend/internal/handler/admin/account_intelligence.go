package admin

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

// IQ requests use the canonical account transport; no gateway policy is changed.
func (h *AccountHandler) TestPelicanAccount(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "Invalid account ID")
		return
	}
	var req struct {
		ModelID         string `json:"model_id"`
		Prompt          string `json:"prompt"`
		ReasoningEffort string `json:"reasoning_effort"`
	}
	if c.ShouldBindJSON(&req) != nil || len(req.Prompt) > 32000 || strings.TrimSpace(req.ModelID) == "" || len(req.ModelID) > 100 {
		response.BadRequest(c, "Invalid intelligence input")
		return
	}
	cfg := service.PelicanTestConfig{Prompt: req.Prompt, ReasoningEffort: req.ReasoningEffort, ParallelCount: 1}
	if err = service.ValidateIntelligenceConfig(&cfg); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Minute)
	defer cancel()
	c.Request = c.Request.WithContext(ctx)
	_ = h.accountTestService.TestPelicanAccountConnection(c, id, req.ModelID, req.Prompt, req.ReasoningEffort)
}
func (h *AccountHandler) ProbeIntelligenceState(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "Invalid account ID")
		return
	}
	var req struct {
		ModelID string `json:"model_id"`
	}
	if c.ShouldBindJSON(&req) != nil || len(req.ModelID) > 100 {
		response.BadRequest(c, "Invalid model")
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 100*time.Second)
	defer cancel()
	result, err := h.accountTestService.ProbeOpenAICodexState(ctx, id, req.ModelID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}
