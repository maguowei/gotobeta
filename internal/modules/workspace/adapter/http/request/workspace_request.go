// Package request 定义 workspace 模块的 HTTP 请求体，仅从 application 命令映射。
package request

import workspacecmd "github.com/maguowei/gotobeta/internal/modules/workspace/application/command"

// CreateWorkspaceRequest 创建工作区请求。
type CreateWorkspaceRequest struct {
	Slug string `json:"slug" binding:"required"`
	Name string `json:"name" binding:"required"`
}

// ToCommand 转换为命令（owner 来自登录态）。
func (r CreateWorkspaceRequest) ToCommand(ownerUserID int64) workspacecmd.CreateWorkspaceCommand {
	return workspacecmd.CreateWorkspaceCommand{Slug: r.Slug, Name: r.Name, OwnerUserID: ownerUserID}
}

// InviteMemberRequest 邀请成员请求。
type InviteMemberRequest struct {
	UserID   int64  `json:"userId,string" binding:"required"`
	RoleCode string `json:"roleCode" binding:"required"`
}

// ToCommand 转换为命令。
func (r InviteMemberRequest) ToCommand(workspaceID, operatorUserID int64) workspacecmd.InviteMemberCommand {
	return workspacecmd.InviteMemberCommand{
		WorkspaceID:    workspaceID,
		OperatorUserID: operatorUserID,
		TargetUserID:   r.UserID,
		RoleCode:       r.RoleCode,
	}
}

// AssignRoleRequest 分配角色请求。
type AssignRoleRequest struct {
	RoleCode string `json:"roleCode" binding:"required"`
}

// ToCommand 转换为命令。
func (r AssignRoleRequest) ToCommand(workspaceID, operatorUserID, targetUserID int64) workspacecmd.AssignRoleCommand {
	return workspacecmd.AssignRoleCommand{
		WorkspaceID:    workspaceID,
		OperatorUserID: operatorUserID,
		TargetUserID:   targetUserID,
		RoleCode:       r.RoleCode,
	}
}
