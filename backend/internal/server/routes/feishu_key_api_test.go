//go:build unit

package routes

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/handler"
	adminhandler "github.com/Wei-Shaw/sub2api/internal/handler/admin"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestFeishuKeyAPIRouteRequiresExistingStepUp(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	admin := r.Group("/api/v1/admin")
	stepUp := middleware.StepUpAuthMiddleware(func(c *gin.Context) {
		c.AbortWithStatusJSON(403, gin.H{"reason": "STEP_UP_REQUIRED"})
	})
	h := &handler.Handlers{Admin: &handler.AdminHandlers{Setting: &adminhandler.SettingHandler{}}}
	registerFeishuSettingsRoutes(admin, h, stepUp)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("POST", "/api/v1/admin/users/feishu-key-delivery/issue", strings.NewReader(`{"user_id":7}`)))
	require.Equal(t, 403, w.Code)
	require.Contains(t, w.Body.String(), "STEP_UP_REQUIRED")
}
