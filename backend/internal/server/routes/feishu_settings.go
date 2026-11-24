package routes

import (
	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/gin-gonic/gin"
)

// Keep the custom routes separate from upstream's shared admin registration.
// Historical regression jobs restore admin.go while compiling current tests.
func registerFeishuSettingsRoutes(admin *gin.RouterGroup, h *handler.Handlers, stepUp middleware.StepUpAuthMiddleware) {
	admin.GET("/users/feishu-sync", h.Admin.Setting.GetFeishuDirectorySync)
	admin.POST("/users/feishu-sync", gin.HandlerFunc(stepUp), h.Admin.Setting.StartFeishuDirectorySync)
	delivery := admin.Group("/users/feishu-key-delivery")
	delivery.GET("/employees", h.Admin.Setting.ListFeishuKeyEmployees)
	delivery.POST("/issue", gin.HandlerFunc(stepUp), h.Admin.Setting.IssueFeishuKeys)
	delivery.GET("/config", h.Admin.Setting.GetFeishuKeyDeliveryConfig)
	delivery.PUT("/config", gin.HandlerFunc(stepUp), h.Admin.Setting.SaveFeishuKeyDeliveryConfig)
	delivery.POST("/preview", gin.HandlerFunc(stepUp), h.Admin.Setting.PreviewFeishuKeyDelivery)
	delivery.GET("", h.Admin.Setting.GetFeishuKeyDelivery)
	delivery.POST("", gin.HandlerFunc(stepUp), h.Admin.Setting.StartFeishuKeyDelivery)
	delivery.POST("/retry", gin.HandlerFunc(stepUp), h.Admin.Setting.RetryFeishuKeyDelivery)
	settings := admin.Group("/settings/feishu")
	settings.GET("", h.Admin.Setting.GetFeishuSettings)
	settings.PUT("", gin.HandlerFunc(stepUp), h.Admin.Setting.UpdateFeishuSettings)
	settings.POST("/email-sync", gin.HandlerFunc(stepUp), h.Admin.Setting.EnableFeishuEmailSync)
	settings.POST("/setup/start", gin.HandlerFunc(stepUp), h.Admin.Setting.StartFeishuSetup)
	settings.POST("/setup/:id/poll", gin.HandlerFunc(stepUp), h.Admin.Setting.PollFeishuSetup)
}
