package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/maguowei/gotobeta/internal/pkg/requestctx"
)

// 自定义响应/请求头名统一在此定义，避免包内字面量漂移。
const (
	headerXRequestID   = "X-Request-ID"
	headerXTraceID     = "X-Trace-ID"
	maxRequestIDLength = 128
)

// RequestID 写入 request id。
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := sanitizeRequestID(c.GetHeader(headerXRequestID))
		if requestID == "" {
			requestID = uuid.NewString()
		}

		ctx := requestctx.WithRequestID(c.Request.Context(), requestID)
		c.Request = c.Request.WithContext(ctx)
		c.Header(headerXRequestID, requestID)

		c.Next()
	}
}

// sanitizeRequestID 拒绝包含 CRLF、控制字符或超长输入的客户端 request id，防止日志/header 注入。
func sanitizeRequestID(raw string) string {
	if raw == "" {
		return ""
	}
	if len(raw) > maxRequestIDLength {
		return ""
	}
	for _, r := range raw {
		if r < 0x20 || r == 0x7f {
			return ""
		}
	}
	return raw
}
