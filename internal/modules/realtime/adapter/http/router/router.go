// Package router 注册 realtime 模块路由。
package router

import (
	"github.com/gin-gonic/gin"

	"github.com/maguowei/gotobeta/internal/modules/realtime/adapter/http/handler"
	"github.com/maguowei/gotobeta/internal/modules/realtime/adapter/ws"
)

// RegisterRoutes 注册 ticket 与 WS 路由。
//
// POST /ws/ticket 需登录鉴权（authMiddlewares）；GET /ws 不走 JWT，改由一次性 ticket 鉴权。
func RegisterRoutes(group *gin.RouterGroup, th *handler.TicketHandler, gw *ws.Gateway, authMiddlewares ...gin.HandlerFunc) {
	ticketGroup := group.Group("/ws")
	ticketGroup.Use(authMiddlewares...)
	ticketGroup.POST("/ticket", th.IssueTicket)

	group.GET("/ws", gw.Handle)
}
