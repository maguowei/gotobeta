package workspace_test

import (
	"testing"

	"github.com/maguowei/gotobeta/internal/modules/workspace/domain/workspace"
)

func TestNewValidatesSlug(t *testing.T) {
	cases := []struct {
		name string
		slug string
		ok   bool
	}{
		{"valid", "acme-team", true},
		{"valid-digits", "team42", true},
		{"empty", "", false},
		{"uppercase", "Acme", false},
		{"leading-hyphen", "-acme", false},
		{"trailing-hyphen", "acme-", false},
		{"space", "ac me", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := workspace.New(1, tc.slug, "Acme", 10)
			if tc.ok && err != nil {
				t.Fatalf("expected valid slug %q, got %v", tc.slug, err)
			}
			if !tc.ok && err == nil {
				t.Fatalf("expected invalid slug %q to fail", tc.slug)
			}
		})
	}
}

func TestNewRequiresOwnerAndName(t *testing.T) {
	if _, err := workspace.New(1, "acme", "", 10); err == nil {
		t.Fatal("empty name should fail")
	}
	if _, err := workspace.New(1, "acme", "Acme", 0); err == nil {
		t.Fatal("missing owner should fail")
	}
}

func TestNewDefaultsActive(t *testing.T) {
	w, err := workspace.New(1, "acme", "Acme", 10)
	if err != nil {
		t.Fatalf("New error: %v", err)
	}
	if w.Status() != workspace.StatusActive {
		t.Fatalf("status = %d, want active", w.Status())
	}
	if w.OwnerUserID() != 10 {
		t.Fatalf("owner = %d, want 10", w.OwnerUserID())
	}
}

func TestDisable(t *testing.T) {
	w, _ := workspace.New(1, "acme", "Acme", 10)
	w.Disable()
	if w.Status() != workspace.StatusDisabled {
		t.Fatalf("status = %d, want disabled", w.Status())
	}
}
