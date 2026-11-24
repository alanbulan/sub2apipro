package admin

import (
	"context"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

func (h *SettingHandler) SetFeishuKeyDelivery(keys *service.APIKeyService, caches ...*service.BillingCacheService) {
	h.feishuKeyDelivery = service.NewFeishuKeyDeliveryService(h.feishuDirectory, keys, h.emailService, caches...)
	// API/control roles own the database; gateway replicas must not send mail.
	if h.settingService.RunsFeishuBackgroundJobs() {
		h.feishuKeyDelivery.StartWorker()
	}
}
func (h *SettingHandler) StopFeishuKeyDelivery() {
	if h != nil && h.feishuKeyDelivery != nil {
		h.feishuKeyDelivery.Stop()
	}
}
func (h *SettingHandler) GetFeishuKeyDeliveryConfig(c *gin.Context) {
	v, err := h.feishuKeyDelivery.Config(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, v)
}
func (h *SettingHandler) SaveFeishuKeyDeliveryConfig(c *gin.Context) {
	var req service.FeishuKeyDeliveryConfig
	if c.ShouldBindJSON(&req) != nil {
		response.BadRequest(c, "Invalid delivery settings")
		return
	}
	v, err := h.feishuKeyDelivery.SaveConfig(c.Request.Context(), req)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, v)
}
func (h *SettingHandler) PreviewFeishuKeyDelivery(c *gin.Context) {
	actor, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "Authentication required")
		return
	}
	var req struct {
		UserIDs []int64 `json:"user_ids"`
		All     bool    `json:"all"`
	}
	if c.ShouldBindJSON(&req) != nil {
		response.BadRequest(c, "Invalid recipient scope")
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Minute)
	defer cancel()
	v, err := h.feishuKeyDelivery.Preview(ctx, actor.UserID, req.UserIDs, req.All)
	if err != nil {
		respondFeishuSetupError(c, err)
		return
	}
	response.Success(c, v)
}
func (h *SettingHandler) GetFeishuKeyDelivery(c *gin.Context) {
	v, err := h.feishuKeyDelivery.Current(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, v)
}
func (h *SettingHandler) StartFeishuKeyDelivery(c *gin.Context) {
	actor, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "Authentication required")
		return
	}
	var req struct {
		PreviewID string `json:"preview_id"`
	}
	if c.ShouldBindJSON(&req) != nil || len(req.PreviewID) != 40 {
		response.BadRequest(c, "Invalid preview")
		return
	}
	v, err := h.feishuKeyDelivery.Start(c.Request.Context(), actor.UserID, req.PreviewID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, v)
}
func (h *SettingHandler) RetryFeishuKeyDelivery(c *gin.Context) {
	actor, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "Authentication required")
		return
	}
	var req struct {
		JobID            string `json:"job_id"`
		IncludeUncertain bool   `json:"include_uncertain"`
	}
	if c.ShouldBindJSON(&req) != nil || len(req.JobID) != 40 {
		response.BadRequest(c, "Invalid batch")
		return
	}
	v, err := h.feishuKeyDelivery.Retry(c.Request.Context(), actor.UserID, req.JobID, req.IncludeUncertain)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, v)
}
