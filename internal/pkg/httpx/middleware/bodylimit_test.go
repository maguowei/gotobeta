package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func bodyLimitRouter(maxBytes int64) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(BodyLimit(maxBytes))
	r.POST("/", func(c *gin.Context) {
		_, _ = c.GetRawData()
		c.Status(http.StatusOK)
	})
	return r
}

func TestBodyLimitRejectsOversizedByContentLength(t *testing.T) {
	r := bodyLimitRouter(16)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(strings.Repeat("x", 100)))
	r.ServeHTTP(w, req)
	if w.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("超长 body 应返回 413，得 %d", w.Code)
	}
}

func TestBodyLimitAllowsWithinLimit(t *testing.T) {
	r := bodyLimitRouter(1024)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("small"))
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("限额内应放行，得 %d", w.Code)
	}
}

func TestBodyLimitDisabledWhenNonPositive(t *testing.T) {
	r := bodyLimitRouter(0)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(strings.Repeat("x", 10_000)))
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("上限为 0 时不限流，应放行，得 %d", w.Code)
	}
}
