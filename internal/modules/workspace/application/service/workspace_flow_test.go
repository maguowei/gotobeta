package service

import (
	"context"
	"log/slog"
	"sync/atomic"
	"testing"
	"time"

	workspacecmd "github.com/maguowei/gotobeta/internal/modules/workspace/application/command"
	workspacequery "github.com/maguowei/gotobeta/internal/modules/workspace/application/query"
	"github.com/maguowei/gotobeta/internal/modules/workspace/domain/membership"
	"github.com/maguowei/gotobeta/internal/modules/workspace/domain/rbac"
	"github.com/maguowei/gotobeta/internal/modules/workspace/domain/workspace"
	"github.com/maguowei/gotobeta/internal/pkg/authz"
)

// --- 测试替身 ---

type allowChecker struct{}

func (allowChecker) Check(context.Context, authz.Request) error { return nil }

type seqIDGen struct{ n atomic.Int64 }

func (g *seqIDGen) NextID(context.Context) (int64, error) { return g.n.Add(1), nil }

type directTx struct{}

func (directTx) RunInTx(ctx context.Context, fn func(context.Context) error) error { return fn(ctx) }

type memWorkspaceRepo struct {
	bySlug map[string]*workspace.Workspace
	byID   map[int64]*workspace.Workspace
	byUser []*workspace.Workspace
}

func newMemWorkspaceRepo() *memWorkspaceRepo {
	return &memWorkspaceRepo{bySlug: map[string]*workspace.Workspace{}, byID: map[int64]*workspace.Workspace{}}
}
func (r *memWorkspaceRepo) Create(_ context.Context, w *workspace.Workspace) error {
	r.bySlug[w.Slug()] = w
	r.byID[w.ID()] = w
	return nil
}
func (r *memWorkspaceRepo) FindByID(_ context.Context, id int64) (*workspace.Workspace, error) {
	if w, ok := r.byID[id]; ok {
		return w, nil
	}
	return nil, workspace.ErrNotFound
}
func (r *memWorkspaceRepo) FindBySlug(_ context.Context, slug string) (*workspace.Workspace, error) {
	if w, ok := r.bySlug[slug]; ok {
		return w, nil
	}
	return nil, workspace.ErrNotFound
}
func (r *memWorkspaceRepo) Save(_ context.Context, w *workspace.Workspace) error {
	r.byID[w.ID()] = w
	return nil
}
func (r *memWorkspaceRepo) ListByMemberUser(context.Context, int64) ([]*workspace.Workspace, error) {
	return r.byUser, nil
}

type memMembershipRepo struct {
	members map[int64]*membership.Member // key=userID
}

func newMemMembershipRepo() *memMembershipRepo {
	return &memMembershipRepo{members: map[int64]*membership.Member{}}
}
func (r *memMembershipRepo) Add(_ context.Context, m *membership.Member) error {
	if _, ok := r.members[m.UserID()]; ok {
		return membership.ErrAlreadyMember
	}
	r.members[m.UserID()] = m
	return nil
}
func (r *memMembershipRepo) FindByWorkspaceUser(_ context.Context, _, userID int64) (*membership.Member, error) {
	if m, ok := r.members[userID]; ok {
		return m, nil
	}
	return nil, membership.ErrNotFound
}
func (r *memMembershipRepo) ListByUser(context.Context, int64) ([]*membership.Member, error) {
	return nil, nil
}
func (r *memMembershipRepo) ListByWorkspace(context.Context, int64) ([]*membership.Member, error) {
	return nil, nil
}

type memRBACRepo struct {
	rolesByCode map[string]*rbac.Role
	perms       map[string]*rbac.Permission
	assigned    int
	bindings    int
	permSeq     atomic.Int64
}

func newMemRBACRepo() *memRBACRepo {
	return &memRBACRepo{rolesByCode: map[string]*rbac.Role{}, perms: map[string]*rbac.Permission{}}
}
func (r *memRBACRepo) CreateRole(_ context.Context, role *rbac.Role) error {
	r.rolesByCode[role.Code()] = role
	return nil
}
func (r *memRBACRepo) FindRoleByCode(_ context.Context, _ int64, code string) (*rbac.Role, error) {
	if role, ok := r.rolesByCode[code]; ok {
		return role, nil
	}
	return nil, rbac.ErrRoleNotFound
}
func (r *memRBACRepo) ListRoles(context.Context, int64) ([]*rbac.Role, error) {
	out := make([]*rbac.Role, 0, len(r.rolesByCode))
	for _, role := range r.rolesByCode {
		out = append(out, role)
	}
	return out, nil
}
func (r *memRBACRepo) CreatePermission(_ context.Context, p *rbac.Permission) error {
	r.perms[p.Code()] = p
	return nil
}
func (r *memRBACRepo) FindPermissionByCode(_ context.Context, _ int64, code string) (*rbac.Permission, error) {
	if p, ok := r.perms[code]; ok {
		return p, nil
	}
	p := rbac.NewPermission(r.permSeq.Add(1), 0, code, code, "res", "act")
	r.perms[code] = p
	return p, nil
}
func (r *memRBACRepo) ListPermissions(context.Context, int64) ([]*rbac.Permission, error) {
	return nil, nil
}
func (r *memRBACRepo) BindRolePermission(context.Context, int64, int64, int64) error {
	r.bindings++
	return nil
}
func (r *memRBACRepo) AssignRole(context.Context, *rbac.UserRole) error {
	r.assigned++
	return nil
}
func (r *memRBACRepo) RevokeRole(context.Context, int64, int64, int64) error { return nil }
func (r *memRBACRepo) ResolveUserActions(context.Context, int64, int64) (map[string]struct{}, error) {
	return map[string]struct{}{}, nil
}
func (r *memRBACRepo) ListUserRoleIDs(context.Context, int64, int64) ([]int64, error) {
	return nil, nil
}
func (r *memRBACRepo) HasRoleCode(context.Context, int64, int64, string) (bool, error) {
	return false, nil
}
func (r *memRBACRepo) PermissionVersion(context.Context, int64, int64) (int64, error) {
	return 1, nil
}
func (r *memRBACRepo) BumpPermissionVersion(context.Context, int64, int64) error {
	return nil
}

