package identity

import "time"

const (
	ProviderGitHub = "github"
	ProviderGoogle = "google"
)

// Identity 是用户第三方登录身份。
type Identity struct {
	ID                      int64
	UserID                  int64
	Provider                string
	ProviderUserID          string
	ProviderEmail           string
	ProviderEmailNormalized string
	ProviderEmailVerified   bool
	DisplayName             string
	AvatarURL               string
	ProfileURL              string
	LinkedAt                time.Time
	LastLoginAt             *time.Time
	CreatedAt               time.Time
	UpdatedAt               time.Time
}
