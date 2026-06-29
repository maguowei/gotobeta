package router

import (
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/maguowei/gotobeta/internal/modules/media/adapter/http/handler"
)

func TestRegisterRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	e := gin.New()
	RegisterRoutes(e.Group("/api/v1"), &handler.AttachmentHandler{}, func(c *gin.Context) { c.Next() })
	if len(e.Routes()) == 0 {
		t.Fatal("应注册路由")
	}
}
