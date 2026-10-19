package admin

import (
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

func (h *SettingHandler) SetFeishuSetupStore(r service.FeishuSetupStore) {
	h.feishuSetup = service.NewFeishuSetupService(h.settingService, r)
}
func (h *SettingHandler) GetFeishuSettings(c *gin.Context) {
	v, err := h.settingService.GetFeishuSettings(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, v)
}
func (h *SettingHandler) UpdateFeishuSettings(c *gin.Context) {
	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request")
		return
	}
	release, err := h.feishuSetup.Lock(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	defer release()
	v, err := h.settingService.GetFeishuSettings(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	v.Enabled = req.Enabled
	if err = h.settingService.SaveFeishuSettings(c.Request.Context(), v); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, v)
}
func (h *SettingHandler) StartFeishuSetup(c *gin.Context) {
	subject, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "Authentication required")
		return
	}
	v, err := h.feishuSetup.Start(c.Request.Context(), subject.UserID)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.Success(c, v)
}
func (h *SettingHandler) PollFeishuSetup(c *gin.Context) {
	subject, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "Authentication required")
		return
	}
	v, err := h.feishuSetup.Poll(c.Request.Context(), c.Param("id"), subject.UserID)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.Success(c, v)
}
