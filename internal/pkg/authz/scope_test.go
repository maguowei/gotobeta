package authz

import (
	"testing"

	"github.com/maguowei/gotobeta/internal/pkg/requestctx"
)

func TestAssertWorkspaceScope(t *testing.T) {
	t.Parallel()

	t.Run("ctx 未注入工作区时跳过", func(t *testing.T) {
		t.Parallel()
		if err := AssertWorkspaceScope(t.Context(), 1); err != nil {
			t.Fatalf("AssertWorkspaceScope() = %v, want nil", err)
		}
	})

	t.Run("工作区一致放行", func(t *testing.T) {
		t.Parallel()
		ctx := requestctx.WithWorkspaceID(t.Context(), 42)
		if err := AssertWorkspaceScope(ctx, 42); err != nil {
			t.Fatalf("AssertWorkspaceScope() = %v, want nil", err)
		}
	})

	t.Run("工作区不一致拒绝", func(t *testing.T) {
		t.Parallel()
		ctx := requestctx.WithWorkspaceID(t.Context(), 42)
		if err := AssertWorkspaceScope(ctx, 43); err == nil {
			t.Fatalf("AssertWorkspaceScope() = nil, want forbidden")
		}
	})
}
