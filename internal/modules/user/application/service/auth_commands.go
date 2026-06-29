package service

import (
	"context"
	stderrors "errors"
	"log/slog"

	usercmd "github.com/maguowei/gotobeta/internal/modules/user/application/command"
	userresult "github.com/maguowei/gotobeta/internal/modules/user/application/result"
	"github.com/maguowei/gotobeta/internal/modules/user/domain/actiontoken"
	"github.com/maguowei/gotobeta/internal/modules/user/domain/oauthstate"
	"github.com/maguowei/gotobeta/internal/modules/user/domain/refreshtoken"
	userdomain "github.com/maguowei/gotobeta/internal/modules/user/domain/user"
	"github.com/maguowei/gotobeta/internal/pkg/apperr"
	loggerx "github.com/maguowei/gotobeta/internal/pkg/logger"
)

// Register 注册邮箱用户并签发会话。
func (s *AuthService) Register(ctx context.Context, cmd usercmd.RegisterCommand) (*userresult.AuthResult, error) {
	now := s.now()
	normalizedEmail, err := userdomain.NormalizeEmail(cmd.Email)
	if err != nil {
		return nil, err
	}
	if err := validatePassword(cmd.Password); err != nil {
		return nil, err
	}

	var out *userresult.AuthResult
	err = s.txRunner.RunInTx(ctx, func(txCtx context.Context) error {
		if _, err := s.repos.Users.FindUserByEmail(txCtx, normalizedEmail); err == nil {
			return apperr.Conflict("邮箱已注册")
		} else if !stderrors.Is(err, userdomain.ErrNotFound) {
			return wrapInfrastructureError("查询用户失败", err)
		}

		userID, err := s.idGenerator.NextID(txCtx)
		if err != nil {
			return wrapInfrastructureError("生成用户 ID 失败", err)
		}
		u, err := userdomain.New(userID, cmd.Email, cmd.DisplayName, now)
		if err != nil {
			return err
		}
		hash, alg, err := s.passwordHasher.Hash(cmd.Password)
		if err != nil {
			return wrapInfrastructureError("生成密码哈希失败", err)
		}
		if err := u.SetPassword(hash, alg, now); err != nil {
			return err
		}
		if err := s.repos.Users.CreateUser(txCtx, u); err != nil {
			return wrapInfrastructureError("保存用户失败", err)
		}
		if _, err := s.createActionToken(txCtx, u.ID(), actiontoken.ActionEmailVerification, normalizedEmail, s.cfg.EmailTokenTTL); err != nil {
			return err
		}
		tokens, err := s.issueSession(txCtx, u, now)
		if err != nil {
			return err
		}
		out = &userresult.AuthResult{User: toUserResult(u), Tokens: tokens}
		return nil
	})
	if err != nil {
		loggerx.WithError(ctx, s.logger, "register user failed", err, slog.String("email", normalizedEmail))
		return nil, err
	}
	return out, nil
}

// Login 使用邮箱密码登录。
func (s *AuthService) Login(ctx context.Context, cmd usercmd.LoginCommand) (*userresult.AuthResult, error) {
	now := s.now()
	normalizedEmail, err := userdomain.NormalizeEmail(cmd.Email)
	if err != nil {
		return nil, err
	}
	u, err := s.repos.Users.FindUserByEmail(ctx, normalizedEmail)
	if err != nil {
		if stderrors.Is(err, userdomain.ErrNotFound) {
			return nil, apperr.Unauthorized("邮箱或密码不正确")
		}
		return nil, wrapInfrastructureError("查询用户失败", err)
	}
	if err := u.EnsureCanLogin(); err != nil {
		return nil, err
	}
	if !u.HasPassword() || s.passwordHasher.Compare(u.PasswordHash(), cmd.Password) != nil {
		return nil, apperr.Unauthorized("邮箱或密码不正确")
	}

	var out *userresult.AuthResult
	err = s.txRunner.RunInTx(ctx, func(txCtx context.Context) error {
		u.TouchLogin(now)
		if err := s.repos.Users.UpdateUserLastLogin(txCtx, u.ID(), now); err != nil {
			return wrapInfrastructureError("更新登录时间失败", err)
		}
		tokens, err := s.issueSession(txCtx, u, now)
		if err != nil {
			return err
		}
		out = &userresult.AuthResult{User: toUserResult(u), Tokens: tokens}
		return nil
	})
	if err != nil {
		loggerx.WithError(ctx, s.logger, "login failed", err, slog.String("email", normalizedEmail))
		return nil, err
	}
	return out, nil
}

