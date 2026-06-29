package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/maguowei/gotobeta/internal/pkg/auth"
	"github.com/maguowei/gotobeta/internal/pkg/rbac"
)

func TestRequirePermissionAllowsMatchingRole(t *testing.T) {
	router := testRouterWithClaims(&auth.Claims{Roles: []string{"admin"}})
	router.GET("/admin", RequirePermission(rbac.Policy{"admin": []rbac.Permission{rbac.Permission("users:read")}}, "users:read"), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/admin", nil))
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d, body = %s", recorder.Code, http.StatusNoContent, recorder.Body.String())
	}
}

func TestRequirePermissionRejectsMissingPermission(t *testing.T) {
	router := testRouterWithClaims(&auth.Claims{Roles: []string{"viewer"}})
	router.GET("/admin", RequirePermission(rbac.Policy{"admin": []rbac.Permission{rbac.Permission("users:read")}}, "users:read"), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/admin", nil))
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d, body = %s", recorder.Code, http.StatusForbidden, recorder.Body.String())
	}
}

func TestRequirePermissionRejectsMissingClaims(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/admin", RequirePermission(rbac.Policy{"admin": []rbac.Permission{rbac.Permission("users:read")}}, "users:read"), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/admin", nil))
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d, body = %s", recorder.Code, http.StatusUnauthorized, recorder.Body.String())
	}
}

func testRouterWithClaims(claims *auth.Claims) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Request = c.Request.WithContext(auth.WithClaims(c.Request.Context(), claims))
		c.Next()
	})
	return router
}
