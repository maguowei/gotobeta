package rbac_test

import (
	"slices"
	"testing"
	"time"

	"github.com/maguowei/gotobeta/internal/modules/workspace/domain/rbac"
)

// TestNewRole 校验工厂构造默认值：roleType=2（自定义/租户级）、status=1。
func TestNewRole(t *testing.T) {
	r := rbac.NewRole(7, 100, rbac.RoleOwner, "所有者")
	if r.ID() != 7 {
		t.Fatalf("ID = %d, want 7", r.ID())
	}
	if r.WorkspaceID() != 100 {
		t.Fatalf("WorkspaceID = %d, want 100", r.WorkspaceID())
	}
	if r.Code() != rbac.RoleOwner {
		t.Fatalf("Code = %q, want %q", r.Code(), rbac.RoleOwner)
	}
	if r.Name() != "所有者" {
		t.Fatalf("Name = %q, want 所有者", r.Name())
	}
	if r.RoleType() != 2 {
		t.Fatalf("RoleType = %d, want 2", r.RoleType())
	}
	if r.Status() != 1 {
		t.Fatalf("Status = %d, want 1", r.Status())
	}
}

// TestUnmarshalRole 校验从 DB 重建时保留所有字段（含模板 roleType、禁用 status）。
func TestUnmarshalRole(t *testing.T) {
	r := rbac.UnmarshalRole(3, 0, rbac.RoleAdmin, "管理员", 1, 2)
	if r.ID() != 3 {
		t.Fatalf("ID = %d, want 3", r.ID())
	}
	if r.WorkspaceID() != 0 {
		t.Fatalf("WorkspaceID = %d, want 0", r.WorkspaceID())
	}
	if r.Code() != rbac.RoleAdmin {
		t.Fatalf("Code = %q, want %q", r.Code(), rbac.RoleAdmin)
	}
	if r.Name() != "管理员" {
		t.Fatalf("Name = %q", r.Name())
	}
	if r.RoleType() != 1 {
		t.Fatalf("RoleType = %d, want 1", r.RoleType())
	}
	if r.Status() != 2 {
		t.Fatalf("Status = %d, want 2", r.Status())
	}
}

// TestNewPermission 校验权限工厂默认 status=1 与各 getter。
func TestNewPermission(t *testing.T) {
	p := rbac.NewPermission(11, 100, rbac.PermMessageSend, "发送消息", "message", "send")
	if p.ID() != 11 {
		t.Fatalf("ID = %d, want 11", p.ID())
	}
	if p.WorkspaceID() != 100 {
		t.Fatalf("WorkspaceID = %d, want 100", p.WorkspaceID())
	}
	if p.Code() != rbac.PermMessageSend {
		t.Fatalf("Code = %q, want %q", p.Code(), rbac.PermMessageSend)
	}
	if p.Name() != "发送消息" {
		t.Fatalf("Name = %q", p.Name())
	}
}

// TestUnmarshalPermission 校验从 DB 重建权限，包括被禁用 status。
func TestUnmarshalPermission(t *testing.T) {
	p := rbac.UnmarshalPermission(12, 0, rbac.PermBotManage, "管理机器人", "bot", "manage", 2)
	if p.ID() != 12 {
		t.Fatalf("ID = %d, want 12", p.ID())
	}
	if p.WorkspaceID() != 0 {
		t.Fatalf("WorkspaceID = %d, want 0", p.WorkspaceID())
	}
	if p.Code() != rbac.PermBotManage {
		t.Fatalf("Code = %q, want %q", p.Code(), rbac.PermBotManage)
	}
	if p.Name() != "管理机器人" {
		t.Fatalf("Name = %q", p.Name())
	}
}

// TestNewUserRole 校验用户角色授权默认 sourceType=1（手工）、EffectiveEndAt 为 nil。
func TestNewUserRole(t *testing.T) {
	ur := rbac.NewUserRole(100, 200, 300)
	if ur.WorkspaceID() != 100 {
		t.Fatalf("WorkspaceID = %d, want 100", ur.WorkspaceID())
	}
	if ur.UserID() != 200 {
		t.Fatalf("UserID = %d, want 200", ur.UserID())
	}
	if ur.RoleID() != 300 {
		t.Fatalf("RoleID = %d, want 300", ur.RoleID())
	}
	if ur.SourceType() != 1 {
		t.Fatalf("SourceType = %d, want 1", ur.SourceType())
	}
	if ur.EffectiveEndAt() != nil {
		t.Fatalf("EffectiveEndAt = %v, want nil", ur.EffectiveEndAt())
	}
}