// Refresh 刷新并轮换 refresh token。
func (s *AuthService) Refresh(ctx context.Context, cmd usercmd.RefreshTokenCommand) (*userresult.AuthResult, error) {
	now := s.now()
	tokenHash := s.secrets.HashToken(cmd.RefreshToken)
	var out *userresult.AuthResult
	err := s.txRunner.RunInTx(ctx, func(txCtx context.Context) error {
		oldToken, err := s.repos.RefreshTokens.FindByHash(txCtx, tokenHash, now)
		if err != nil {
			return apperr.Unauthorized("refresh token 无效")
		}
		u, err := s.repos.Users.FindUserByID(txCtx, oldToken.UserID)
		if err != nil {
			return wrapInfrastructureError("查询用户失败", err)
		}
		if err := u.EnsureCanLogin(); err != nil {
			return err
		}
		tokens, newTokenID, err := s.issueSessionWithID(txCtx, u, now)
		if err != nil {
			return err
		}
		if err := s.repos.RefreshTokens.Revoke(txCtx, oldToken.TokenID, newTokenID, "rotated", now); err != nil {
			if stderrors.Is(err, refreshtoken.ErrNotFound) {
				return apperr.Unauthorized("refresh token 无效")
			}
			return wrapInfrastructureError("轮换 refresh token 失败", err)
		}
		out = &userresult.AuthResult{User: toUserResult(u), Tokens: tokens}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// Logout 撤销 refresh token。
func (s *AuthService) Logout(ctx context.Context, cmd usercmd.LogoutCommand) error {
	token, err := s.repos.RefreshTokens.FindByHash(ctx, s.secrets.HashToken(cmd.RefreshToken), s.now())
	if err != nil {
		return nil
	}
	return s.repos.RefreshTokens.Revoke(ctx, token.TokenID, "", "logout", s.now())
}

// UpdateProfile 更新当前用户资料。
func (s *AuthService) UpdateProfile(ctx context.Context, cmd usercmd.UpdateProfileCommand) (*userresult.UserResult, error) {
	now := s.now()
	u, err := s.repos.Users.FindUserByID(ctx, cmd.UserID)
	if err != nil {
		return nil, mapUserLookupError(err)
	}
	u.UpdateProfile(cmd.DisplayName, cmd.AvatarURL, now)
	if err := s.repos.Users.SaveUser(ctx, u); err != nil {
		return nil, wrapInfrastructureError("更新用户资料失败", err)
	}
	return toUserResult(u), nil
}

// ChangePassword 修改当前用户密码。
func (s *AuthService) ChangePassword(ctx context.Context, cmd usercmd.ChangePasswordCommand) error {
	if err := validatePassword(cmd.NewPassword); err != nil {
		return err
	}
	u, err := s.repos.Users.FindUserByID(ctx, cmd.UserID)
	if err != nil {
		return mapUserLookupError(err)
	}
	if u.HasPassword() && s.passwordHasher.Compare(u.PasswordHash(), cmd.OldPassword) != nil {
		return apperr.Unauthorized("原密码不正确")
	}
	hash, alg, err := s.passwordHasher.Hash(cmd.NewPassword)
	if err != nil {
		return wrapInfrastructureError("生成密码哈希失败", err)
	}
	if err := u.SetPassword(hash, alg, s.now()); err != nil {
		return err
	}
	return s.repos.Users.SaveUser(ctx, u)
}

// ForgotPassword 创建密码重置 token 并发送开发邮件。
func (s *AuthService) ForgotPassword(ctx context.Context, cmd usercmd.ForgotPasswordCommand) error {
	normalizedEmail, err := userdomain.NormalizeEmail(cmd.Email)
	if err != nil {
		return err
	}
	u, err := s.repos.Users.FindUserByEmail(ctx, normalizedEmail)
	if err != nil {
		if stderrors.Is(err, userdomain.ErrNotFound) {
			return nil
		}
		return wrapInfrastructureError("查询用户失败", err)
	}
	raw, err := s.createActionToken(ctx, u.ID(), actiontoken.ActionPasswordReset, normalizedEmail, s.cfg.PasswordResetTTL)
	if err != nil {
		return err
	}
	return s.emailSender.SendPasswordReset(ctx, u.Email(), raw)
}

// ResetPassword 使用一次性 token 重置密码。
func (s *AuthService) ResetPassword(ctx context.Context, cmd usercmd.ResetPasswordCommand) error {
	if err := validatePassword(cmd.NewPassword); err != nil {
		return err
	}
	now := s.now()
	return s.txRunner.RunInTx(ctx, func(txCtx context.Context) error {
		action, err := s.repos.ActionTokens.Consume(txCtx, s.secrets.HashToken(cmd.Token), actiontoken.ActionPasswordReset, now)
		if err != nil {
			return apperr.InvalidParam("密码重置 token 无效或已过期")
		}
		u, err := s.repos.Users.FindUserByID(txCtx, action.UserID)
		if err != nil {
			return mapUserLookupError(err)
		}
		hash, alg, err := s.passwordHasher.Hash(cmd.NewPassword)
		if err != nil {
			return wrapInfrastructureError("生成密码哈希失败", err)
		}
		if err := u.SetPassword(hash, alg, now); err != nil {
			return err
		}
		return s.repos.Users.SaveUser(txCtx, u)
	})
}

// SendEmailVerification 重新发送邮箱验证 token。
func (s *AuthService) SendEmailVerification(ctx context.Context, userID int64) error {
	u, err := s.repos.Users.FindUserByID(ctx, userID)
	if err != nil {
		return mapUserLookupError(err)
	}
	raw, err := s.createActionToken(ctx, u.ID(), actiontoken.ActionEmailVerification, u.EmailNormalized(), s.cfg.EmailTokenTTL)
	if err != nil {
		return err
	}
	return s.emailSender.SendEmailVerification(ctx, u.Email(), raw)
}

// VerifyEmail 使用一次性 token 验证邮箱。
func (s *AuthService) VerifyEmail(ctx context.Context, cmd usercmd.VerifyEmailCommand) error {
	now := s.now()
	return s.txRunner.RunInTx(ctx, func(txCtx context.Context) error {
		action, err := s.repos.ActionTokens.Consume(txCtx, s.secrets.HashToken(cmd.Token), actiontoken.ActionEmailVerification, now)
		if err != nil {
			return apperr.InvalidParam("邮箱验证 token 无效或已过期")
		}
		u, err := s.repos.Users.FindUserByID(txCtx, action.UserID)
		if err != nil {
			return mapUserLookupError(err)
		}
		u.VerifyEmail(now)
		return s.repos.Users.SaveUser(txCtx, u)
	})
}

// StartOAuth 创建 OAuth state 并返回第三方授权 URL。
func (s *AuthService) StartOAuth(ctx context.Context, cmd usercmd.StartOAuthCommand) (*userresult.OAuthStartResult, error) {
	adapter, ok := s.oauthProviders.Get(cmd.Provider)
	if !ok {
		return nil, apperr.InvalidParam("不支持的 OAuth provider")
	}
	redirectURL, err := allowedOAuthRedirectURL(cmd.RedirectURL, s.cfg.SuccessRedirectURL)
	if err != nil {
		return nil, err
	}
	state, err := s.secrets.NewToken()
	if err != nil {
		return nil, wrapInfrastructureError("生成 OAuth state 失败", err)
	}
	now := s.now()
	if err := s.repos.OAuthStates.Create(ctx, &oauthstate.OAuthState{
		StateHash:   s.secrets.HashToken(state),
		Provider:    cmd.Provider,
		RedirectURL: redirectURL,
		ExpiresAt:   now.Add(s.cfg.OAuthStateTTL),
		CreatedAt:   now,
		UpdatedAt:   now,
	}); err != nil {
		return nil, wrapInfrastructureError("保存 OAuth state 失败", err)
	}
	return &userresult.OAuthStartResult{AuthURL: adapter.AuthCodeURL(state), State: state}, nil
}

// HandleOAuthCallback 处理 OAuth callback，返回前端重定向地址和一次性登录码。
func (s *AuthService) HandleOAuthCallback(ctx context.Context, cmd usercmd.HandleOAuthCallbackCommand) (*userresult.OAuthCallbackResult, error) {
	adapter, ok := s.oauthProviders.Get(cmd.Provider)
	if !ok {
		return nil, apperr.InvalidParam("不支持的 OAuth provider")
	}
	now := s.now()
	oauthState, err := s.repos.OAuthStates.Consume(ctx, cmd.Provider, s.secrets.HashToken(cmd.State), now)
	if err != nil {
		return nil, apperr.InvalidParam("OAuth state 无效或已过期")
	}
	profile, err := adapter.Exchange(ctx, cmd.Code)
	if err != nil {
		return nil, wrapInfrastructureError("OAuth 换取用户资料失败", err)
	}
	u, err := s.loginOAuthProfile(ctx, profile, now)
	if err != nil {
		return nil, err
	}
	loginCode, err := s.createActionToken(ctx, u.ID(), actiontoken.ActionOAuthLoginCode, u.EmailNormalized(), s.cfg.OAuthLoginCodeTTL)
	if err != nil {
		return nil, err
	}
	return &userresult.OAuthCallbackResult{
		RedirectURL: appendLoginCode(oauthState.RedirectURL, loginCode),
		LoginCode:   loginCode,
	}, nil
}

// ExchangeOAuthLoginCode 把 OAuth 登录码换成 token。
func (s *AuthService) ExchangeOAuthLoginCode(ctx context.Context, cmd usercmd.ExchangeOAuthLoginCodeCommand) (*userresult.AuthResult, error) {
	now := s.now()
	var out *userresult.AuthResult
	err := s.txRunner.RunInTx(ctx, func(txCtx context.Context) error {
		action, err := s.repos.ActionTokens.Consume(txCtx, s.secrets.HashToken(cmd.Code), actiontoken.ActionOAuthLoginCode, now)
		if err != nil {
			return apperr.InvalidParam("OAuth 登录码无效或已过期")
		}
		u, err := s.repos.Users.FindUserByID(txCtx, action.UserID)
		if err != nil {
			return mapUserLookupError(err)
		}
		tokens, err := s.issueSession(txCtx, u, now)
		if err != nil {
			return err
		}
		out = &userresult.AuthResult{User: toUserResult(u), Tokens: tokens}
		return nil
	})
	return out, err
}

// UnlinkIdentity 解绑第三方身份，保留至少一种登录方式。
func (s *AuthService) UnlinkIdentity(ctx context.Context, cmd usercmd.UnlinkIdentityCommand) error {
	methods, err := s.repos.Users.CountLoginMethods(ctx, cmd.UserID)
	if err != nil {
		return wrapInfrastructureError("查询登录方式失败", err)
	}
	if methods <= 1 {
		return apperr.Conflict("至少保留一种登录方式")
	}
	return s.repos.Identities.Delete(ctx, cmd.UserID, cmd.Provider)
}
