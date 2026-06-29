package user

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/maguowei/gotobeta/internal/ent"
	"github.com/maguowei/gotobeta/internal/infra/cache"
	"github.com/maguowei/gotobeta/internal/infra/config"
	"github.com/maguowei/gotobeta/internal/infra/entdb"
	"github.com/maguowei/gotobeta/internal/infra/localid"
	userhandler "github.com/maguowei/gotobeta/internal/modules/user/adapter/http/handler"
	userrouter "github.com/maguowei/gotobeta/internal/modules/user/adapter/http/router"
	userport "github.com/maguowei/gotobeta/internal/modules/user/application/port"
	usersvc "github.com/maguowei/gotobeta/internal/modules/user/application/service"
	useremail "github.com/maguowei/gotobeta/internal/modules/user/infra/email"
	useroauth "github.com/maguowei/gotobeta/internal/modules/user/infra/oauth"
	userpersist "github.com/maguowei/gotobeta/internal/modules/user/infra/persistence"
	usersecurity "github.com/maguowei/gotobeta/internal/modules/user/infra/security"
	"github.com/maguowei/gotobeta/internal/pkg/auth"
	httpmiddleware "github.com/maguowei/gotobeta/internal/pkg/httpx/middleware"
)

// Module 持有装配好的 User HTTP 入口。
type Module struct {
	authHandler    *userhandler.AuthHandler
	userHandler    *userhandler.UserHandler
	authMiddleware gin.HandlerFunc
	rateLimit      gin.HandlerFunc
}

// New 完成 User 模块的全部装配（repo -> service -> handler + middleware）。
// kv 为 nil（Redis 未启用）时 JWT 吊销黑名单降级为不可用。
func New(client *ent.Client, logger *slog.Logger, cfg *config.Config, kv *cache.RedisKV) (*Module, error) {
	accessTTL, err := time.ParseDuration(cfg.Auth.JWT.AccessTTL)
	if err != nil {
		return nil, err
	}
	refreshTTL, err := time.ParseDuration(cfg.Auth.JWT.RefreshTTL)
	if err != nil {
		return nil, err
	}
	emailTTL, err := time.ParseDuration(cfg.Auth.Email.VerificationTTL)
	if err != nil {
		return nil, err
	}
	passwordResetTTL, err := time.ParseDuration(cfg.Auth.Email.PasswordResetTTL)
	if err != nil {
		return nil, err
	}
	oauthStateTTL, err := time.ParseDuration(cfg.Auth.OAuth.StateTTL)
	if err != nil {
		return nil, err
	}
	oauthLoginCodeTTL, err := time.ParseDuration(cfg.Auth.OAuth.LoginCodeTTL)
	if err != nil {
		return nil, err
	}

	// JWT 吊销黑名单：Redis 启用时构造，否则保持 nil 接口（吊销降级，logout 仅撤 refresh）。
	var tokenRevoker userport.TokenRevoker
	var jwtRevoker auth.RevocationChecker
	if kv != nil {
		blacklist := usersecurity.NewTokenBlacklist(kv)
		tokenRevoker = blacklist
		jwtRevoker = blacklist
	}

	secrets := usersecurity.NewRandomSecretGenerator()
	svc := usersvc.NewAuthService(
		usersvc.Repositories{
			Users:         userpersist.NewUserRepository(client),
			Identities:    userpersist.NewIdentityRepository(client),
			RefreshTokens: userpersist.NewRefreshTokenRepository(client),
			ActionTokens:  userpersist.NewActionTokenRepository(client),
			OAuthStates:   userpersist.NewOAuthStateRepository(client),
		},
		localid.New(),
		entdb.NewEntTxRunner(client),
		usersecurity.NewBcryptPasswordHasher(0),
		secrets,
		usersecurity.NewJWTIssuer(cfg.Auth.JWT, accessTTL),
		tokenRevoker,
		useroauth.NewRegistry(cfg.Auth.OAuth),
		useremail.NewSender(cfg.Auth.Email.Sender, logger),
		usersvc.Config{
			RefreshTTL:         refreshTTL,
			EmailTokenTTL:      emailTTL,
			PasswordResetTTL:   passwordResetTTL,
			OAuthStateTTL:      oauthStateTTL,
			OAuthLoginCodeTTL:  oauthLoginCodeTTL,
			SuccessRedirectURL: cfg.Auth.OAuth.SuccessRedirectURL,
		},
		logger,
	)

	return &Module{
		authHandler: userhandler.NewAuthHandler(svc),
		userHandler: userhandler.NewUserHandler(svc),
		authMiddleware: httpmiddleware.AuthJWT(httpmiddleware.AuthJWTOptions{
			Enabled:    cfg.Auth.JWT.Enabled,
			Issuer:     cfg.Auth.JWT.Issuer,
			Audience:   cfg.Auth.JWT.Audience,
			HMACSecret: cfg.Auth.JWT.HMACSecret,
			ClockSkew:  cfg.Auth.JWT.ClockSkew,
			Revoker:    jwtRevoker,
		}),
		rateLimit: buildRateLimit(cfg),
	}, nil
}

// buildRateLimit 按配置构造认证端点限流中间件；未启用时返回 nil（不限流）。
func buildRateLimit(cfg *config.Config) gin.HandlerFunc {
	if !cfg.Auth.RateLimit.Enabled {
		return nil
	}

	return httpmiddleware.NewLimiter(
		cfg.Auth.RateLimit.RequestsPerMinute,
		cfg.Auth.RateLimit.Burst,
	).Middleware(nil)
}

// Mount 把 User 路由挂到给定的路由组。
func (m *Module) Mount(rg *gin.RouterGroup) {
	userrouter.RegisterRoutes(rg, m.authHandler, m.userHandler, m.authMiddleware, m.rateLimit)
}

// AuthMiddleware 返回 JWT 登录态中间件，供其它需要鉴权的模块（如 demo）复用，
// 确保鉴权服务里不会出现公开可写的业务路由。
func (m *Module) AuthMiddleware() gin.HandlerFunc {
	return m.authMiddleware
}
