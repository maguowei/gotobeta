package router

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/maguowei/gotobeta/internal/modules/user/adapter/http/handler"
)

func TestRegisterRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	RegisterRoutes(
		engine.Group("/api/v1"),
		handler.NewAuthHandler(nil),
		handler.NewUserHandler(nil),
		func(c *gin.Context) { c.Next() },
		func(c *gin.Context) { c.Next() },
	)

	if got := len(engine.Routes()); got != 16 {
		t.Fatalf("route count = %d, want 16", got)
	}
}

// TestRegisterRoutesAppliesRateLimitToCredentialEndpoints 验证限流中间件作用于
// 凭据敏感端点（如 /auth/login），但不影响 OAuth 浏览器跳转端点。
func TestRegisterRoutesAppliesRateLimitToCredentialEndpoints(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	// nil 依赖的真实 handler 会 panic；用 Recovery 容纳，断言只看是否被限流（429）。
	engine.Use(gin.Recovery())
	blocked := func(c *gin.Context) { c.AbortWithStatus(http.StatusTooManyRequests) }
	RegisterRoutes(
		engine.Group("/api/v1"),
		handler.NewAuthHandler(nil),
		handler.NewUserHandler(nil),
		func(c *gin.Context) { c.Next() },
		blocked,
	)

	loginReq := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/v1/auth/login", http.NoBody)
	loginRec := httptest.NewRecorder()
	engine.ServeHTTP(loginRec, loginReq)
	if loginRec.Code != http.StatusTooManyRequests {
		t.Fatalf("/auth/login status = %d, want 429 (must be rate limited)", loginRec.Code)
	}

	oauthReq := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/auth/oauth/github/start", http.NoBody)
	oauthRec := httptest.NewRecorder()
	engine.ServeHTTP(oauthRec, oauthReq)
	if oauthRec.Code == http.StatusTooManyRequests {
		t.Fatal("/auth/oauth/:provider/start must not be rate limited")
	}
}
