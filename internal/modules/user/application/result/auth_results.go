// Package result 定义用户认证用例出参。
package result

import "time"

// TokenPair 是登录令牌。
type TokenPair struct {
	AccessToken           string
	AccessTokenExpiresAt  time.Time
	RefreshToken          string
	RefreshTokenExpiresAt time.Time
}

// AuthResult 是认证结果。
type AuthResult struct {
	User   *UserResult
	Tokens TokenPair
}

// UserResult 是用户资料结果。
type UserResult struct {
	ID            int64
	Email         string
	EmailVerified bool
	DisplayName   string
	AvatarURL     string
	Status        string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// OAuthStartResult 是 OAuth start 结果。
type OAuthStartResult struct {
	AuthURL string
	State   string
}

// OAuthCallbackResult 是 OAuth callback 结果。
type OAuthCallbackResult struct {
	RedirectURL string
	LoginCode   string
}

// IdentityResult 是第三方登录身份结果。
type IdentityResult struct {
	Provider      string
	ProviderEmail string
	DisplayName   string
	ProfileURL    string
	LinkedAt      time.Time
	LastLoginAt   *time.Time
}
