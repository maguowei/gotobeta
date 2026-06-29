package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	usercmd "github.com/maguowei/gotobeta/internal/modules/user/application/command"
	userquery "github.com/maguowei/gotobeta/internal/modules/user/application/query"
	userresult "github.com/maguowei/gotobeta/internal/modules/user/application/result"
	usersvc "github.com/maguowei/gotobeta/internal/modules/user/application/service"
	"github.com/maguowei/gotobeta/internal/pkg/auth"
)

// 编译期断言：应用服务实现必须满足 handler 声明的用例接口，
// 方法签名漂移在最近的修改点直接编译失败。
var (
	_ AuthUseCase = (*usersvc.AuthService)(nil)
	_ UserUseCase = (*usersvc.AuthService)(nil)
)

// mockAuthUseCase 同时实现 AuthUseCase 和 UserUseCase。
type mockAuthUseCase struct {
	registerCmd    usercmd.RegisterCommand
	loginCmd       usercmd.LoginCommand
	refreshToken   string
	userID         int64
	updateCmd      usercmd.UpdateProfileCommand
	passwordCmd    usercmd.ChangePasswordCommand
	unlinkProvider string
	authOut        *userresult.AuthResult
	userOut        *userresult.UserResult
	identities     []*userresult.IdentityResult
	err            error
}

func (s *mockAuthUseCase) Register(_ context.Context, cmd usercmd.RegisterCommand) (*userresult.AuthResult, error) {
	s.registerCmd = cmd
	return s.authOut, s.err
}

func (s *mockAuthUseCase) Login(_ context.Context, cmd usercmd.LoginCommand) (*userresult.AuthResult, error) {
	s.loginCmd = cmd
	return s.authOut, s.err
}

func (s *mockAuthUseCase) Refresh(_ context.Context, cmd usercmd.RefreshTokenCommand) (*userresult.AuthResult, error) {
	s.refreshToken = cmd.RefreshToken
	return s.authOut, s.err
}
func (s *mockAuthUseCase) Logout(_ context.Context, cmd usercmd.LogoutCommand) error {
	s.refreshToken = cmd.RefreshToken
	return s.err
}
func (s *mockAuthUseCase) ForgotPassword(context.Context, usercmd.ForgotPasswordCommand) error {
	return s.err
}
func (s *mockAuthUseCase) ResetPassword(context.Context, usercmd.ResetPasswordCommand) error {
	return s.err
}
func (s *mockAuthUseCase) SendEmailVerification(_ context.Context, userID int64) error {
	s.userID = userID
	return s.err
}
func (s *mockAuthUseCase) VerifyEmail(context.Context, usercmd.VerifyEmailCommand) error {
	return s.err
}
func (s *mockAuthUseCase) StartOAuth(context.Context, usercmd.StartOAuthCommand) (*userresult.OAuthStartResult, error) {
	return &userresult.OAuthStartResult{AuthURL: "https://example.com/oauth"}, s.err
}
func (s *mockAuthUseCase) HandleOAuthCallback(context.Context, usercmd.HandleOAuthCallbackCommand) (*userresult.OAuthCallbackResult, error) {
	return &userresult.OAuthCallbackResult{RedirectURL: "https://app.example.com/callback?code=abc"}, s.err
}
func (s *mockAuthUseCase) ExchangeOAuthLoginCode(context.Context, usercmd.ExchangeOAuthLoginCodeCommand) (*userresult.AuthResult, error) {
	return s.authOut, s.err
}
func (s *mockAuthUseCase) CurrentUser(_ context.Context, query userquery.GetCurrentUserQuery) (*userresult.UserResult, error) {
	s.userID = query.UserID
	return s.userOut, s.err
}
func (s *mockAuthUseCase) UpdateProfile(_ context.Context, cmd usercmd.UpdateProfileCommand) (*userresult.UserResult, error) {
	s.userID = cmd.UserID
	s.updateCmd = cmd
	return s.userOut, s.err
}
func (s *mockAuthUseCase) ChangePassword(_ context.Context, cmd usercmd.ChangePasswordCommand) error {
	s.userID = cmd.UserID
	s.passwordCmd = cmd
	return s.err
}
func (s *mockAuthUseCase) ListIdentities(_ context.Context, query userquery.ListIdentitiesQuery) ([]*userresult.IdentityResult, error) {
	s.userID = query.UserID
	return s.identities, s.err
}
func (s *mockAuthUseCase) UnlinkIdentity(_ context.Context, cmd usercmd.UnlinkIdentityCommand) error {
	s.userID = cmd.UserID
	s.unlinkProvider = cmd.Provider
	return s.err
}

