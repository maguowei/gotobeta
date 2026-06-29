package oauthstate

import "time"

// OAuthState 是 OAuth 登录 state 记录。
type OAuthState struct {
	StateHash   string
	Provider    string
	RedirectURL string
	ExpiresAt   time.Time
	ConsumedAt  *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
