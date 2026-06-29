package router

import (
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/maguowei/gotobeta/internal/modules/workspace/adapter/http/handler"
)

func TestRegisterRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	e := gin.New()
	RegisterRoutes(e.Group("/api/v1"), &handler.WorkspaceHandler{}, func(c *gin.Context) { c.Next() })
	if len(e.Routes()) != 5 {
		t.Fatalf("应注册 5 条路由, got %d", len(e.Routes()))
	}
}
