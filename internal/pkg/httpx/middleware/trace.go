package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	traceapi "go.opentelemetry.io/otel/trace"
)

// TraceContext 把 otelhttp 包装的 handler 用 Gin 中间件桥接。
//
// 必须位于中间件链顶端（仅次于 Recovery），让后续中间件与 handler
// 都拥有携带 SpanContext 的 ctx。同时把 traceId 回写到响应头
// X-Trace-Id，便于前端/调用方排障。
func TraceContext(serviceName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		otelhttp.NewHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			c.Request = r
			if sc := traceapi.SpanContextFromContext(r.Context()); sc.IsValid() {
				c.Writer.Header().Set(headerXTraceID, sc.TraceID().String())
			}
			c.Next()
		}), serviceName).ServeHTTP(c.Writer, c.Request)
	}
}
