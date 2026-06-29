// Package router 注册 messaging 模块路由。
package router

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/maguowei/gotobeta/internal/modules/messaging/adapter/http/handler"
	"github.com/maguowei/gotobeta/internal/pkg/auth"
	"github.com/maguowei/gotobeta/internal/pkg/httpx/middleware"
)

// UserRateKey 按认证用户 ID 给发消息频控分桶；无登录 claims 时返回空串，
// 由限流中间件回退到 ClientIP 兜底。
func UserRateKey(c *gin.Context) string {
	if claims, ok := auth.ClaimsFromContext(c.Request.Context()); ok && claims.UserID > 0 {
		return strconv.FormatInt(claims.UserID, 10)
	}
	return ""
}

// RegisterRoutes 注册会话与消息路由。middlewares 通常为登录鉴权中间件。
// sendRateLimit 为可选的发消息频控中间件（仅挂在发消息路由），nil 时不限流。
// 所有路由带 :ws，统一挂 WorkspaceScope 把工作区 id 注入 context（DataScope 依据）。
func RegisterRoutes(group *gin.RouterGroup, h *handler.ConversationHandler, mh *handler.MessageHandler, sendRateLimit gin.HandlerFunc, middlewares ...gin.HandlerFunc) {
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

	// 发消息路由额外挂按用户频控，抵御单用户刷消息打爆下游。
	sendHandlers := make([]gin.HandlerFunc, 0, 2)
	if sendRateLimit != nil {
		sendHandlers = append(sendHandlers, sendRateLimit)
	}
	sendHandlers = append(sendHandlers, mh.SendMessage)
	group.POST("/workspaces/:ws/conversations/:cid/messages", sendHandlers...)

	group.GET("/workspaces/:ws/conversations/:cid/messages", mh.PullMessages)
	group.POST("/workspaces/:ws/conversations/:cid/messages/:mid/recall", mh.RecallMessage)
	group.POST("/workspaces/:ws/conversations/:cid/read", mh.ReportRead)
}
