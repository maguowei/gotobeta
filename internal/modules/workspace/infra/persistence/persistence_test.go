package persistence

import (
	"log/slog"
	"testing"
	"time"

	"github.com/maguowei/gotobeta/internal/ent"
	"github.com/maguowei/gotobeta/internal/modules/workspace/domain/acl"
	"github.com/maguowei/gotobeta/internal/modules/workspace/domain/workspace"
)

func TestConstructorsInjectDependencies(t *testing.T) {
	t.Parallel()
	logger := slog.Default()
	if NewWorkspaceRepository(nil, logger) == nil {
		t.Fatal("NewWorkspaceRepository = nil")
	}
	if NewMembershipRepository(nil, logger) == nil {
		t.Fatal("NewMembershipRepository = nil")
	}
	if NewRBACRepository(nil, logger, nil) == nil {
		t.Fatal("NewRBACRepository = nil")
	}
	if NewACLRepository(nil, logger) == nil {
		t.Fatal("NewACLRepository = nil")
	}
}

func TestWorkspaceToEntity(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 6, 29, 1, 2, 3, 0, time.UTC)
	got := workspaceToEntity(&ent.Workspace{
		BizID: 7, Slug: "acme", Name: "Acme", OwnerUserID: 9,
		Status: int8(workspace.StatusActive), Settings: map[string]any{"k": "v"},
		CreatedAt: now, UpdatedAt: now,
	})
	if got.ID() != 7 || got.Slug() != "acme" || got.OwnerUserID() != 9 {
		t.Fatalf("mapping mismatch: %+v", got)
	}
	if got.Status() != workspace.StatusActive {
		t.Fatalf("status = %d, want active", got.Status())
	}
}

func TestACLToEntityDenyEffect(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 6, 29, 1, 2, 3, 0, time.UTC)
	got := aclToEntity(&ent.RbacAclEntry{
		BizID: 1, WorkspaceID: 2, SubjectType: acl.SubjectUser, SubjectID: 10,
		ResourceType: "conversation", ResourceID: "100", ActionCode: "message.send",
		Effect: int8(acl.EffectDeny), Reason: "冻结", SourceType: 1, CreatedBy: 9, CreatedAt: now,
	})
	if got.Effect() != acl.EffectDeny {
		t.Fatalf("effect = %d, want deny", got.Effect())
	}
	if got.ActionCode() != "message.send" {
		t.Fatalf("action = %s", got.ActionCode())
	}
}
