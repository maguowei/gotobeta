package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestMetricsUsesRoutePatternAndUnknownFallback(t *testing.T) {
	gin.SetMode(gin.TestMode)

	requestsTotal := prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "test_http_requests_total"},
		[]string{"method", "path", "status"},
	)
	requestDuration := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{Name: "test_http_request_duration_seconds"},
		[]string{"method", "path"},
	)

	router := gin.New()
	router.Use(Metrics(HTTPMetrics{
		RequestsTotal:   requestsTotal,
		RequestDuration: requestDuration,
	}))
	router.GET("/items/:id", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/items/123", nil))
	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/items/123/extra", nil))

	if got := testutil.ToFloat64(requestsTotal.WithLabelValues(http.MethodGet, "/items/:id", "2xx")); got != 1 {
		t.Fatalf("route pattern metric count = %v, want 1", got)
	}

	if got := testutil.ToFloat64(requestsTotal.WithLabelValues(http.MethodGet, "unknown", "4xx")); got != 1 {
		t.Fatalf("unknown route metric count = %v, want 1", got)
	}

	if got := testutil.ToFloat64(requestsTotal.WithLabelValues(http.MethodGet, "/items/123/extra", "4xx")); got != 0 {
		t.Fatalf("raw unmatched path metric count = %v, want 0", got)
	}

	// 验证未分组的原始状态码不再产生 series。
	if got := testutil.ToFloat64(requestsTotal.WithLabelValues(http.MethodGet, "/items/:id", "204")); got != 0 {
		t.Fatalf("raw status code metric should be 0, got %v", got)
	}
}
