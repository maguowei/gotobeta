package router

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/maguowei/gotobeta/internal/pkg/health"
)

func TestHealthRoutes(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	RegisterRoutes(engine, health.NewRegistry())

	paths := []string{"/health", "/healthz", "/readyz"}
	for _, path := range paths {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, path, nil)
		engine.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("%s: want 200, got %d", path, w.Code)
		}
	}
}
