// Package response 定义 workspace 模块的 HTTP 响应体，仅从 application 结果映射。
package response

import (
	"time"

	workspaceresult "github.com/maguowei/gotobeta/internal/modules/workspace/application/result"
)

// WorkspaceResponse 是工作区响应。
type WorkspaceResponse struct {
	WorkspaceID int64  `json:"workspaceId,string"`
	Slug        string `json:"slug"`
	Name        string `json:"name"`
	OwnerUserID int64  `json:"ownerUserId,string"`
	Status      int8   `json:"status"`
	CreatedAt   string `json:"createdAt"`
}

// ToWorkspaceResponse 转换工作区结果。
func ToWorkspaceResponse(out *workspaceresult.WorkspaceResult) WorkspaceResponse {
	return WorkspaceResponse{
		WorkspaceID: out.ID,
		Slug:        out.Slug,
		Name:        out.Name,
		OwnerUserID: out.OwnerUserID,
		Status:      out.Status,
		CreatedAt:   out.CreatedAt.Format(time.DateTime),
	}
}

// ToWorkspaceListResponse 批量转换。
func ToWorkspaceListResponse(items []*workspaceresult.WorkspaceResult) []WorkspaceResponse {
	out := make([]WorkspaceResponse, 0, len(items))
	for _, item := range items {
		out = append(out, ToWorkspaceResponse(item))
	}
	return out
}

// MemberResponse 是成员响应。
type MemberResponse struct {
	WorkspaceID int64  `json:"workspaceId,string"`
	UserID      int64  `json:"userId,string"`
	Status      int8   `json:"status"`
	JoinedAt    string `json:"joinedAt"`
}

// ToMemberResponse 转换成员结果。
func ToMemberResponse(out *workspaceresult.MemberResult) MemberResponse {
	return MemberResponse{
		WorkspaceID: out.WorkspaceID,
		UserID:      out.UserID,
		Status:      out.Status,
		JoinedAt:    out.JoinedAt.Format(time.DateTime),
	}
}

// RoleResponse 是角色响应。
type RoleResponse struct {
	RoleID int64  `json:"roleId,string"`
	Code   string `json:"code"`
	Name   string `json:"name"`
}

// ToRoleListResponse 批量转换角色。
func ToRoleListResponse(items []*workspaceresult.RoleResult) []RoleResponse {
	out := make([]RoleResponse, 0, len(items))
	for _, item := range items {
		out = append(out, RoleResponse{RoleID: item.ID, Code: item.Code, Name: item.Name})
	}
	return out
}
