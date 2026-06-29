package user

import (
	"testing"
	"time"
)

func TestNormalizeEmail(t *testing.T) {
	got, err := NormalizeEmail(" Alice@Example.COM ")
	if err != nil {
		t.Fatalf("NormalizeEmail() error = %v", err)
	}
	if got != "alice@example.com" {
		t.Fatalf("NormalizeEmail() = %q, want alice@example.com", got)
	}
}

func TestNewUserInitializesActiveUser(t *testing.T) {
	now := time.Date(2026, 5, 30, 12, 0, 0, 0, time.UTC)
	u, err := New(42, "alice@example.com", "Alice", now)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if u.ID() != 42 || u.Status() != StatusActive || u.EmailNormalized() != "alice@example.com" {
		t.Fatalf("user = %+v, want active normalized user", u)
	}
}

func TestUserMutators(t *testing.T) {
	now := time.Date(2026, 5, 30, 12, 0, 0, 0, time.UTC)
	u, err := New(42, "alice@example.com", "Alice", now)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if err := u.EnsureCanLogin(); err != nil {
		t.Fatalf("EnsureCanLogin() error = %v", err)
	}
	if u.HasPassword() {
		t.Fatalf("HasPassword() = true, want false")
	}
	if err := u.SetPassword("hash", "bcrypt", now.Add(time.Minute)); err != nil {
		t.Fatalf("SetPassword() error = %v", err)
	}
	if !u.HasPassword() || u.PasswordHashAlg() != "bcrypt" || u.PasswordSetAt() == nil {
		t.Fatalf("password state = %+v", u)
	}

	u.VerifyEmail(now.Add(2 * time.Minute))
	if u.EmailVerifiedAt() == nil {
		t.Fatalf("EmailVerifiedAt = nil")
	}
	u.TouchLogin(now.Add(3 * time.Minute))
	if u.LastLoginAt() == nil {
		t.Fatalf("LastLoginAt = nil")
	}
	u.UpdateProfile(" Alice Cooper ", " https://example.com/a.png ", now.Add(4*time.Minute))
	if u.DisplayName() != "Alice Cooper" || u.AvatarURL() != "https://example.com/a.png" {
		t.Fatalf("profile = %+v", u)
	}
}

func TestUserRejectsDisabledLogin(t *testing.T) {
	u := UnmarshalFromDB(FromDB{
		ID:     42,
		Email:  "alice@example.com",
		Status: StatusDisabled,
	})
	if err := u.EnsureCanLogin(); err == nil {
		t.Fatalf("EnsureCanLogin() error = nil, want disabled error")
	}
}
