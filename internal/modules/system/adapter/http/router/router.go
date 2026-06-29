package router

import (
	"github.com/gin-gonic/gin"

	"github.com/maguowei/gotobeta/internal/modules/system/adapter/http/handler"
	"github.com/maguowei/gotobeta/internal/pkg/health"
)

// RegisterRoutes 注册系统路由。
func RegisterRoutes(engine *gin.Engine, registry *health.Registry) {
	h := handler.NewSystemHandler(registry)
	engine.GET("/health", h.Healthz)
	engine.GET("/healthz", h.Healthz)
	engine.GET("/readyz", h.Readyz)
}
