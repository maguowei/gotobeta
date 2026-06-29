// Package command 定义用户认证写用例入参。
package command

import "time"

// RegisterCommand 是注册命令。
type RegisterCommand struct {
	Email       string
	Password    string
	DisplayName string
}

// LoginCommand 是登录命令。
type LoginCommand struct {
	Email    string
	Password string
}

// RefreshTokenCommand 是刷新会话命令。
type RefreshTokenCommand struct {
	RefreshToken string
}

// LogoutCommand 是退出登录命令。
type LogoutCommand struct {
	RefreshToken string
	// AccessTokenID 是当前 access token 的 jti，用于加入吊销黑名单；为空则跳过。
	AccessTokenID string
	// AccessTokenExpiresAt 是 access token 过期时间，黑名单 TTL 据此推算。
	AccessTokenExpiresAt time.Time
}

// ForgotPasswordCommand 是忘记密码命令。
type ForgotPasswordCommand struct {
	Email string
}

// ResetPasswordCommand 是重置密码命令。
type ResetPasswordCommand struct {
	Token       string
	NewPassword string
}

// VerifyEmailCommand 是邮箱验证命令。
type VerifyEmailCommand struct {
	Token string
}

// UpdateProfileCommand 是更新资料命令。
type UpdateProfileCommand struct {
	UserID      int64
	DisplayName string
	AvatarURL   string
}

// ChangePasswordCommand 是修改密码命令。
type ChangePasswordCommand struct {
	UserID      int64
	OldPassword string
	NewPassword string
}

// StartOAuthCommand 是发起 OAuth 登录命令。
type StartOAuthCommand struct {
	Provider    string
	RedirectURL string
}

// HandleOAuthCallbackCommand 是处理 OAuth 回调命令。
type HandleOAuthCallbackCommand struct {
	Provider string
	State    string
	Code     string
}

// ExchangeOAuthLoginCodeCommand 是 OAuth 登录码换 token 命令。
type ExchangeOAuthLoginCodeCommand struct {
	Code string
}

// UnlinkIdentityCommand 是解绑第三方身份命令。
type UnlinkIdentityCommand struct {
	UserID   int64
	Provider string
}
