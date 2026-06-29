package request

import usercmd "github.com/maguowei/gotobeta/internal/modules/user/application/command"

// RegisterRequest 是注册请求。
type RegisterRequest struct {
	Email       string `json:"email" binding:"required,email"`
	Password    string `json:"password" binding:"required,min=8"`
	DisplayName string `json:"displayName"`
}

// ToCommand 转换为注册命令。
func (r RegisterRequest) ToCommand() usercmd.RegisterCommand {
	return usercmd.RegisterCommand{Email: r.Email, Password: r.Password, DisplayName: r.DisplayName}
}

// LoginRequest 是登录请求。
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// ToCommand 转换为登录命令。
func (r LoginRequest) ToCommand() usercmd.LoginCommand {
	return usercmd.LoginCommand{Email: r.Email, Password: r.Password}
}

// RefreshRequest 是刷新 token 请求。
type RefreshRequest struct {
	RefreshToken string `json:"refreshToken" binding:"required"`
}

// ToCommand 转换为刷新会话命令。
func (r RefreshRequest) ToCommand() usercmd.RefreshTokenCommand {
	return usercmd.RefreshTokenCommand{RefreshToken: r.RefreshToken}
}

// LogoutRequest 是退出请求。
type LogoutRequest struct {
	RefreshToken string `json:"refreshToken" binding:"required"`
}

// ToCommand 转换为退出登录命令。
func (r LogoutRequest) ToCommand() usercmd.LogoutCommand {
	return usercmd.LogoutCommand{RefreshToken: r.RefreshToken}
}

// ForgotPasswordRequest 是忘记密码请求。
type ForgotPasswordRequest struct {
	Email string `json:"email" binding:"required,email"`
}

// ToCommand 转换为忘记密码命令。
func (r ForgotPasswordRequest) ToCommand() usercmd.ForgotPasswordCommand {
	return usercmd.ForgotPasswordCommand{Email: r.Email}
}

// ResetPasswordRequest 是重置密码请求。
type ResetPasswordRequest struct {
	Token       string `json:"token" binding:"required"`
	NewPassword string `json:"newPassword" binding:"required,min=8"`
}

// ToCommand 转换为重置密码命令。
func (r ResetPasswordRequest) ToCommand() usercmd.ResetPasswordCommand {
	return usercmd.ResetPasswordCommand{Token: r.Token, NewPassword: r.NewPassword}
}

// VerifyEmailRequest 是邮箱验证请求。
type VerifyEmailRequest struct {
	Token string `json:"token" binding:"required"`
}

// ToCommand 转换为邮箱验证命令。
func (r VerifyEmailRequest) ToCommand() usercmd.VerifyEmailCommand {
	return usercmd.VerifyEmailCommand{Token: r.Token}
}

// OAuthTokenRequest 是 OAuth 登录码换 token 请求。
type OAuthTokenRequest struct {
	Code string `json:"code" binding:"required"`
}

// ToCommand 转换为 OAuth 登录码换 token 命令。
func (r OAuthTokenRequest) ToCommand() usercmd.ExchangeOAuthLoginCodeCommand {
	return usercmd.ExchangeOAuthLoginCodeCommand{Code: r.Code}
}

// UpdateProfileRequest 是更新资料请求。
type UpdateProfileRequest struct {
	DisplayName string `json:"displayName"`
	AvatarURL   string `json:"avatarUrl"`
}

// ToCommand 转换为更新资料命令。
func (r UpdateProfileRequest) ToCommand(userID int64) usercmd.UpdateProfileCommand {
	return usercmd.UpdateProfileCommand{UserID: userID, DisplayName: r.DisplayName, AvatarURL: r.AvatarURL}
}

// ChangePasswordRequest 是修改密码请求。
type ChangePasswordRequest struct {
	OldPassword string `json:"oldPassword"`
	NewPassword string `json:"newPassword" binding:"required,min=8"`
}

// ToCommand 转换为修改密码命令。
func (r ChangePasswordRequest) ToCommand(userID int64) usercmd.ChangePasswordCommand {
	return usercmd.ChangePasswordCommand{UserID: userID, OldPassword: r.OldPassword, NewPassword: r.NewPassword}
}
