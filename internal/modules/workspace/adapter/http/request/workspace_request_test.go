package request

import "testing"

func TestRequestToCommand(t *testing.T) {
	cw := CreateWorkspaceRequest{Slug: "acme", Name: "Acme"}.ToCommand(9)
	if cw.Slug != "acme" || cw.Name != "Acme" || cw.OwnerUserID != 9 {
		t.Fatalf("create 映射错误: %+v", cw)
	}
	im := InviteMemberRequest{UserID: 7, RoleCode: "member"}.ToCommand(1, 9)
	if im.WorkspaceID != 1 || im.OperatorUserID != 9 || im.TargetUserID != 7 || im.RoleCode != "member" {
		t.Fatalf("invite 映射错误: %+v", im)
	}
	ar := AssignRoleRequest{RoleCode: "admin"}.ToCommand(1, 9, 7)
	if ar.WorkspaceID != 1 || ar.OperatorUserID != 9 || ar.TargetUserID != 7 || ar.RoleCode != "admin" {
		t.Fatalf("assign 映射错误: %+v", ar)
	}
}