func newSvc(ws *memWorkspaceRepo, ms *memMembershipRepo, rb *memRBACRepo) *WorkspaceService {
	return NewWorkspaceService(ws, ms, rb, allowChecker{}, &seqIDGen{}, directTx{}, slog.Default())
}

// --- 用例流程测试 ---

func TestCreateWorkspaceHappyPath(t *testing.T) {
	ws, ms, rb := newMemWorkspaceRepo(), newMemMembershipRepo(), newMemRBACRepo()
	svc := newSvc(ws, ms, rb)
	out, err := svc.CreateWorkspace(context.Background(), workspacecmd.CreateWorkspaceCommand{
		Slug: "acme", Name: "Acme", OwnerUserID: 9,
	})
	if err != nil {
		t.Fatalf("创建失败: %v", err)
	}
	if out.Slug != "acme" || out.OwnerUserID != 9 {
		t.Fatalf("结果错误: %+v", out)
	}
	if len(rb.rolesByCode) != 4 {
		t.Fatalf("应复制 4 个角色, got %d", len(rb.rolesByCode))
	}
	if _, ok := ms.members[9]; !ok {
		t.Fatal("owner 应被加入成员")
	}
	if rb.assigned != 1 {
		t.Fatalf("owner 应被赋角色一次, got %d", rb.assigned)
	}
}

func TestInviteMemberHappyPath(t *testing.T) {
	ws, ms, rb := newMemWorkspaceRepo(), newMemMembershipRepo(), newMemRBACRepo()
	rb.rolesByCode["member"] = rbac.NewRole(100, 1, "member", "成员")
	svc := newSvc(ws, ms, rb)
	out, err := svc.InviteMember(context.Background(), workspacecmd.InviteMemberCommand{
		WorkspaceID: 1, OperatorUserID: 9, TargetUserID: 7, RoleCode: "member",
	})
	if err != nil {
		t.Fatalf("邀请失败: %v", err)
	}
	if out.UserID != 7 {
		t.Fatalf("结果错误: %+v", out)
	}
}

func TestInviteMemberRoleNotFound(t *testing.T) {
	svc := newSvc(newMemWorkspaceRepo(), newMemMembershipRepo(), newMemRBACRepo())
	_, err := svc.InviteMember(context.Background(), workspacecmd.InviteMemberCommand{
		WorkspaceID: 1, OperatorUserID: 9, TargetUserID: 7, RoleCode: "ghost",
	})
	if err == nil {
		t.Fatal("角色不存在应报错")
	}
}

func TestAssignRoleHappyPath(t *testing.T) {
	rb := newMemRBACRepo()
	rb.rolesByCode["admin"] = rbac.NewRole(101, 1, "admin", "管理员")
	svc := newSvc(newMemWorkspaceRepo(), newMemMembershipRepo(), rb)
	if err := svc.AssignRole(context.Background(), workspacecmd.AssignRoleCommand{
		WorkspaceID: 1, OperatorUserID: 9, TargetUserID: 7, RoleCode: "admin",
	}); err != nil {
		t.Fatalf("分配失败: %v", err)
	}
	if rb.assigned != 1 {
		t.Fatalf("应分配一次, got %d", rb.assigned)
	}
}

func TestListMyWorkspaces(t *testing.T) {
	ws := newMemWorkspaceRepo()
	ws.byUser = []*workspace.Workspace{
		workspace.UnmarshalFromDB(1, "acme", "Acme", 9, workspace.StatusActive, nil, time.Time{}, time.Time{}),
	}
	svc := newSvc(ws, newMemMembershipRepo(), newMemRBACRepo())
	items, err := svc.ListMyWorkspaces(context.Background(), workspacequery.ListMyWorkspacesQuery{UserID: 9})
	if err != nil || len(items) != 1 {
		t.Fatalf("列表错误: %v %d", err, len(items))
	}
}

func TestListRolesRequiresMembership(t *testing.T) {
	svc := newSvc(newMemWorkspaceRepo(), newMemMembershipRepo(), newMemRBACRepo())
	_, err := svc.ListRoles(context.Background(), workspacequery.ListRolesQuery{WorkspaceID: 1, OperatorUserID: 9})
	if err == nil {
		t.Fatal("非成员应被拒绝")
	}
}

func TestListRolesHappyPath(t *testing.T) {
	ms := newMemMembershipRepo()
	m, _ := membership.New(1, 1, 9)
	ms.members[9] = m
	rb := newMemRBACRepo()
	rb.rolesByCode["owner"] = rbac.NewRole(1, 1, "owner", "所有者")
	svc := newSvc(newMemWorkspaceRepo(), ms, rb)
	items, err := svc.ListRoles(context.Background(), workspacequery.ListRolesQuery{WorkspaceID: 1, OperatorUserID: 9})
	if err != nil || len(items) != 1 {
		t.Fatalf("列表错误: %v %d", err, len(items))
	}
}
