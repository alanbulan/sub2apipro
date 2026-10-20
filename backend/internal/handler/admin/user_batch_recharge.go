package admin

import (
	"context"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

func (h *UserHandler) BatchRecharge(c *gin.Context) {
	var req struct {
		UserIDs []int64 `json:"user_ids" binding:"required,min=1,max=500,dive,gt=0"`
		Amount  float64 `json:"amount" binding:"required,gt=0,lte=1000000000"`
		Notes   string  `json:"notes" binding:"max=1000"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid recharge request")
		return
	}
	if c.GetHeader("Idempotency-Key") == "" {
		response.BadRequest(c, "Idempotency-Key is required")
		return
	}
	svc, ok := h.adminService.(interface {
		BatchRechargeUsers(context.Context, []int64, float64, string, string) (*service.BatchUserRechargeResult, error)
	})
	if !ok {
		response.ErrorFrom(c, service.ErrServiceUnavailable)
		return
	}
	executeAdminIdempotentJSONWithTimeout(c, "admin.users.batch-recharge", req, service.DefaultWriteIdempotencyTTL(), 60*time.Second, func(ctx context.Context) (any, error) {
		return svc.BatchRechargeUsers(ctx, req.UserIDs, req.Amount, req.Notes, adminActorScope(c)+":"+c.GetHeader("Idempotency-Key"))
	})
}
