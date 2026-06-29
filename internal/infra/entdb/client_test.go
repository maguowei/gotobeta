package entdb

import (
	"strings"
	"testing"

	"github.com/maguowei/gotobeta/internal/infra/config"
)

func TestEnsureMySQLTimeZone(t *testing.T) {
	tests := []struct {
		name string
		dsn  string
		want []string
	}{
		{
			name: "adds defaults",
			dsn:  "user:pass@tcp(127.0.0.1:3306)/app",
			want: []string{"parseTime=true", "loc=Local"},
		},
		{
			name: "keeps existing values",
			dsn:  "user:pass@tcp(127.0.0.1:3306)/app?parseTime=false&loc=UTC",
			want: []string{"parseTime=false", "loc=UTC"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ensureMySQLTimeZone(tt.dsn)
			if err != nil {
				t.Fatalf("ensureMySQLTimeZone() error = %v", err)
			}
			for _, item := range tt.want {
				if !strings.Contains(got, item) {
					t.Fatalf("dsn = %q, want contains %q", got, item)
				}
			}
		})
	}
}

func TestEnsureMySQLTimeZoneRejectsInvalidQuery(t *testing.T) {
	if _, err := ensureMySQLTimeZone("user:pass@tcp(localhost:3306)/app?bad=%zz"); err == nil {
		t.Fatalf("ensureMySQLTimeZone() error = nil, want invalid query error")
	}
}

func TestNewEntClientRejectsUnknownDriver(t *testing.T) {
	_, _, err := NewEntClient(&config.DatabaseConfig{
		Driver: "unknown",
		DSN:    "user:pass@tcp(127.0.0.1:3306)/app",
	})
	if err == nil || !strings.Contains(err.Error(), "open database") {
		t.Fatalf("NewEntClient() error = %v, want open database error", err)
	}
}
