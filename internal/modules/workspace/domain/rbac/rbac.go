// Package rbac 是动态 RBAC 聚合：角色、权限、用户角色授权。
//
// 工作区即租户（workspace_id），workspace_id=0 为平台模板。
// 鉴权动作集合通过 ResolveUserActions 解析（user_roles → role_permissions → permissions）。
package rbac

import (
	"context"
	"errors"
	"time"
)

var (
	// ErrRoleNotFound 表示角色不存在。
	ErrRoleNotFound = errors.New("rbac: role not found")
	// ErrPermissionNotFound 表示权限不存在。
	ErrPermissionNotFound = errors.New("rbac: permission not found")
)

// 默认角色编码（建工作区时从平台模板复制为租户级）。
const (
	RoleOwner  = "owner"
	RoleAdmin  = "admin"
	RoleMember = "member"
	RoleGuest  = "guest"
)

// 权限编码目录（动作授权）。
const (
	PermWorkspaceManage  = "workspace.manage"
	PermMemberInvite     = "member.invite"
	PermMemberRemove     = "member.remove"
	PermRoleManage       = "role.manage"
	PermChannelCreate    = "channel.create"
	PermChannelArchive   = "channel.archive"
	PermConversationRead = "conversation.read"
	PermMessageSend      = "message.send"
	PermMessageRecall    = "message.recall"
	PermBotManage        = "bot.manage"
)

// Role 是角色。
type Role struct {
	id          int64
	workspaceID int64
	code        string
	name        string
	roleType    int8
	status      int8
}

// NewRole 创建工作区角色。
func NewRole(id, workspaceID int64, code, name string) *Role {
	return &Role{id: id, workspaceID: workspaceID, code: code, name: name, roleType: 2, status: 1}
}

// UnmarshalRole 从数据库重建角色。
func UnmarshalRole(id, workspaceID int64, code, name string, roleType, status int8) *Role {
	return &Role{id: id, workspaceID: workspaceID, code: code, name: name, roleType: roleType, status: status}
}

func (r *Role) ID() int64          { return r.id }
func (r *Role) WorkspaceID() int64 { return r.workspaceID }
func (r *Role) Code() string       { return r.code }
func (r *Role) Name() string       { return r.name }
func (r *Role) RoleType() int8     { return r.roleType }
func (r *Role) Status() int8       { return r.status }

// Permission 是权限定义（动作目录条目）。
type Permission struct {
	id           int64
	workspaceID  int64
	code         string
	name         string
	resourceType string
	actionKey    string
	status       int8
}

// NewPermission 创建权限定义。
func NewPermission(id, workspaceID int64, code, name, resourceType, actionKey string) *Permission {
	return &Permission{id: id, workspaceID: workspaceID, code: code, name: name, resourceType: resourceType, actionKey: actionKey, status: 1}
}

// UnmarshalPermission 从数据库重建权限。
func UnmarshalPermission(id, workspaceID int64, code, name, resourceType, actionKey string, status int8) *Permission {
	return &Permission{id: id, workspaceID: workspaceID, code: code, name: name, resourceType: resourceType, actionKey: actionKey, status: status}
}

func (p *Permission) ID() int64          { return p.id }
func (p *Permission) WorkspaceID() int64 { return p.workspaceID }
func (p *Permission) Code() string       { return p.code }
func (p *Permission) Name() string       { return p.name }

// UserRole 是用户在工作区内的角色授权。
type UserRole struct {
	workspaceID    int64
	userID         int64
	roleID         int64
	sourceType     int8
	effectiveEndAt *time.Time
}

// NewUserRole 创建用户角色授权。sourceType 默认 1（手工）。
func NewUserRole(workspaceID, userID, roleID int64) *UserRole {
	return &UserRole{workspaceID: workspaceID, userID: userID, roleID: roleID, sourceType: 1}
}

