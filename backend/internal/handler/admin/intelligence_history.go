package admin

import (
	"net/http"
	"strconv"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/gin-gonic/gin"
)

func (h *ScheduledTestHandler) GetResult(c *gin.Context) {
	planID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || planID <= 0 {
		response.BadRequest(c, "invalid plan id")
		return
	}
	resultID, err := strconv.ParseInt(c.Param("resultID"), 10, 64)
	if err != nil || resultID <= 0 {
		response.BadRequest(c, "invalid result id")
		return
	}
	result, err := h.scheduledTestSvc.GetResult(c.Request.Context(), planID, resultID)
	if err != nil {
		response.NotFound(c, "result not found or expired")
		return
	}
	c.JSON(http.StatusOK, result)
}

// ListPelicanHistory returns account-independent summaries for the record dashboard.
// Optional account_id narrows them to one account for the IQ test's records window.
func (h *ScheduledTestHandler) ListPelicanHistory(c *gin.Context) {
	beforeID, err := strconv.ParseInt(c.DefaultQuery("before_id", "0"), 10, 64)
	if err != nil || beforeID < 0 {
		response.BadRequest(c, "invalid before_id")
		return
	}
	accountID, err := strconv.ParseInt(c.DefaultQuery("account_id", "0"), 10, 64)
	if err != nil || accountID < 0 {
		response.BadRequest(c, "invalid account_id")
		return
	}
	page, err := h.scheduledTestSvc.ListPelicanHistory(c.Request.Context(), beforeID, 100, accountID)
	if err != nil {
		response.InternalError(c, "Failed to load pelican history")
		return
	}
	c.JSON(http.StatusOK, page)
}
