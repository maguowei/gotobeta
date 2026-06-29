// Package router 注册 media 模块路由。
package router

import (
	"github.com/gin-gonic/gin"

	"github.com/maguowei/gotobeta/internal/modules/media/adapter/http/handler"
)

// RegisterRoutes 注册附件路由。middlewares 通常为登录鉴权中间件。
func RegisterRoutes(group *gin.RouterGroup, h *handler.AttachmentHandler, middlewares ...gin.HandlerFunc) {
	if len(middlewares) > 0 {
		group = group.Group("")
		group.Use(middlewares...)
	}
	group.POST("/attachments/presign", h.Presign)
	group.POST("/attachments/:id/commit", h.Commit)
}
