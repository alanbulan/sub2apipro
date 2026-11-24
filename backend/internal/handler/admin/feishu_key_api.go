package admin

import (
	"context"
	"encoding/json"
	"io"
	"strconv"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

func (h *SettingHandler) ListFeishuKeyEmployees(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	if !requireAdminKeyJWT(c) {
		return
	}
	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil {
		response.BadRequest(c, "Invalid page")
		return
	}
	pageSize, err := strconv.Atoi(c.DefaultQuery("page_size", "50"))
	if err != nil {
		response.BadRequest(c, "Invalid page_size")
		return
	}
	var issued *bool
	if value, exists := c.GetQuery("key_issued"); exists {
		if value != "true" && value != "false" {
			response.BadRequest(c, "key_issued must be true or false")
			return
		}
		flag := value == "true"
		issued = &flag
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Minute)
	defer cancel()
	v, err := h.feishuKeyDelivery.ListEmployees(ctx, page, pageSize, issued, c.Query("search"))
	if err != nil {
		respondFeishuSetupError(c, err)
		return
	}
	response.Success(c, v)
}

func (h *SettingHandler) IssueFeishuKeys(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	if !requireAdminKeyJWT(c) {
		return
	}
	actor, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok || actor.UserID <= 0 {
		response.Unauthorized(c, "Authentication required")
		return
	}
	var req service.FeishuKeyIssueRequest
	decoder := json.NewDecoder(io.LimitReader(c.Request.Body, 16385))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&req) != nil {
		response.BadRequest(c, "Send user_id or user_ids, and optional group_id")
		return
	}
	var trailing any
	if decoder.Decode(&trailing) != io.EOF {
		response.BadRequest(c, "Send one JSON object")
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Minute)
	defer cancel()
	middleware.SetAuditAction(c, "admin.feishu_keys.issue")
	v, err := h.feishuKeyDelivery.IssueKeys(ctx, actor.UserID, req)
	if err != nil {
		respondFeishuSetupError(c, err)
		return
	}
	middleware.SetAuditExtra(c, map[string]any{"source": "api", "requested_count": len(v.Items)})
	response.Success(c, v)
}
