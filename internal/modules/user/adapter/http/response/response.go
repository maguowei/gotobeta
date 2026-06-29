package response

import (
	"time"

	userresult "github.com/maguowei/gotobeta/internal/modules/user/application/result"
)

// AuthResponse 是认证响应。
type AuthResponse struct {
	User                  *UserResponse `json:"user"`
	AccessToken           string        `json:"accessToken"`
	AccessTokenExpiresAt  time.Time     `json:"accessTokenExpiresAt"`
	RefreshToken          string        `json:"refreshToken"`
	RefreshTokenExpiresAt time.Time     `json:"refreshTokenExpiresAt"`
}

// UserResponse 是用户资料响应。
type UserResponse struct {
	ID            int64     `json:"id"`
	Email         string    `json:"email"`
	EmailVerified bool      `json:"emailVerified"`
	DisplayName   string    `json:"displayName"`
	AvatarURL     string    `json:"avatarUrl"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

// IdentityResponse 是三方身份响应。
type IdentityResponse struct {
	Provider      string     `json:"provider"`
	ProviderEmail string     `json:"providerEmail"`
	DisplayName   string     `json:"displayName"`
	ProfileURL    string     `json:"profileUrl"`
	LinkedAt      time.Time  `json:"linkedAt"`
	LastLoginAt   *time.Time `json:"lastLoginAt"`
}

// ToAuthResponse 转换认证响应。
func ToAuthResponse(out *userresult.AuthResult) *AuthResponse {
	return &AuthResponse{
		User:                  ToUserResponse(out.User),
		AccessToken:           out.Tokens.AccessToken,
		AccessTokenExpiresAt:  out.Tokens.AccessTokenExpiresAt,
		RefreshToken:          out.Tokens.RefreshToken,
		RefreshTokenExpiresAt: out.Tokens.RefreshTokenExpiresAt,
	}
}

// ToUserResponse 转换用户资料。
func ToUserResponse(out *userresult.UserResult) *UserResponse {
	return &UserResponse{
		ID:            out.ID,
		Email:         out.Email,
		EmailVerified: out.EmailVerified,
		DisplayName:   out.DisplayName,
		AvatarURL:     out.AvatarURL,
		Status:        out.Status,
		CreatedAt:     out.CreatedAt,
		UpdatedAt:     out.UpdatedAt,
	}
}

// ToIdentityResponses 转换三方身份列表。
func ToIdentityResponses(items []*userresult.IdentityResult) []*IdentityResponse {
	out := make([]*IdentityResponse, 0, len(items))
	for _, item := range items {
		out = append(out, &IdentityResponse{
			Provider:      item.Provider,
			ProviderEmail: item.ProviderEmail,
			DisplayName:   item.DisplayName,
			ProfileURL:    item.ProfileURL,
			LinkedAt:      item.LinkedAt,
			LastLoginAt:   item.LastLoginAt,
		})
	}
	return out
}
