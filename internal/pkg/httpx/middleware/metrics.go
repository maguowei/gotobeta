package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/maguowei/gotobeta/internal/pkg/observe"
	"github.com/prometheus/client_golang/prometheus"
)

// HTTPMetrics 持有 HTTP 中间件所需的 Prometheus 收集器。
// 由组合根从 infra metrics 注入（应保证非 nil），中间件本身不依赖 infra 层。
type HTTPMetrics struct {
	RequestsTotal   *prometheus.CounterVec
	RequestDuration *prometheus.HistogramVec
}

// Metrics 记录 HTTP 指标。
func Metrics(m HTTPMetrics) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		path := c.FullPath()
		if path == "" {
			path = "unknown"
		}

		status := observe.StatusClass(c.Writer.Status())
		m.RequestsTotal.WithLabelValues(c.Request.Method, path, status).Inc()

		duration := time.Since(start).Seconds()
		observer := m.RequestDuration.WithLabelValues(c.Request.Method, path)
		observe.ObserveWithTraceID(c.Request.Context(), observer, duration)
	}
}
