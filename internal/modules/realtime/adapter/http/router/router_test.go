package router

import (
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/maguowei/gotobeta/internal/modules/realtime/adapter/http/handler"
	"github.com/maguowei/gotobeta/internal/modules/realtime/adapter/ws"
)

func TestRegisterRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	e := gin.New()
	RegisterRoutes(e.Group("/api/v1"), &handler.TicketHandler{}, &ws.Gateway{}, nil, func(c *gin.Context) { c.Next() })
	if len(e.Routes()) < 2 {
		t.Fatalf("应注册 ticket 与 ws 路由, got %d", len(e.Routes()))
	}
}
