package auth

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type claimsContextKey struct{}

// Claims 是应用内部使用的 JWT claims。
type Claims struct {
	jwt.RegisteredClaims
	UserID      int64    `json:"user_id,omitempty"`
	Email       string   `json:"email,omitempty"`
	Roles       []string `json:"roles,omitempty"`
	Permissions []string `json:"permissions,omitempty"`
}

// TokenConfig 是签发访问令牌所需的最小配置。
type TokenConfig struct {
	Issuer     string
	Audience   string
	HMACSecret string
	TTL        time.Duration
	ClockSkew  string
}

// WithClaims 把认证 claims 写入 context。
func WithClaims(ctx context.Context, claims *Claims) context.Context {
	return context.WithValue(ctx, claimsContextKey{}, claims)
}

// ClaimsFromContext 从 context 读取认证 claims。
func ClaimsFromContext(ctx context.Context) (*Claims, bool) {
	claims, ok := ctx.Value(claimsContextKey{}).(*Claims)
	return claims, ok
}

// ParseBearer 解析 Authorization: Bearer <token>。
func ParseBearer(header string) (string, bool) {
	parts := strings.Fields(header)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || parts[1] == "" {
		return "", false
	}
	return parts[1], true
}

// ParseToken 校验 HMAC JWT 并返回 claims。
func ParseToken(tokenString string, cfg TokenConfig) (*Claims, error) {
	if strings.TrimSpace(tokenString) == "" {
		return nil, fmt.Errorf("token is required")
	}
	var leeway time.Duration
	if strings.TrimSpace(cfg.ClockSkew) != "" {
		parsed, err := time.ParseDuration(cfg.ClockSkew)
		if err != nil {
			return nil, fmt.Errorf("parse clock skew: %w", err)
		}
		leeway = parsed
	}
	options := []jwt.ParserOption{
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(cfg.Issuer),
		jwt.WithLeeway(leeway),
	}
	if strings.TrimSpace(cfg.Audience) != "" {
		options = append(options, jwt.WithAudience(cfg.Audience))
	}
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method %s", token.Method.Alg())
		}
		return []byte(cfg.HMACSecret), nil
	}, options...)
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	return claims, nil
}

// IssueAccessToken 签发应用访问令牌。
func IssueAccessToken(userID int64, email string, cfg TokenConfig, now time.Time) (string, time.Time, error) {
	if userID <= 0 {
		return "", time.Time{}, fmt.Errorf("user id is required")
	}
	if strings.TrimSpace(cfg.HMACSecret) == "" {
		return "", time.Time{}, fmt.Errorf("hmac secret is required")
	}
	ttl := cfg.TTL
	if ttl <= 0 {
		ttl = 15 * time.Minute
	}
	expiresAt := now.Add(ttl)
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    cfg.Issuer,
			Subject:   fmt.Sprintf("%d", userID),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(now),
		},
		UserID: userID,
		Email:  email,
	}
	if strings.TrimSpace(cfg.Audience) != "" {
		claims.Audience = jwt.ClaimStrings{cfg.Audience}
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(cfg.HMACSecret))
	if err != nil {
		return "", time.Time{}, err
	}
	return token, expiresAt, nil
}
