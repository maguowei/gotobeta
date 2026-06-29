package authz

import (
	"context"
	"testing"
	"time"

	"github.com/maguowei/gotobeta/internal/modules/workspace/domain/acl"
	"github.com/maguowei/gotobeta/internal/modules/workspace/domain/rbac"
	pkgauthz "github.com/maguowei/gotobeta/internal/pkg/authz"
)

type fakeRBAC struct {
	actions map[string]struct{}
	owner   bool
	roleIDs []int64
}

func (f fakeRBAC) ResolveUserActions(_ context.Context, _, _ int64) (map[string]struct{}, error) {
	return f.actions, nil
}
func (f fakeRBAC) ListUserRoleIDs(_ context.Context, _, _ int64) ([]int64, error) {
	return f.roleIDs, nil
}
func (f fakeRBAC) HasRoleCode(_ context.Context, _, _ int64, code string) (bool, error) {
	return f.owner && code == rbac.RoleOwner, nil
}
func (fakeRBAC) CreateRole(context.Context, *rbac.Role) error                      { return nil }
func (fakeRBAC) FindRoleByCode(context.Context, int64, string) (*rbac.Role, error) { return nil, nil }
func (fakeRBAC) ListRoles(context.Context, int64) ([]*rbac.Role, error)            { return nil, nil }
func (fakeRBAC) CreatePermission(context.Context, *rbac.Permission) error          { return nil }
func (fakeRBAC) FindPermissionByCode(context.Context, int64, string) (*rbac.Permission, error) {
	return nil, nil
}
func (fakeRBAC) ListPermissions(context.Context, int64) ([]*rbac.Permission, error) { return nil, nil }
func (fakeRBAC) BindRolePermission(context.Context, int64, int64, int64) error      { return nil }
func (fakeRBAC) AssignRole(context.Context, *rbac.UserRole) error                   { return nil }
func (fakeRBAC) RevokeRole(context.Context, int64, int64, int64) error              { return nil }
func (fakeRBAC) BumpUserVersion(context.Context, int64, int64) (int64, error)       { return 0, nil }
func (fakeRBAC) GetUserVersion(context.Context, int64, int64) (int64, error)        { return 0, nil }
func (fakeRBAC) RecordChange(context.Context, rbac.ChangeLogEntry) error            { return nil }

type fakeACL struct {
	decisive *acl.Entry
}

func (f fakeACL) FindDecisive(_ context.Context, _ int64, _ int64, _ []int64, _, _, _ string, _ time.Time) (*acl.Entry, error) {
	return f.decisive, nil
}
func (fakeACL) Grant(context.Context, *acl.Entry) error { return nil }
func (fakeACL) Revoke(context.Context, int64) error     { return nil }

func req() pkgauthz.Request {
	return pkgauthz.Request{
		WorkspaceID:  1,
		Subject:      pkgauthz.Subject{UserID: 10},
		Action:       "message.send",
		ResourceType: "conversation",
		ResourceID:   "100",
	}
}

func denyEntry(t *testing.T) *acl.Entry {
	t.Helper()
	e, err := acl.NewEntry(1, 1, acl.SubjectUser, 10, "conversation", "100", "message.send", acl.EffectDeny, "冻结", 9, nil)
	if err != nil {
		t.Fatal(err)
	}
	return e
}

func allowEntry(t *testing.T) *acl.Entry {
	t.Helper()
	e, err := acl.NewEntry(2, 1, acl.SubjectUser, 10, "conversation", "100", "message.send", acl.EffectAllow, "特批", 9, nil)
	if err != nil {
		t.Fatal(err)
	}
	return e
}

func TestRBACAllowsAction(t *testing.T) {
	c := NewChecker(fakeRBAC{actions: map[string]struct{}{"message.send": {}}}, fakeACL{})
	if err := c.Check(context.Background(), req()); err != nil {
		t.Fatalf("RBAC 命中应放行，得 %v", err)
	}
}

func TestACLDenyOverridesRBAC(t *testing.T) {
	c := NewChecker(fakeRBAC{actions: map[string]struct{}{"message.send": {}}}, fakeACL{decisive: denyEntry(t)})
	if err := c.Check(context.Background(), req()); err == nil {
		t.Fatal("ACL 拒绝应覆盖 RBAC 允许")
	}
}

func TestACLAllowWithoutRBAC(t *testing.T) {
	c := NewChecker(fakeRBAC{actions: map[string]struct{}{}}, fakeACL{decisive: allowEntry(t)})
	if err := c.Check(context.Background(), req()); err != nil {
		t.Fatalf("ACL 允许应放行，得 %v", err)
	}
}

func TestOwnerShortCircuit(t *testing.T) {
	c := NewChecker(fakeRBAC{actions: map[string]struct{}{}, owner: true}, fakeACL{})
	if err := c.Check(context.Background(), req()); err != nil {
		t.Fatalf("owner 应放行，得 %v", err)
	}
}

func TestForbiddenWhenNoGrant(t *testing.T) {
	c := NewChecker(fakeRBAC{actions: map[string]struct{}{}}, fakeACL{})
	err := c.Check(context.Background(), req())
	if err == nil {
		t.Fatal("无授权应拒绝")
	}
}

func TestWorkspaceLevelActionNoResource(t *testing.T) {
	c := NewChecker(fakeRBAC{actions: map[string]struct{}{"channel.create": {}}}, fakeACL{})
	r := req()
	r.Action = "channel.create"
	r.ResourceType = ""
	r.ResourceID = ""
	if err := c.Check(context.Background(), r); err != nil {
		t.Fatalf("工作区级动作命中应放行，得 %v", err)
	}
}
