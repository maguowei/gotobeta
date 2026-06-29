package response

import (
	"testing"
	"time"

	userresult "github.com/maguowei/gotobeta/internal/modules/user/application/result"
)

func TestResponseConverters(t *testing.T) {
	now := time.Date(2026, 5, 30, 12, 0, 0, 0, time.UTC)
	user := &userresult.UserResult{
		ID:            42,
		Email:         "alice@example.com",
		EmailVerified: true,
		DisplayName:   "Alice",
		AvatarURL:     "https://example.com/a.png",
		Status:        "active",
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	auth := ToAuthResponse(&userresult.AuthResult{
		User: user,
		Tokens: userresult.TokenPair{
			AccessToken:           "access",
			AccessTokenExpiresAt:  now.Add(15 * time.Minute),
			RefreshToken:          "refresh",
			RefreshTokenExpiresAt: now.Add(24 * time.Hour),
		},
	})
	if auth.User.Email != "alice@example.com" || auth.AccessToken != "access" {
		t.Fatalf("ToAuthResponse() = %+v", auth)
	}

	if got := ToUserResponse(user); got.ID != 42 || !got.EmailVerified {
		t.Fatalf("ToUserResponse() = %+v", got)
	}

	lastLogin := now.Add(time.Hour)
	identities := ToIdentityResponses([]*userresult.IdentityResult{
		{
			Provider:      "github",
			ProviderEmail: "alice@example.com",
			DisplayName:   "Alice",
			ProfileURL:    "https://github.com/alice",
			LinkedAt:      now,
			LastLoginAt:   &lastLogin,
		},
	})
	if len(identities) != 1 || identities[0].LastLoginAt == nil {
		t.Fatalf("ToIdentityResponses() = %+v", identities)
	}
}
