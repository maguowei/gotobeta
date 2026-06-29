package requestctx

import (
	"context"
	"testing"
)

func TestWorkspaceIDRoundTrip(t *testing.T) {
	ctx := WithWorkspaceID(context.Background(), 42)

	got, ok := WorkspaceID(ctx)
	if !ok || got != 42 {
		t.Fatalf("WorkspaceID() = (%d, %v), want (42, true)", got, ok)
	}
}

func TestWorkspaceIDUnsetReturnsFalse(t *testing.T) {
	got, ok := WorkspaceID(context.Background())
	if ok || got != 0 {
		t.Fatalf("WorkspaceID() = (%d, %v), want (0, false)", got, ok)
	}
}
