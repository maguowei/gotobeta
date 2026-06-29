package response

import (
	"testing"
	"time"

	workspaceresult "github.com/maguowei/gotobeta/internal/modules/workspace/application/result"
)

func TestResponses(t *testing.T) {
	now := time.Date(2026, 6, 29, 10, 0, 0, 0, time.UTC)
	ws := ToWorkspaceResponse(&workspaceresult.WorkspaceResult{ID: 1, Slug: "acme", Name: "Acme", OwnerUserID: 9, Status: 1, CreatedAt: now})
	if ws.WorkspaceID != 1 || ws.Slug != "acme" || ws.CreatedAt == "" {
		t.Fatalf("workspace 响应错误: %+v", ws)
	}
	list := ToWorkspaceListResponse([]*workspaceresult.WorkspaceResult{{ID: 1, CreatedAt: now}})
	if len(list) != 1 {
		t.Fatalf("列表长度错误: %d", len(list))
	}
	m := ToMemberResponse(&workspaceresult.MemberResult{WorkspaceID: 1, UserID: 7, Status: 1, JoinedAt: now})
	if m.WorkspaceID != 1 || m.UserID != 7 {
		t.Fatalf("member 响应错误: %+v", m)
	}
	roles := ToRoleListResponse([]*workspaceresult.RoleResult{{ID: 2, Code: "admin", Name: "管理员"}})
	if len(roles) != 1 || roles[0].Code != "admin" {
		t.Fatalf("role 响应错误: %+v", roles)
	}
}