func TestAuthHandlerRegister(t *testing.T) {
	gin.SetMode(gin.TestMode)
	usecase := &mockAuthUseCase{authOut: testAuthResult()}
	handler := NewAuthHandler(usecase)
	router := gin.New()
	router.POST("/auth/register", handler.Register)

	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/auth/register", strings.NewReader(`{"email":"alice@example.com","password":"password-123","displayName":"Alice"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	if usecase.registerCmd.Email != "alice@example.com" || usecase.registerCmd.DisplayName != "Alice" {
		t.Fatalf("register command = %+v", usecase.registerCmd)
	}

	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal body: %v", err)
	}
	if body["code"] != float64(0) {
		t.Fatalf("code = %v, want 0", body["code"])
	}
}

func TestAuthHandlerRejectsInvalidRegisterJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewAuthHandler(&mockAuthUseCase{})
	router := gin.New()
	router.POST("/auth/register", handler.Register)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/auth/register", strings.NewReader(`{`)))

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestAuthHandlerEndpoints(t *testing.T) {
	gin.SetMode(gin.TestMode)
	usecase := &mockAuthUseCase{authOut: testAuthResult()}
	handler := NewAuthHandler(usecase)
	router := gin.New()
	router.POST("/auth/login", handler.Login)
	router.POST("/auth/refresh", handler.Refresh)
	router.POST("/auth/logout", handler.Logout)
	router.POST("/auth/password/forgot", handler.ForgotPassword)
	router.POST("/auth/password/reset", handler.ResetPassword)
	router.POST("/auth/email/verify", handler.VerifyEmail)
	router.POST("/auth/oauth/token", handler.OAuthToken)

	tests := []struct {
		name string
		path string
		body string
	}{
		{name: "login", path: "/auth/login", body: `{"email":"alice@example.com","password":"password-123"}`},
		{name: "refresh", path: "/auth/refresh", body: `{"refreshToken":"refresh"}`},
		{name: "logout", path: "/auth/logout", body: `{"refreshToken":"refresh"}`},
		{name: "forgot", path: "/auth/password/forgot", body: `{"email":"alice@example.com"}`},
		{name: "reset", path: "/auth/password/reset", body: `{"token":"reset-token","newPassword":"password-123"}`},
		{name: "verify", path: "/auth/email/verify", body: `{"token":"verify-token"}`},
		{name: "oauthToken", path: "/auth/oauth/token", body: `{"code":"oauth-code"}`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := t.Context()
			// logout 在生产中位于 JWT 中间件之后，handler 依赖 context 中的 claims 取 jti。
			if tc.name == "logout" {
				ctx = auth.WithClaims(ctx, &auth.Claims{UserID: 42})
			}
			req := httptest.NewRequestWithContext(ctx, http.MethodPost, tc.path, strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
			}
		})
	}
	if usecase.loginCmd.Email != "alice@example.com" || usecase.refreshToken != "refresh" {
		t.Fatalf("captured usecase input = %+v", usecase)
	}
}

