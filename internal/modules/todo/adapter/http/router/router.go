package router

import (
	"github.com/gin-gonic/gin"

	"github.com/maguowei/gotobeta/internal/modules/todo/adapter/http/handler"
)

// RegisterRoutes 注册 Todo 路由。
// 传入的 middlewares 会作用于全部 Todo 路由（例如在启用鉴权的服务里要求登录），
// 不传则保持公开（纯 demo 场景）。
func RegisterRoutes(group *gin.RouterGroup, h *handler.TodoHandler, middlewares ...gin.HandlerFunc) {
	if len(middlewares) > 0 {
		group = group.Group("")
		group.Use(middlewares...)
	}
	group.GET("/todos", h.ListTodos)
	group.GET("/todos/:id", h.GetTodo)
	group.POST("/todos", h.CreateTodo)
	group.POST("/todos/:id/complete", h.CompleteTodo)
	group.DELETE("/todos/:id", h.DeleteTodo)
}
