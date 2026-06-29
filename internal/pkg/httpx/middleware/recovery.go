package middleware

import (
	"log/slog"
	"runtime/debug"

	sentrysdk "github.com/getsentry/sentry-go"
	"github.com/gin-gonic/gin"

	"github.com/maguowei/gotobeta/internal/pkg/httpx/response"
	"github.com/maguowei/gotobeta/internal/pkg/requestctx"
)

// Sentry 将当前 HTTP 请求绑定到 Sentry hub。
func Sentry() gin.HandlerFunc {
	return func(c *gin.Context) {
		hub := sentrysdk.CurrentHub().Clone()
		hub.Scope().SetRequest(c.Request)
		setSentryRequestTags(c, hub)
		if requestID := requestctx.GetRequestID(c.Request.Context()); requestID != "" {
			hub.Scope().SetTag("request_id", requestID)
		}

		ctx := sentrysdk.SetHubOnContext(c.Request.Context(), hub)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

// Recovery 捕获 panic。
func Recovery(logger *slog.Logger) gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered any) {
		stack := debug.Stack()
		if hub := sentrysdk.GetHubFromContext(c.Request.Context()); hub != nil {
			setSentryRequestTags(c, hub)
			hub.RecoverWithContext(c.Request.Context(), recovered)
		}
		logger.ErrorContext(c.Request.Context(), "panic recovered",
			slog.Any("panic", recovered),
			slog.String("stack", string(stack)),
		)
		response.ErrorWithCode(c, response.CodeInternal, "服务器内部错误")
	})
}

func setSentryRequestTags(c *gin.Context, hub *sentrysdk.Hub) {
	hub.Scope().SetTag("request_method", c.Request.Method)
	hub.Scope().SetTag("request_path", routeTemplate(c))
}

func routeTemplate(c *gin.Context) string {
	if path := c.FullPath(); path != "" {
		return path
	}
	return "unknown"
}
