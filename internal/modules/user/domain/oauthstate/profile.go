package oauthstate

// Profile 是三方平台返回的用户资料。
type Profile struct {
	Provider       string
	ProviderUserID string
	Email          string
	EmailVerified  bool
	DisplayName    string
	AvatarURL      string
	ProfileURL     string
}
