package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func TestTraceContextAddsTraceIDResponseHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	otel.SetTracerProvider(sdktrace.NewTracerProvider())

	router := gin.New()
	router.Use(TraceContext("gotobeta"))
	router.GET("/ping", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/ping", nil)
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status code = %d, want %d", recorder.Code, http.StatusNoContent)
	}
	if recorder.Header().Get("X-Trace-Id") == "" {
		t.Fatalf("X-Trace-Id header is empty")
	}
}
