package router

import (
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/maguowei/gotobeta/internal/modules/messaging/adapter/http/handler"
)

func TestRegisterRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	e := gin.New()
	// 仅校验注册不 panic；handler 为 nil 时方法值延迟到调用才解引用。
	RegisterRoutes(e.Group("/api/v1"), &handler.ConversationHandler{}, &handler.MessageHandler{}, func(c *gin.Context) { c.Next() })
	if len(e.Routes()) == 0 {
		t.Fatal("应注册路由")
	}
}
