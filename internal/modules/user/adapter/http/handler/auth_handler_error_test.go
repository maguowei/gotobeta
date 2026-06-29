package handler

import (
	stderrors "errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/maguowei/gotobeta/internal/pkg/auth"
)

// errUseCase 用于命中 usecase 返回 error 的分支。
var errUseCase = stderrors.New("usecase failed")

// TestAuthHandlerUseCaseErrors 覆盖 AuthHandler 各端点 usecase 返回 error 时的错误分支。
func TestAuthHandlerUseCaseErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)
	usecase := &mockAuthUseCase{err: errUseCase}
	handler := NewAuthHandler(usecase)
	router := gin.New()
	router.POST("/auth/register", handler.Register)
	router.POST("/auth/login", handler.Login)
	router.POST("/auth/refresh", handler.Refresh)
	router.POST("/auth/logout", handler.Logout)
	router.POST("/auth/password/forgot", handler.ForgotPassword)
	router.POST("/auth/password/reset", handler.ResetPassword)
	router.POST("/auth/email/verify", handler.VerifyEmail)
	router.POST("/auth/oauth/token", handler.OAuthToken)
	router.GET("/auth/oauth/:provider/start", handler.StartOAuth)
	router.GET("/auth/oauth/:provider/callback", handler.OAuthCallback)

	tests := []struct {
		name   string
		method string
		path   string
		body   string
	}{
		{name: "register", method: http.MethodPost, path: "/auth/register", body: `{"email":"a@b.com","password":"password-123","displayName":"A"}`},
		{name: "login", method: http.MethodPost, path: "/auth/login", body: `{"email":"a@b.com","password":"password-123"}`},
		{name: "refresh", method: http.MethodPost, path: "/auth/refresh", body: `{"refreshToken":"refresh"}`},
		{name: "logout", method: http.MethodPost, path: "/auth/logout", body: `{"refreshToken":"refresh"}`},
		{name: "forgot", method: http.MethodPost, path: "/auth/password/forgot", body: `{"email":"a@b.com"}`},
		{name: "reset", method: http.MethodPost, path: "/auth/password/reset", body: `{"token":"t","newPassword":"password-123"}`},
		{name: "verify", method: http.MethodPost, path: "/auth/email/verify", body: `{"token":"t"}`},
		{name: "oauthToken", method: http.MethodPost, path: "/auth/oauth/token", body: `{"code":"c"}`},
		{name: "startOAuth", method: http.MethodGet, path: "/auth/oauth/github/start", body: ""},
		{name: "oauthCallback", method: http.MethodGet, path: "/auth/oauth/github/callback?state=s&code=c", body: ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var bodyReader io.Reader = http.NoBody
			if tc.body != "" {
				bodyReader = strings.NewReader(tc.body)
			}
			req := httptest.NewRequestWithContext(t.Context(), tc.method, tc.path, bodyReader)
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			// usecase 返回 error 时不能是 2xx 成功或 3xx 重定向。
			if rec.Code < http.StatusBadRequest {
				t.Fatalf("status = %d, want client/server error; body=%s", rec.Code, rec.Body.String())
			}
		})
	}
}

// TestAuthHandlerRejectsInvalidJSON 覆盖各端点 JSON 绑定失败的参数错误分支。
func TestAuthHandlerRejectsInvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewAuthHandler(&mockAuthUseCase{})
	router := gin.New()
	router.POST("/auth/login", handler.Login)
	router.POST("/auth/refresh", handler.Refresh)
	router.POST("/auth/logout", handler.Logout)
	router.POST("/auth/password/forgot", handler.ForgotPassword)
	router.POST("/auth/password/reset", handler.ResetPassword)
	router.POST("/auth/email/verify", handler.VerifyEmail)
	router.POST("/auth/oauth/token", handler.OAuthToken)

	paths := []string{
		"/auth/login",
		"/auth/refresh",
		"/auth/logout",
		"/auth/password/forgot",
		"/auth/password/reset",
		"/auth/email/verify",
		"/auth/oauth/token",
	}

	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, path, strings.NewReader(`{`))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400; body=%s", rec.Code, rec.Body.String())
			}
		})
	}
}

// TestAuthHandlerSendEmailVerificationMissingClaims 覆盖未认证时 RequireClaims 返回 false 的分支。
func TestAuthHandlerSendEmailVerificationMissingClaims(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewAuthHandler(&mockAuthUseCase{})
	router := gin.New()
	router.POST("/users/me/email/verification", handler.SendEmailVerification)

	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/users/me/email/verification", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401; body=%s", rec.Code, rec.Body.String())
	}
}

// TestAuthHandlerSendEmailVerificationUseCaseError 覆盖已认证但 usecase 返回 error 的分支。
func TestAuthHandlerSendEmailVerificationUseCaseError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewAuthHandler(&mockAuthUseCase{err: errUseCase})
	router := gin.New()
	router.POST("/users/me/email/verification", handler.SendEmailVerification)

	req := httptest.NewRequestWithContext(auth.WithClaims(t.Context(), &auth.Claims{UserID: 9}), http.MethodPost, "/users/me/email/verification", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code < http.StatusBadRequest {
		t.Fatalf("status = %d, want error; body=%s", rec.Code, rec.Body.String())
	}
}
