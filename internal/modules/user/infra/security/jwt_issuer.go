package security

import (
	"time"

	"github.com/maguowei/gotobeta/internal/infra/config"
	userdomain "github.com/maguowei/gotobeta/internal/modules/user/domain/user"
	"github.com/maguowei/gotobeta/internal/pkg/auth"
)

// JWTIssuer 签发 access token。
type JWTIssuer struct {
	cfg config.AuthJWTConfig
	ttl time.Duration
}

// NewJWTIssuer 创建 JWTIssuer。
func NewJWTIssuer(cfg config.AuthJWTConfig, ttl time.Duration) *JWTIssuer {
	return &JWTIssuer{cfg: cfg, ttl: ttl}
}

// IssueAccessToken 签发 access token。
func (i *JWTIssuer) IssueAccessToken(user *userdomain.User, now time.Time) (string, time.Time, error) {
	return auth.IssueAccessToken(user.ID(), user.EmailNormalized(), auth.TokenConfig{
		Issuer:     i.cfg.Issuer,
		Audience:   i.cfg.Audience,
		HMACSecret: i.cfg.HMACSecret,
		TTL:        i.ttl,
		ClockSkew:  i.cfg.ClockSkew,
	}, now)
}
