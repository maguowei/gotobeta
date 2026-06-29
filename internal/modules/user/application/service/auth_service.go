package service

import (
	"context"
	stderrors "errors"
	"fmt"
	"log/slog"
	"net/url"
	"strings"
	"time"

	userport "github.com/maguowei/gotobeta/internal/modules/user/application/port"
	userresult "github.com/maguowei/gotobeta/internal/modules/user/application/result"
	"github.com/maguowei/gotobeta/internal/modules/user/domain/actiontoken"
	"github.com/maguowei/gotobeta/internal/modules/user/domain/identity"
	"github.com/maguowei/gotobeta/internal/modules/user/domain/oauthstate"
	"github.com/maguowei/gotobeta/internal/modules/user/domain/refreshtoken"
	userdomain "github.com/maguowei/gotobeta/internal/modules/user/domain/user"
	"github.com/maguowei/gotobeta/internal/pkg/apperr"
	"github.com/maguowei/gotobeta/internal/pkg/idgen"
	"github.com/maguowei/gotobeta/internal/pkg/persistence"
)

// Repositories 聚合用户认证用例依赖的各聚合仓储。
type Repositories struct {
	Users         userdomain.Repository
	Identities    identity.Repository
	RefreshTokens refreshtoken.Repository
	ActionTokens  actiontoken.Repository
	OAuthStates   oauthstate.Repository
}

// Config 是认证应用服务配置。
type Config struct {
	RefreshTTL         time.Duration
	EmailTokenTTL      time.Duration
	PasswordResetTTL   time.Duration
	OAuthStateTTL      time.Duration
	OAuthLoginCodeTTL  time.Duration
	SuccessRedirectURL string
}

// AuthService 编排用户认证用例。
type AuthService struct {
	repos          Repositories
	idGenerator    idgen.Generator
	txRunner       persistence.TxRunner
	passwordHasher userport.PasswordHasher
	secrets        userport.SecretGenerator
	tokenIssuer    userport.AccessTokenIssuer
	oauthProviders userport.OAuthProviders
	emailSender    userport.EmailSender
	cfg            Config
	logger         *slog.Logger
	now            func() time.Time
}

// NewAuthService 创建 AuthService。
func NewAuthService(
	repos Repositories,
	idGenerator idgen.Generator,
	txRunner persistence.TxRunner,
	passwordHasher userport.PasswordHasher,
	secrets userport.SecretGenerator,
	tokenIssuer userport.AccessTokenIssuer,
	oauthProviders userport.OAuthProviders,
	emailSender userport.EmailSender,
	cfg Config,
	logger *slog.Logger,
) *AuthService {
	return &AuthService{
		repos:          repos,
		idGenerator:    idGenerator,
		txRunner:       txRunner,
		passwordHasher: passwordHasher,
		secrets:        secrets,
		tokenIssuer:    tokenIssuer,
		oauthProviders: oauthProviders,
		emailSender:    emailSender,
		cfg:            cfg,
		logger:         logger,
		now:            time.Now,
	}
}

func (s *AuthService) loginOAuthProfile(ctx context.Context, profile *oauthstate.Profile, now time.Time) (*userdomain.User, error) {
	var u *userdomain.User
	err := s.txRunner.RunInTx(ctx, func(txCtx context.Context) error {
		ident, err := s.repos.Identities.Find(txCtx, profile.Provider, profile.ProviderUserID)
		if err == nil {
			u, err = s.repos.Users.FindUserByID(txCtx, ident.UserID)
			if err != nil {
				return mapUserLookupError(err)
			}
		} else if stderrors.Is(err, identity.ErrNotFound) {
			normalizedEmail, err := userdomain.NormalizeEmail(profile.Email)
			if err != nil {
				return err
			}
			u, err = s.repos.Users.FindUserByEmail(txCtx, normalizedEmail)
			if stderrors.Is(err, userdomain.ErrNotFound) {
				userID, err := s.idGenerator.NextID(txCtx)
				if err != nil {
					return wrapInfrastructureError("生成用户 ID 失败", err)
				}
				u, err = userdomain.New(userID, profile.Email, profile.DisplayName, now)
				if err != nil {
					return err
				}
				if profile.EmailVerified {
					u.VerifyEmail(now)
				}
				if err := s.repos.Users.CreateUser(txCtx, u); err != nil {
					return wrapInfrastructureError("保存 OAuth 用户失败", err)
				}
			} else if err != nil {
				return wrapInfrastructureError("查询 OAuth 邮箱用户失败", err)
			} else if !profile.EmailVerified {
				return apperr.Unauthorized("OAuth provider 邮箱未验证，不能绑定已有账号")
			}
		} else {
			return wrapInfrastructureError("查询第三方身份失败", err)
		}

		identityID, err := s.idGenerator.NextID(txCtx)
		if err != nil {
			return wrapInfrastructureError("生成第三方身份 ID 失败", err)
		}
		normalizedProviderEmail, _ := userdomain.NormalizeEmail(profile.Email)
		if err := s.repos.Identities.Upsert(txCtx, &identity.Identity{
			ID:                      identityID,
			UserID:                  u.ID(),
			Provider:                profile.Provider,
			ProviderUserID:          profile.ProviderUserID,
			ProviderEmail:           profile.Email,
			ProviderEmailNormalized: normalizedProviderEmail,
			ProviderEmailVerified:   profile.EmailVerified,
			DisplayName:             profile.DisplayName,
			AvatarURL:               profile.AvatarURL,
			ProfileURL:              profile.ProfileURL,
			LinkedAt:                now,
			LastLoginAt:             &now,
			CreatedAt:               now,
			UpdatedAt:               now,
		}); err != nil {
			return wrapInfrastructureError("保存第三方身份失败", err)
		}
		u.TouchLogin(now)
		return s.repos.Users.UpdateUserLastLogin(txCtx, u.ID(), now)
	})
	return u, err
}

