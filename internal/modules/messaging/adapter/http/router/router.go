// Package router 注册 messaging 模块路由。
package router

import (
	"github.com/gin-gonic/gin"

	"github.com/maguowei/gotobeta/internal/modules/messaging/adapter/http/handler"
)

// RegisterRoutes 注册会话路由。middlewares 通常为登录鉴权中间件。
func RegisterRoutes(group *gin.RouterGroup, h *handler.ConversationHandler, middlewares ...gin.HandlerFunc) {
	if len(middlewares) > 0 {
		group = group.Group("")
		group.Use(middlewares...)
	}
	group.POST("/workspaces/:ws/conversations", h.CreateConversation)
	group.GET("/workspaces/:ws/conversations", h.ListConversations)
	group.POST("/workspaces/:ws/conversations/:cid/members", h.AddMember)
	group.DELETE("/workspaces/:ws/conversations/:cid/members/:mid", h.RemoveMember)
	group.GET("/workspaces/:ws/conversations/:cid/members", h.ListMembers)
}
