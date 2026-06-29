package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/maguowei/gotobeta/internal/pkg/apperr"
	"github.com/maguowei/gotobeta/internal/pkg/auth"
	httpresponse "github.com/maguowei/gotobeta/internal/pkg/httpx/response"
)

// AuthJWTOptions 控制 AuthJWT 中间件行为，由组合根从 typed config 映射而来，
// 避免共享内核反向依赖全局基础设施的配置包。
type AuthJWTOptions struct {
	Enabled    bool
	Issuer     string
	Audience   string
	HMACSecret string
	ClockSkew  string
	// Revoker 校验 token 是否已被吊销（logout 黑名单）；nil 时跳过吊销检查。
	Revoker auth.RevocationChecker
}

// AuthJWT 校验 Bearer JWT，并把 claims 写入请求 context。
func AuthJWT(opts AuthJWTOptions) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !opts.Enabled {
			c.Next()
			return
		}
		token, ok := auth.ParseBearer(c.GetHeader("Authorization"))
		if !ok {
			httpresponse.Error(c, apperr.Unauthorized("missing bearer token"))
			c.Abort()
			return
		}
		claims, err := auth.ParseToken(token, auth.TokenConfig{
			Issuer:     opts.Issuer,
			Audience:   opts.Audience,
			HMACSecret: opts.HMACSecret,
			ClockSkew:  opts.ClockSkew,
		})
		if err != nil {
			httpresponse.Error(c, apperr.Unauthorized("invalid bearer token"))
			c.Abort()
			return
		}
		// 吊销检查 fail-open：黑名单不可用（如 Redis 故障）时放行，避免误杀正常请求；
		// access token 短时有效，吊销不可用窗口的风险有限。
		if opts.Revoker != nil && claims.ID != "" {
			if revoked, err := opts.Revoker.IsRevoked(c.Request.Context(), claims.ID); err == nil && revoked {
				httpresponse.Error(c, apperr.Unauthorized("token has been revoked"))
				c.Abort()
				return
			}
		}
		c.Request = c.Request.WithContext(auth.WithClaims(c.Request.Context(), claims))
		c.Next()
	}
}

// RequireClaims 从 Gin context 读取认证 claims。
func RequireClaims(c *gin.Context) (*auth.Claims, bool) {
	claims, ok := auth.ClaimsFromContext(c.Request.Context())
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"code": httpresponse.CodeUnauthorized, "message": "未认证"})
		c.Abort()
		return nil, false
	}
	return claims, true
}