func TestAuthHandlerRedirectAndClaimsEndpoints(t *testing.T) {
	gin.SetMode(gin.TestMode)
	usecase := &mockAuthUseCase{authOut: testAuthResult()}
	handler := NewAuthHandler(usecase)
	router := gin.New()
	router.GET("/auth/oauth/:provider/start", handler.StartOAuth)
	router.GET("/auth/oauth/:provider/callback", handler.OAuthCallback)
	router.POST("/users/me/email/verification", handler.SendEmailVerification)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/auth/oauth/github/start?redirect_url=https://app.example.com", nil))
	if rec.Code != http.StatusFound || rec.Header().Get("Location") == "" {
		t.Fatalf("start oauth status=%d location=%q", rec.Code, rec.Header().Get("Location"))
	}

	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/auth/oauth/github/callback?state=s&code=c", nil))
	if rec.Code != http.StatusFound || rec.Header().Get("Location") == "" {
		t.Fatalf("callback status=%d location=%q", rec.Code, rec.Header().Get("Location"))
	}

	req := httptest.NewRequestWithContext(auth.WithClaims(t.Context(), &auth.Claims{UserID: 42}), http.MethodPost, "/users/me/email/verification", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || usecase.userID != 42 {
		t.Fatalf("send verification status=%d userID=%d", rec.Code, usecase.userID)
	}
}

func TestUserHandlerEndpoints(t *testing.T) {
	gin.SetMode(gin.TestMode)
	usecase := &mockAuthUseCase{
		userOut: testAuthResult().User,
		identities: []*userresult.IdentityResult{
			{
				Provider:      "github",
				ProviderEmail: "alice@example.com",
				DisplayName:   "Alice",
				ProfileURL:    "https://github.com/alice",
				LinkedAt:      time.Date(2026, 5, 30, 12, 0, 0, 0, time.UTC),
			},
		},
	}
	handler := NewUserHandler(usecase)
	router := gin.New()
	router.GET("/users/me", handler.Me)
	router.PATCH("/users/me", handler.UpdateMe)
	router.PUT("/users/me/password", handler.ChangePassword)
	router.GET("/users/me/identities", handler.ListIdentities)
	router.DELETE("/users/me/identities/:provider", handler.UnlinkIdentity)

	tests := []struct {
		name   string
		method string
		path   string
		body   string
	}{
		{name: "me", method: http.MethodGet, path: "/users/me"},
		{name: "update", method: http.MethodPatch, path: "/users/me", body: `{"displayName":"Alice","avatarUrl":"https://example.com/a.png"}`},
		{name: "password", method: http.MethodPut, path: "/users/me/password", body: `{"oldPassword":"password-123","newPassword":"password-456"}`},
		{name: "identities", method: http.MethodGet, path: "/users/me/identities"},
		{name: "unlink", method: http.MethodDelete, path: "/users/me/identities/github"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := auth.WithClaims(t.Context(), &auth.Claims{UserID: 42})
			req := httptest.NewRequestWithContext(ctx, tc.method, tc.path, strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
			}
		})
	}
	if usecase.userID != 42 || usecase.updateCmd.DisplayName != "Alice" || usecase.passwordCmd.NewPassword != "password-456" || usecase.unlinkProvider != "github" {
		t.Fatalf("captured usecase input = %+v", usecase)
	}
}

func testAuthResult() *userresult.AuthResult {
	now := time.Date(2026, 5, 30, 12, 0, 0, 0, time.UTC)
	return &userresult.AuthResult{
		User: &userresult.UserResult{
			ID:            42,
			Email:         "alice@example.com",
			EmailVerified: true,
			DisplayName:   "Alice",
			Status:        "active",
			CreatedAt:     now,
			UpdatedAt:     now,
		},
		Tokens: userresult.TokenPair{
			AccessToken:           "access",
			AccessTokenExpiresAt:  now.Add(15 * time.Minute),
			RefreshToken:          "refresh",
			RefreshTokenExpiresAt: now.Add(24 * time.Hour),
		},
	}
}
