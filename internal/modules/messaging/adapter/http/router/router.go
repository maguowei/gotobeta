// Package router 注册 messaging 模块路由。
package router

import (
	"github.com/gin-gonic/gin"

	"github.com/maguowei/gotobeta/internal/modules/messaging/adapter/http/handler"
	"github.com/maguowei/gotobeta/internal/pkg/httpx/middleware"
)

// RegisterRoutes 注册会话与消息路由。middlewares 通常为登录鉴权中间件。
// 所有路由带 :ws，统一挂 WorkspaceScope 把工作区 id 注入 context（DataScope 依据）。
func RegisterRoutes(group *gin.RouterGroup, h *handler.ConversationHandler, mh *handler.MessageHandler, middlewares ...gin.HandlerFunc) {
	if len(middlewares) > 0 {
		group = group.Group("")
		group.Use(middlewares...)
	}
	group.Use(middleware.WorkspaceScope("ws"))
	group.POST("/workspaces/:ws/conversations", h.CreateConversation)
	group.GET("/workspaces/:ws/conversations", h.ListConversations)
	group.POST("/workspaces/:ws/conversations/:cid/members", h.AddMember)
	group.DELETE("/workspaces/:ws/conversations/:cid/members/:mid", h.RemoveMember)
	group.GET("/workspaces/:ws/conversations/:cid/members", h.ListMembers)

	group.POST("/workspaces/:ws/conversations/:cid/messages", mh.SendMessage)
	group.GET("/workspaces/:ws/conversations/:cid/messages", mh.PullMessages)
	group.POST("/workspaces/:ws/conversations/:cid/messages/:mid/recall", mh.RecallMessage)
	group.POST("/workspaces/:ws/conversations/:cid/read", mh.ReportRead)
}