func (s *AuthService) issueSession(ctx context.Context, u *userdomain.User, now time.Time) (userresult.TokenPair, error) {
	tokens, _, err := s.issueSessionWithID(ctx, u, now)
	return tokens, err
}

func (s *AuthService) issueSessionWithID(ctx context.Context, u *userdomain.User, now time.Time) (userresult.TokenPair, string, error) {
	access, accessExpiresAt, err := s.tokenIssuer.IssueAccessToken(u, now)
	if err != nil {
		return userresult.TokenPair{}, "", wrapInfrastructureError("签发 access token 失败", err)
	}
	refresh, err := s.secrets.NewToken()
	if err != nil {
		return userresult.TokenPair{}, "", wrapInfrastructureError("生成 refresh token 失败", err)
	}
	tokenID, err := s.secrets.NewToken()
	if err != nil {
		return userresult.TokenPair{}, "", wrapInfrastructureError("生成 refresh token ID 失败", err)
	}
	refreshExpiresAt := now.Add(s.cfg.RefreshTTL)
	refreshToken, err := refreshtoken.New(tokenID, u.ID(), s.secrets.HashToken(refresh), refreshExpiresAt, now)
	if err != nil {
		return userresult.TokenPair{}, "", err
	}
	if err := s.repos.RefreshTokens.Create(ctx, refreshToken); err != nil {
		return userresult.TokenPair{}, "", wrapInfrastructureError("保存 refresh token 失败", err)
	}
	return userresult.TokenPair{
		AccessToken:           access,
		AccessTokenExpiresAt:  accessExpiresAt,
		RefreshToken:          refresh,
		RefreshTokenExpiresAt: refreshExpiresAt,
	}, tokenID, nil
}

func (s *AuthService) createActionToken(ctx context.Context, userID int64, purpose string, targetEmail string, ttl time.Duration) (string, error) {
	raw, err := s.secrets.NewToken()
	if err != nil {
		return "", wrapInfrastructureError("生成动作 token 失败", err)
	}
	tokenID, err := s.secrets.NewToken()
	if err != nil {
		return "", wrapInfrastructureError("生成动作 token ID 失败", err)
	}
	now := s.now()
	token, err := actiontoken.New(tokenID, userID, purpose, s.secrets.HashToken(raw), targetEmail, now.Add(ttl), now)
	if err != nil {
		return "", err
	}
	if err := s.repos.ActionTokens.Create(ctx, token); err != nil {
		return "", wrapInfrastructureError("保存动作 token 失败", err)
	}
	return raw, nil
}

func toUserResult(u *userdomain.User) *userresult.UserResult {
	return &userresult.UserResult{
		ID:            u.ID(),
		Email:         u.Email(),
		EmailVerified: u.EmailVerifiedAt() != nil,
		DisplayName:   u.DisplayName(),
		AvatarURL:     u.AvatarURL(),
		Status:        string(u.Status()),
		CreatedAt:     u.CreatedAt(),
		UpdatedAt:     u.UpdatedAt(),
	}
}

func validatePassword(password string) error {
	if len(strings.TrimSpace(password)) < 8 {
		return apperr.InvalidParam("密码长度不能小于 8")
	}
	return nil
}

func allowedOAuthRedirectURL(requested string, configured string) (string, error) {
	allowed, err := normalizeOAuthRedirectURL(configured)
	if err != nil {
		return "", apperr.Internal("OAuth redirect_url 配置无效", err)
	}
	if strings.TrimSpace(requested) == "" {
		return allowed, nil
	}
	candidate, err := normalizeOAuthRedirectURL(requested)
	if err != nil || candidate != allowed {
		return "", apperr.InvalidParam("OAuth redirect_url 不被允许")
	}
	return candidate, nil
}

func normalizeOAuthRedirectURL(value string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(value))
	if err != nil {
		return "", err
	}
	if (parsed.Scheme != "https" && parsed.Scheme != "http") || parsed.Host == "" || parsed.User != nil || parsed.Fragment != "" {
		return "", fmt.Errorf("invalid redirect URL")
	}
	return parsed.String(), nil
}

func mapUserLookupError(err error) error {
	if stderrors.Is(err, userdomain.ErrNotFound) {
		return apperr.NotFound("用户不存在")
	}
	return wrapInfrastructureError("查询用户失败", err)
}

func wrapInfrastructureError(message string, err error) error {
	if stderrors.Is(err, context.Canceled) || stderrors.Is(err, context.DeadlineExceeded) {
		return err
	}
	return apperr.Internal(message, err)
}

func appendLoginCode(base string, code string) string {
	separator := "?"
	if strings.Contains(base, "?") {
		separator = "&"
	}
	return fmt.Sprintf("%s%scode=%s", base, separator, code)
}