func (u *UserRole) WorkspaceID() int64         { return u.workspaceID }
func (u *UserRole) UserID() int64              { return u.userID }
func (u *UserRole) RoleID() int64              { return u.roleID }
func (u *UserRole) SourceType() int8           { return u.sourceType }
func (u *UserRole) EffectiveEndAt() *time.Time { return u.effectiveEndAt }

// Repository 定义 RBAC 仓储接口。
type Repository interface {
	CreateRole(ctx context.Context, r *Role) error
	FindRoleByCode(ctx context.Context, workspaceID int64, code string) (*Role, error)
	ListRoles(ctx context.Context, workspaceID int64) ([]*Role, error)

	CreatePermission(ctx context.Context, p *Permission) error
	FindPermissionByCode(ctx context.Context, workspaceID int64, code string) (*Permission, error)
	ListPermissions(ctx context.Context, workspaceID int64) ([]*Permission, error)
	BindRolePermission(ctx context.Context, workspaceID, roleID, permissionID int64) error

	AssignRole(ctx context.Context, ur *UserRole) error
	RevokeRole(ctx context.Context, workspaceID, userID, roleID int64) error

	// ResolveUserActions 解析用户在工作区内的全部有效权限编码集合。
	ResolveUserActions(ctx context.Context, workspaceID, userID int64) (map[string]struct{}, error)
	// ListUserRoleIDs 返回用户在工作区内的有效角色 ID 列表（供 ACL 角色主体匹配）。
	ListUserRoleIDs(ctx context.Context, workspaceID, userID int64) ([]int64, error)
	// HasRoleCode 判断用户是否拥有某角色编码（如 owner 短路）。
	HasRoleCode(ctx context.Context, workspaceID, userID int64, code string) (bool, error)
}

// DefaultRoleTemplates 返回平台模板角色（workspace_id=0 seed 用）。
func DefaultRoleTemplates() []struct{ Code, Name string } {
	return []struct{ Code, Name string }{
		{RoleOwner, "所有者"},
		{RoleAdmin, "管理员"},
		{RoleMember, "成员"},
		{RoleGuest, "访客"},
	}
}

// DefaultPermissionTemplates 返回平台模板权限（workspace_id=0 seed 用）。
func DefaultPermissionTemplates() []struct{ Code, Name, ResourceType, ActionKey string } {
	return []struct{ Code, Name, ResourceType, ActionKey string }{
		{PermWorkspaceManage, "管理工作区", "workspace", "manage"},
		{PermMemberInvite, "邀请成员", "member", "invite"},
		{PermMemberRemove, "移除成员", "member", "remove"},
		{PermRoleManage, "管理角色", "role", "manage"},
		{PermChannelCreate, "创建频道", "channel", "create"},
		{PermChannelArchive, "归档频道", "channel", "archive"},
		{PermConversationRead, "查看会话", "conversation", "read"},
		{PermMessageSend, "发送消息", "message", "send"},
		{PermMessageRecall, "撤回消息", "message", "recall"},
		{PermBotManage, "管理机器人", "bot", "manage"},
	}
}

// DefaultRolePermissions 返回各默认角色绑定的权限编码（seed 用）。
// member/guest 的差异：guest 仅可读与发消息，member 可建频道。
func DefaultRolePermissions() map[string][]string {
	all := []string{
		PermWorkspaceManage, PermMemberInvite, PermMemberRemove, PermRoleManage,
		PermChannelCreate, PermChannelArchive, PermConversationRead,
		PermMessageSend, PermMessageRecall, PermBotManage,
	}
	return map[string][]string{
		RoleOwner: all,
		RoleAdmin: {
			PermMemberInvite, PermMemberRemove, PermChannelCreate, PermChannelArchive,
			PermConversationRead, PermMessageSend, PermMessageRecall, PermBotManage,
		},
		RoleMember: {PermChannelCreate, PermConversationRead, PermMessageSend, PermMessageRecall},
		RoleGuest:  {PermConversationRead, PermMessageSend},
	}
}