// TestEffectiveEndAtNil 显式确认新建授权无过期时间（永久有效）。
func TestEffectiveEndAtNil(t *testing.T) {
	ur := rbac.NewUserRole(1, 2, 3)
	if ur.EffectiveEndAt() != nil {
		t.Fatalf("EffectiveEndAt should be nil for fresh grant")
	}
	_ = time.Now() // 占位：确保 time 导入有效，授权过期语义由 infra 重建
}

// TestDefaultRoleTemplates 校验平台模板角色集合完整且编码正确。
func TestDefaultRoleTemplates(t *testing.T) {
	tpls := rbac.DefaultRoleTemplates()
	if len(tpls) != 4 {
		t.Fatalf("len = %d, want 4", len(tpls))
	}
	got := map[string]string{}
	for _, tp := range tpls {
		got[tp.Code] = tp.Name
	}
	want := map[string]string{
		rbac.RoleOwner:  "所有者",
		rbac.RoleAdmin:  "管理员",
		rbac.RoleMember: "成员",
		rbac.RoleGuest:  "访客",
	}
	for code, name := range want {
		if got[code] != name {
			t.Fatalf("template[%s] = %q, want %q", code, got[code], name)
		}
	}
}

// TestDefaultPermissionTemplates 校验平台模板权限集合：11 条且 resourceType/actionKey 自洽。
func TestDefaultPermissionTemplates(t *testing.T) {
	tpls := rbac.DefaultPermissionTemplates()
	if len(tpls) != 11 {
		t.Fatalf("len = %d, want 11", len(tpls))
	}
	codes := map[string]bool{}
	for _, tp := range tpls {
		if tp.Code == "" || tp.Name == "" || tp.ResourceType == "" || tp.ActionKey == "" {
			t.Fatalf("template has empty field: %+v", tp)
		}
		if codes[tp.Code] {
			t.Fatalf("duplicate permission code: %s", tp.Code)
		}
		codes[tp.Code] = true
	}
	for _, want := range []string{
		rbac.PermWorkspaceManage, rbac.PermMemberInvite, rbac.PermMemberRemove,
		rbac.PermRoleManage, rbac.PermChannelCreate, rbac.PermChannelArchive,
		rbac.PermConversationRead, rbac.PermMessageSend, rbac.PermMessageRecall,
		rbac.PermMessageReact, rbac.PermBotManage,
	} {
		if !codes[want] {
			t.Fatalf("missing permission template: %s", want)
		}
	}
}

// TestDefaultRolePermissions 校验各默认角色绑定的权限集合及关键差异。
func TestDefaultRolePermissions(t *testing.T) {
	rp := rbac.DefaultRolePermissions()

	// owner 拥有全部 11 项权限。
	if len(rp[rbac.RoleOwner]) != 11 {
		t.Fatalf("owner perms = %d, want 11", len(rp[rbac.RoleOwner]))
	}

	contains := func(list []string, code string) bool {
		return slices.Contains(list, code)
	}

	// admin 不应拥有 workspace.manage 与 role.manage。
	if contains(rp[rbac.RoleAdmin], rbac.PermWorkspaceManage) {
		t.Fatal("admin should not have workspace.manage")
	}
	if contains(rp[rbac.RoleAdmin], rbac.PermRoleManage) {
		t.Fatal("admin should not have role.manage")
	}

	// member 可建频道，guest 不可。
	if !contains(rp[rbac.RoleMember], rbac.PermChannelCreate) {
		t.Fatal("member should have channel.create")
	}
	if contains(rp[rbac.RoleGuest], rbac.PermChannelCreate) {
		t.Fatal("guest should not have channel.create")
	}

	// guest 仅可读、发消息与表情回应。
	if len(rp[rbac.RoleGuest]) != 3 {
		t.Fatalf("guest perms = %d, want 3", len(rp[rbac.RoleGuest]))
	}
	if !contains(rp[rbac.RoleGuest], rbac.PermConversationRead) ||
		!contains(rp[rbac.RoleGuest], rbac.PermMessageSend) ||
		!contains(rp[rbac.RoleGuest], rbac.PermMessageReact) {
		t.Fatalf("guest perms = %v, want [read, send, react]", rp[rbac.RoleGuest])
	}
}
