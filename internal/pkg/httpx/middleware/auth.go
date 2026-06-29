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
