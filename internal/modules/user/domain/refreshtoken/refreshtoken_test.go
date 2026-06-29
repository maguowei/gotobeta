package refreshtoken

import (
	"testing"
	"time"
)

func TestNewValidatesInput(t *testing.T) {
	now := time.Now()
	future := now.Add(time.Hour)

	tests := []struct {
		name      string
		tokenID   string
		userID    int64
		tokenHash string
		expiresAt time.Time
		wantErr   bool
	}{
		{"valid", "tid", 1, "hash", future, false},
		{"empty token id", "", 1, "hash", future, true},
		{"invalid user", "tid", 0, "hash", future, true},
		{"empty hash", "tid", 1, "", future, true},
		{"not after now", "tid", 1, "hash", now, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := New(tt.tokenID, tt.userID, tt.tokenHash, tt.expiresAt, now)
			if tt.wantErr {
				if err == nil {
					t.Fatal("New() error = nil, want validation error")
				}
				return
			}
			if err != nil {
				t.Fatalf("New() error = %v", err)
			}
			if token.TokenID != tt.tokenID {
				t.Fatalf("TokenID = %q, want %q", token.TokenID, tt.tokenID)
			}
			if !token.CreatedAt.Equal(now) || !token.UpdatedAt.Equal(now) {
				t.Fatal("New() must stamp CreatedAt/UpdatedAt with now")
			}
			if token.RevokedAt != nil {
				t.Fatal("New() RevokedAt must be nil")
			}
		})
	}
}
