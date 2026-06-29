package middleware

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/maguowei/gotobeta/internal/pkg/requestctx"
)

// Logger 记录请求日志。
func Logger(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		logger.InfoContext(c.Request.Context(), "http request",
			slog.String("requestId", requestctx.GetRequestID(c.Request.Context())),
			slog.String("method", c.Request.Method),
			slog.String("path", c.FullPath()),
			slog.Int("status", c.Writer.Status()),
			slog.Duration("latency", time.Since(start)),
		)
	}
}
