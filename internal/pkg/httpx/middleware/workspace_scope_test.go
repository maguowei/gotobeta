package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/maguowei/gotobeta/internal/pkg/requestctx"
)

func TestWorkspaceScopeInjectsID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	var got int64
	var ok bool
	router.GET("/workspaces/:ws/x", WorkspaceScope("ws"), func(c *gin.Context) {
		got, ok = requestctx.WorkspaceID(c.Request.Context())
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/workspaces/42/x", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if !ok || got != 42 {
		t.Fatalf("workspace id = (%d, %v), want (42, true)", got, ok)
	}
}

func TestWorkspaceScopeRejectsInvalidParam(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/workspaces/:ws/x", WorkspaceScope("ws"), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/workspaces/abc/x", nil))

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}
