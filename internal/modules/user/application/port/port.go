// Package port 定义用户认证应用层依赖的技术性端口。
package port

import (
	"context"
	"time"

	"github.com/maguowei/gotobeta/internal/modules/user/domain/oauthstate"
	userdomain "github.com/maguowei/gotobeta/internal/modules/user/domain/user"
)

// PasswordHasher 定义密码哈希端口。
type PasswordHasher interface {
	Hash(password string) (hash string, algorithm string, err error)
	Compare(hash string, password string) error
}

// SecretGenerator 生成可给用户返回的一次性 secret。
type SecretGenerator interface {
	NewToken() (string, error)
	HashToken(token string) string
}

// AccessTokenIssuer 签发 access token。
type AccessTokenIssuer interface {
	IssueAccessToken(u *userdomain.User, now time.Time) (token string, expiresAt time.Time, err error)
}

// TokenRevoker 把已签发的 access token（按 jti）加入吊销黑名单。
// ttl 为 token 剩余有效期，过期后黑名单条目自动清理；实现可降级（黑名单不可用时静默成功）。
type TokenRevoker interface {
	Revoke(ctx context.Context, jti string, ttl time.Duration) error
}

// OAuthProvider 是单个 OAuth provider 适配器。
type OAuthProvider interface {
	AuthCodeURL(state string) string
	Exchange(ctx context.Context, code string) (*oauthstate.Profile, error)
}

// OAuthProviders 按 provider 名称查找适配器。
type OAuthProviders interface {
	Get(provider string) (interface {
		AuthCodeURL(state string) string
		Exchange(ctx context.Context, code string) (*oauthstate.Profile, error)
	}, bool)
}

// EmailSender 发送认证邮件。
type EmailSender interface {
	SendEmailVerification(ctx context.Context, email string, token string) error
	SendPasswordReset(ctx context.Context, email string, token string) error
}
