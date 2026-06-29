package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/maguowei/gotobeta/internal/pkg/auth"
)

// newUserRouter 注册 UserHandler 全部路由，供错误分支测试复用。
func newUserRouter(usecase UserUseCase) *gin.Engine {
	handler := NewUserHandler(usecase)
	router := gin.New()
	router.GET("/users/me", handler.Me)
	router.PATCH("/users/me", handler.UpdateMe)
	router.PUT("/users/me/password", handler.ChangePassword)
	router.GET("/users/me/identities", handler.ListIdentities)
	router.DELETE("/users/me/identities/:provider", handler.UnlinkIdentity)
	return router
}

// TestUserHandlerMissingClaims 覆盖各端点未认证时 RequireClaims 返回 false 的分支。
func TestUserHandlerMissingClaims(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := newUserRouter(&mockAuthUseCase{})

	tests := []struct {
		name   string
		method string
		path   string
	}{
		{name: "me", method: http.MethodGet, path: "/users/me"},
		{name: "update", method: http.MethodPatch, path: "/users/me"},
		{name: "password", method: http.MethodPut, path: "/users/me/password"},
		{name: "identities", method: http.MethodGet, path: "/users/me/identities"},
		{name: "unlink", method: http.MethodDelete, path: "/users/me/identities/github"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequestWithContext(t.Context(), tc.method, tc.path, http.NoBody)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, want 401; body=%s", rec.Code, rec.Body.String())
			}
		})
	}
}

// TestUserHandlerInvalidJSON 覆盖带 body 端点 JSON 绑定失败的参数错误分支。
func TestUserHandlerInvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := newUserRouter(&mockAuthUseCase{})

	tests := []struct {
		name   string
		method string
		path   string
	}{
		{name: "update", method: http.MethodPatch, path: "/users/me"},
		{name: "password", method: http.MethodPut, path: "/users/me/password"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := auth.WithClaims(t.Context(), &auth.Claims{UserID: 9})
			req := httptest.NewRequestWithContext(ctx, tc.method, tc.path, strings.NewReader(`{`))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400; body=%s", rec.Code, rec.Body.String())
			}
		})
	}
}

// TestUserHandlerUseCaseErrors 覆盖各端点已认证但 usecase 返回 error 的分支。
func TestUserHandlerUseCaseErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := newUserRouter(&mockAuthUseCase{err: errUseCase})

	tests := []struct {
		name   string
		method string
		path   string
		body   string
	}{
		{name: "me", method: http.MethodGet, path: "/users/me"},
		{name: "update", method: http.MethodPatch, path: "/users/me", body: `{"displayName":"A","avatarUrl":"https://example.com/a.png"}`},
		{name: "password", method: http.MethodPut, path: "/users/me/password", body: `{"oldPassword":"password-123","newPassword":"password-456"}`},
		{name: "identities", method: http.MethodGet, path: "/users/me/identities"},
		{name: "unlink", method: http.MethodDelete, path: "/users/me/identities/github"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := auth.WithClaims(t.Context(), &auth.Claims{UserID: 9})
			var bodyReader = http.NoBody
			req := httptest.NewRequestWithContext(ctx, tc.method, tc.path, bodyReader)
			if tc.body != "" {
				req = httptest.NewRequestWithContext(ctx, tc.method, tc.path, strings.NewReader(tc.body))
			}
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			if rec.Code < http.StatusBadRequest {
				t.Fatalf("status = %d, want error; body=%s", rec.Code, rec.Body.String())
			}
		})
	}
}
