// Package router 注册 workspace 模块路由。
package router

import (
	"github.com/gin-gonic/gin"

	"github.com/maguowei/gotobeta/internal/modules/workspace/adapter/http/handler"
)

// RegisterRoutes 注册工作区路由。middlewares 通常为登录鉴权中间件。
func RegisterRoutes(group *gin.RouterGroup, h *handler.WorkspaceHandler, middlewares ...gin.HandlerFunc) {
	if len(middlewares) > 0 {
		group = group.Group("")
		group.Use(middlewares...)
	}
	group.POST("/workspaces", h.CreateWorkspace)
	group.GET("/workspaces", h.ListMyWorkspaces)
	group.POST("/workspaces/:ws/members", h.InviteMember)
	group.POST("/workspaces/:ws/members/:uid/roles", h.AssignRole)
	group.GET("/workspaces/:ws/roles", h.ListRoles)
}
