// Package router 注册 workspace 模块路由。
package router

import (
	"github.com/gin-gonic/gin"

	"github.com/maguowei/gotobeta/internal/modules/workspace/adapter/http/handler"
	"github.com/maguowei/gotobeta/internal/pkg/httpx/middleware"
)

// RegisterRoutes 注册工作区路由。middlewares 通常为登录鉴权中间件。
// 带 :ws 的路由额外挂 WorkspaceScope，把受信路由段的工作区 id 注入 context，
// 作为 DataScope 工作区隔离的依据（repo 层拦截器据此过滤）。
func RegisterRoutes(group *gin.RouterGroup, h *handler.WorkspaceHandler, middlewares ...gin.HandlerFunc) {
	if len(middlewares) > 0 {
		group = group.Group("")
		group.Use(middlewares...)
	}
	group.POST("/workspaces", h.CreateWorkspace)
	group.GET("/workspaces", h.ListMyWorkspaces)

	scoped := group.Group("", middleware.WorkspaceScope("ws"))
	scoped.POST("/workspaces/:ws/members", h.InviteMember)
	scoped.POST("/workspaces/:ws/members/:uid/roles", h.AssignRole)
	scoped.GET("/workspaces/:ws/roles", h.ListRoles)
}
