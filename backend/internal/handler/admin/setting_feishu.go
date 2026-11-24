package admin

import (
	"errors"

	"github.com/Wei-Shaw/sub2api/internal/pkg/feishu"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

func (h *SettingHandler) SetFeishuSetupStore(r service.FeishuSetupStore) {
	h.feishuSetup = service.NewFeishuSetupService(h.settingService, r)
}
func (h *SettingHandler) SetFeishuDirectoryAuth(auth *service.AuthService) {
	h.feishuDirectory = service.NewFeishuDirectoryService(auth, h.feishuSetup)
}
func (h *SettingHandler) StartFeishuDirectorySync(c *gin.Context) {
	job, err := h.feishuDirectory.Start(c.Request.Context())
	if err != nil {
		respondFeishuSetupError(c, err)
		return
	}
	response.Success(c, job)
}
func (h *SettingHandler) GetFeishuDirectorySync(c *gin.Context) {
	job, err := h.feishuDirectory.Current(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, job)
}
func (h *SettingHandler) GetFeishuSettings(c *gin.Context) {
	v, err := h.settingService.GetFeishuSettings(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, v)
}
func (h *SettingHandler) EnableFeishuEmailSync(c *gin.Context) {
	v, err := h.feishuSetup.EnableEmailSync(c.Request.Context())
	if err != nil {
		respondFeishuSetupError(c, err)
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
	var req struct {
		SiteOrigin string `json:"site_origin"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request")
		return
	}
	// Browsers supply Origin independently of the JSON body. Configuration falls
	// back to the current browser site only if the canonical frontend URL is unset.
	if origin := c.GetHeader("Origin"); origin != "" && origin != req.SiteOrigin {
		response.BadRequest(c, "Site origin mismatch")
		return
	}
	v, err := h.feishuSetup.Start(c.Request.Context(), subject.UserID, req.SiteOrigin)
	if err != nil {
		respondFeishuSetupError(c, err)
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
		respondFeishuSetupError(c, err)
		return
	}
	response.Success(c, v)
}

func respondFeishuSetupError(c *gin.Context, err error) {
	var providerErr *feishu.Error
	if errors.As(err, &providerErr) {
		response.BadRequest(c, providerErr.Error())
		return
	}
	response.ErrorFrom(c, err)
}
