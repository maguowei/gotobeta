// Package command 定义 workspace 模块的写用例入参（CQRS 命令）。
package command

// CreateWorkspaceCommand 创建工作区。
type CreateWorkspaceCommand struct {
	Slug        string
	Name        string
	OwnerUserID int64
}

// InviteMemberCommand 邀请用户加入工作区并赋予角色。
type InviteMemberCommand struct {
	WorkspaceID    int64
	OperatorUserID int64
	TargetUserID   int64
	RoleCode       string
}

// AssignRoleCommand 给工作区成员分配角色。
type AssignRoleCommand struct {
	WorkspaceID    int64
	OperatorUserID int64
	TargetUserID   int64
	RoleCode       string
}
