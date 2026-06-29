package actiontoken

import (
	"testing"
	"time"
)

func TestIsValidPurpose(t *testing.T) {
	for _, p := range []string{ActionEmailVerification, ActionPasswordReset, ActionOAuthLoginCode} {
		if !IsValidPurpose(p) {
			t.Fatalf("IsValidPurpose(%q) = false, want true", p)
		}
	}
	if IsValidPurpose("unknown") {
		t.Fatal("IsValidPurpose(unknown) = true, want false")
	}
}

func TestNewValidatesInput(t *testing.T) {
	now := time.Now()
	future := now.Add(time.Hour)

	tests := []struct {
		name      string
		tokenID   string
		userID    int64
		purpose   string
		tokenHash string
		expiresAt time.Time
		wantErr   bool
	}{
		{"valid", "tid", 1, ActionPasswordReset, "hash", future, false},
		{"empty token id", "", 1, ActionPasswordReset, "hash", future, true},
		{"invalid user", "tid", 0, ActionPasswordReset, "hash", future, true},
		{"invalid purpose", "tid", 1, "bogus", "hash", future, true},
		{"empty hash", "tid", 1, ActionPasswordReset, "", future, true},
		{"not after now", "tid", 1, ActionPasswordReset, "hash", now, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := New(tt.tokenID, tt.userID, tt.purpose, tt.tokenHash, "user@example.com", tt.expiresAt, now)
			if tt.wantErr {
				if err == nil {
					t.Fatal("New() error = nil, want validation error")
				}
				return
			}
			if err != nil {
				t.Fatalf("New() error = %v", err)
			}
			if token.Purpose != tt.purpose {
				t.Fatalf("Purpose = %q, want %q", token.Purpose, tt.purpose)
			}
			if token.ConsumedAt != nil {
				t.Fatal("New() ConsumedAt must be nil")
			}
		})
	}
}
