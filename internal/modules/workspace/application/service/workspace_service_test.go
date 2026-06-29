package service

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	workspacecmd "github.com/maguowei/gotobeta/internal/modules/workspace/application/command"
	"github.com/maguowei/gotobeta/internal/modules/workspace/domain/workspace"
	"github.com/maguowei/gotobeta/internal/pkg/apperr"
	"github.com/maguowei/gotobeta/internal/pkg/authz"
)

type denyChecker struct{}

func (denyChecker) Check(context.Context, authz.Request) error {
	return apperr.Forbidden("denied")
}

// fakeWorkspaceRepoSlugExists 让 FindBySlug 返回已存在工作区。
type fakeWorkspaceRepoSlugExists struct{ workspace.Repository }

func (fakeWorkspaceRepoSlugExists) FindBySlug(context.Context, string) (*workspace.Workspace, error) {
	return workspace.UnmarshalFromDB(1, "acme", "Acme", 9, workspace.StatusActive, nil, time.Time{}, time.Time{}), nil
}

func TestInviteMemberDeniedByChecker(t *testing.T) {
	svc := NewWorkspaceService(nil, nil, nil, denyChecker{}, nil, nil, slog.Default())
	_, err := svc.InviteMember(context.Background(), workspacecmd.InviteMemberCommand{
		WorkspaceID: 1, OperatorUserID: 2, TargetUserID: 3, RoleCode: "member",
	})
	if err == nil {
		t.Fatal("checker 拒绝时应返回错误")
	}
	var de *apperr.DomainError
	if !errors.As(err, &de) {
		t.Fatalf("应为 DomainError，得 %v", err)
	}
}

func TestCreateWorkspaceSlugConflict(t *testing.T) {
	svc := NewWorkspaceService(fakeWorkspaceRepoSlugExists{}, nil, nil, denyChecker{}, nil, nil, slog.Default())
	_, err := svc.CreateWorkspace(context.Background(), workspacecmd.CreateWorkspaceCommand{
		Slug: "acme", Name: "Acme", OwnerUserID: 9,
	})
	if err == nil {
		t.Fatal("slug 已占用应冲突")
	}
}
