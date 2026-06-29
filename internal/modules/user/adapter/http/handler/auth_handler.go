package handler

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	userreq "github.com/maguowei/gotobeta/internal/modules/user/adapter/http/request"
	userresp "github.com/maguowei/gotobeta/internal/modules/user/adapter/http/response"
	usercmd "github.com/maguowei/gotobeta/internal/modules/user/application/command"
	userresult "github.com/maguowei/gotobeta/internal/modules/user/application/result"
	httpmiddleware "github.com/maguowei/gotobeta/internal/pkg/httpx/middleware"
	httpresponse "github.com/maguowei/gotobeta/internal/pkg/httpx/response"
)

// AuthUseCase 定义认证 handler 依赖。
type AuthUseCase interface {
	Register(ctx context.Context, cmd usercmd.RegisterCommand) (*userresult.AuthResult, error)
	Login(ctx context.Context, cmd usercmd.LoginCommand) (*userresult.AuthResult, error)
	Refresh(ctx context.Context, cmd usercmd.RefreshTokenCommand) (*userresult.AuthResult, error)
	Logout(ctx context.Context, cmd usercmd.LogoutCommand) error
	ForgotPassword(ctx context.Context, cmd usercmd.ForgotPasswordCommand) error
	ResetPassword(ctx context.Context, cmd usercmd.ResetPasswordCommand) error
	SendEmailVerification(ctx context.Context, userID int64) error
	VerifyEmail(ctx context.Context, cmd usercmd.VerifyEmailCommand) error
	StartOAuth(ctx context.Context, cmd usercmd.StartOAuthCommand) (*userresult.OAuthStartResult, error)
	HandleOAuthCallback(ctx context.Context, cmd usercmd.HandleOAuthCallbackCommand) (*userresult.OAuthCallbackResult, error)
	ExchangeOAuthLoginCode(ctx context.Context, cmd usercmd.ExchangeOAuthLoginCodeCommand) (*userresult.AuthResult, error)
}

// AuthHandler 处理认证 HTTP 请求。
type AuthHandler struct {
	usecase AuthUseCase
}

// NewAuthHandler 创建 AuthHandler。
func NewAuthHandler(usecase AuthUseCase) *AuthHandler {
	return &AuthHandler{usecase: usecase}
}

// Register 注册。
func (h *AuthHandler) Register(c *gin.Context) {
	var req userreq.RegisterRequest
	if !bindJSON(c, &req) {
		return
	}
	out, err := h.usecase.Register(c.Request.Context(), req.ToCommand())
	if err != nil {
		httpresponse.Error(c, err)
		return
	}
	httpresponse.Success(c, userresp.ToAuthResponse(out))
}

// Login 登录。
func (h *AuthHandler) Login(c *gin.Context) {
	var req userreq.LoginRequest
	if !bindJSON(c, &req) {
		return
	}
	out, err := h.usecase.Login(c.Request.Context(), req.ToCommand())
	if err != nil {
		httpresponse.Error(c, err)
		return
	}
	httpresponse.Success(c, userresp.ToAuthResponse(out))
}

// Refresh 刷新 token。
func (h *AuthHandler) Refresh(c *gin.Context) {
	var req userreq.RefreshRequest
	if !bindJSON(c, &req) {
		return
	}
	out, err := h.usecase.Refresh(c.Request.Context(), req.ToCommand())
	if err != nil {
		httpresponse.Error(c, err)
		return
	}
	httpresponse.Success(c, userresp.ToAuthResponse(out))
}

// Logout 退出登录。
func (h *AuthHandler) Logout(c *gin.Context) {
	var req userreq.LogoutRequest
	if !bindJSON(c, &req) {
		return
	}
	if err := h.usecase.Logout(c.Request.Context(), req.ToCommand()); err != nil {
		httpresponse.Error(c, err)
		return
	}
	httpresponse.Success(c, nil)
}

// ForgotPassword 发送密码重置邮件。
func (h *AuthHandler) ForgotPassword(c *gin.Context) {
	var req userreq.ForgotPasswordRequest
	if !bindJSON(c, &req) {
		return
	}
	if err := h.usecase.ForgotPassword(c.Request.Context(), req.ToCommand()); err != nil {
		httpresponse.Error(c, err)
		return
	}
	httpresponse.Success(c, nil)
}

// ResetPassword 重置密码。
func (h *AuthHandler) ResetPassword(c *gin.Context) {
	var req userreq.ResetPasswordRequest
	if !bindJSON(c, &req) {
		return
	}
	if err := h.usecase.ResetPassword(c.Request.Context(), req.ToCommand()); err != nil {
		httpresponse.Error(c, err)
		return
	}
	httpresponse.Success(c, nil)
}

// SendEmailVerification 发送邮箱验证邮件。
func (h *AuthHandler) SendEmailVerification(c *gin.Context) {
	claims, ok := httpmiddleware.RequireClaims(c)
	if !ok {
		return
	}
	if err := h.usecase.SendEmailVerification(c.Request.Context(), claims.UserID); err != nil {
		httpresponse.Error(c, err)
		return
	}
	httpresponse.Success(c, nil)
}

// VerifyEmail 验证邮箱。
func (h *AuthHandler) VerifyEmail(c *gin.Context) {
	var req userreq.VerifyEmailRequest
	if !bindJSON(c, &req) {
		return
	}
	if err := h.usecase.VerifyEmail(c.Request.Context(), req.ToCommand()); err != nil {
		httpresponse.Error(c, err)
		return
	}
	httpresponse.Success(c, nil)
}

// StartOAuth 跳转到三方授权页。
func (h *AuthHandler) StartOAuth(c *gin.Context) {
	out, err := h.usecase.StartOAuth(c.Request.Context(), usercmd.StartOAuthCommand{
		Provider:    c.Param("provider"),
		RedirectURL: c.Query("redirect_url"),
	})
	if err != nil {
		httpresponse.Error(c, err)
		return
	}
	c.Redirect(http.StatusFound, out.AuthURL)
}

// OAuthCallback 处理三方回调。
func (h *AuthHandler) OAuthCallback(c *gin.Context) {
	out, err := h.usecase.HandleOAuthCallback(c.Request.Context(), usercmd.HandleOAuthCallbackCommand{
		Provider: c.Param("provider"),
		State:    c.Query("state"),
		Code:     c.Query("code"),
	})
	if err != nil {
		httpresponse.Error(c, err)
		return
	}
	c.Redirect(http.StatusFound, out.RedirectURL)
}

// OAuthToken 使用一次性登录码换 token。
func (h *AuthHandler) OAuthToken(c *gin.Context) {
	var req userreq.OAuthTokenRequest
	if !bindJSON(c, &req) {
		return
	}
	out, err := h.usecase.ExchangeOAuthLoginCode(c.Request.Context(), req.ToCommand())
	if err != nil {
		httpresponse.Error(c, err)
		return
	}
	httpresponse.Success(c, userresp.ToAuthResponse(out))
}

func bindJSON(c *gin.Context, target any) bool {
	if err := c.ShouldBindJSON(target); err != nil {
		httpresponse.ErrorWithCode(c, httpresponse.CodeInvalidParam, "请求参数格式错误")
		return false
	}
	return true
}
