//go:build unit

package admin

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestFeishuKeyAPIHandlersRequireAdminLoginJWT(t *testing.T) {
	h := &SettingHandler{}
	for _, handler := range []gin.HandlerFunc{h.ListFeishuKeyEmployees, h.IssueFeishuKeys} {
		for _, auth := range []struct{ method, role string }{{"", "admin"}, {"admin_api_key", "admin"}, {"jwt", "user"}} {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Set("auth_method", auth.method)
			c.Set("user_role", auth.role)
			handler(c)
			require.Equal(t, 403, w.Code)
			require.Equal(t, "no-store", w.Header().Get("Cache-Control"))
		}
	}
}

func TestFeishuKeyAPIHandlerRejectsUnknownOrMultipleJSONObjects(t *testing.T) {
	h := &SettingHandler{}
	for _, body := range []string{`{`, `{"user_id":7,"initial_balance":100}`, `{"user_id":7} {"user_id":8}`} {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/api/v1/admin/users/feishu-key-delivery/issue", strings.NewReader(body))
		c.Set("auth_method", service.AuditAuthMethodJWT)
		c.Set("user_role", "admin")
		c.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: 1})
		h.IssueFeishuKeys(c)
		require.Equal(t, 400, w.Code)
		require.Equal(t, "no-store", w.Header().Get("Cache-Control"))
	}
}
