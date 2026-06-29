package persistence

import (
	"testing"
	"time"

	"github.com/maguowei/gotobeta/internal/ent"
	userdomain "github.com/maguowei/gotobeta/internal/modules/user/domain/user"
)

func TestToUserMapsNullableFields(t *testing.T) {
	now := time.Date(2026, 5, 30, 12, 0, 0, 0, time.UTC)
	hash := "hash"
	alg := "bcrypt"

	got := toUser(&ent.User{
		BizID:           42,
		Email:           "alice@example.com",
		EmailNormalized: "alice@example.com",
		EmailVerifiedAt: &now,
		PasswordHash:    &hash,
		PasswordHashAlg: &alg,
		PasswordSetAt:   &now,
		DisplayName:     "Alice",
		AvatarURL:       "https://example.com/a.png",
		Status:          string(userdomain.StatusActive),
		LastLoginAt:     &now,
		CreatedAt:       now,
		UpdatedAt:       now,
	})

	if got.ID() != 42 || got.PasswordHash() != hash || got.PasswordHashAlg() != alg || got.EmailVerifiedAt() == nil {
		t.Fatalf("toUser() nullable fields mismatch: ID=%d PasswordHash=%q PasswordHashAlg=%q EmailVerifiedAt=%v", got.ID(), got.PasswordHash(), got.PasswordHashAlg(), got.EmailVerifiedAt())
	}
}

func TestToIdentityMapsFields(t *testing.T) {
	now := time.Date(2026, 5, 30, 12, 0, 0, 0, time.UTC)

	got := toIdentity(&ent.UserIdentity{
		BizID:                 7,
		UserBizID:             42,
		Provider:              "github",
		ProviderUserID:        "u-1",
		ProviderEmail:         "alice@example.com",
		ProviderEmailVerified: true,
		LinkedAt:              now,
		LastLoginAt:           &now,
		CreatedAt:             now,
		UpdatedAt:             now,
	})

	if got.ID != 7 || got.UserID != 42 || got.Provider != "github" || got.LastLoginAt == nil {
		t.Fatalf("toIdentity() mismatch: %+v", got)
	}
}

func TestToRefreshTokenMapsNullableFields(t *testing.T) {
	now := time.Date(2026, 5, 30, 12, 0, 0, 0, time.UTC)
	replaced := "next-token"

	got := toRefreshToken(&ent.AuthRefreshToken{
		TokenID:           "t-1",
		UserBizID:         42,
		TokenHash:         "hash",
		ReplacedByTokenID: &replaced,
		ExpiresAt:         now,
		RevokedAt:         &now,
		CreatedAt:         now,
		UpdatedAt:         now,
	})

	if got.TokenID != "t-1" || got.UserID != 42 || got.ReplacedByTokenID != replaced || got.RevokedAt == nil {
		t.Fatalf("toRefreshToken() mismatch: %+v", got)
	}
}

func TestToOAuthStateMapsNullableFields(t *testing.T) {
	now := time.Date(2026, 5, 30, 12, 0, 0, 0, time.UTC)

	got := toOAuthState(&ent.OAuthLoginState{
		StateHash:   "state-hash",
		Provider:    "google",
		RedirectURL: "https://example.com/cb",
		ExpiresAt:   now,
		ConsumedAt:  &now,
		CreatedAt:   now,
		UpdatedAt:   now,
	})

	if got.StateHash != "state-hash" || got.Provider != "google" || got.ConsumedAt == nil {
		t.Fatalf("toOAuthState() mismatch: %+v", got)
	}
}

func TestToActionTokenMapsFields(t *testing.T) {
	now := time.Date(2026, 5, 30, 12, 0, 0, 0, time.UTC)

	got := toActionToken(&ent.AuthActionToken{
		TokenID:               "a-1",
		UserBizID:             42,
		Purpose:               "email_verification",
		TokenHash:             "hash",
		TargetEmailNormalized: "alice@example.com",
		ExpiresAt:             now,
		ConsumedAt:            &now,
		CreatedAt:             now,
		UpdatedAt:             now,
	})

	if got.TokenID != "a-1" || got.UserID != 42 || got.Purpose != "email_verification" || got.ConsumedAt == nil {
		t.Fatalf("toActionToken() mismatch: %+v", got)
	}
}

func TestOptionalAndValueString(t *testing.T) {
	if optionalString("") != nil {
		t.Fatal("optionalString(\"\") should be nil")
	}
	if got := optionalString("x"); got == nil || *got != "x" {
		t.Fatalf("optionalString(\"x\") = %v, want pointer to x", got)
	}
	if valueString(nil) != "" {
		t.Fatal("valueString(nil) should be empty string")
	}
	v := "y"
	if valueString(&v) != "y" {
		t.Fatalf("valueString(&\"y\") = %q, want y", valueString(&v))
	}
}

func TestNewRepositoriesNotNil(t *testing.T) {
	if NewUserRepository(nil) == nil ||
		NewIdentityRepository(nil) == nil ||
		NewRefreshTokenRepository(nil) == nil ||
		NewActionTokenRepository(nil) == nil ||
		NewOAuthStateRepository(nil) == nil {
		t.Fatal("repository constructors must return non-nil")
	}
}
