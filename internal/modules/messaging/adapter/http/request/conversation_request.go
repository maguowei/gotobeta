// Package request 定义 messaging 模块的 HTTP 请求体，仅从 application 命令映射。
package request

import messagingcmd "github.com/maguowei/gotobeta/internal/modules/messaging/application/command"

// CreateConversationRequest 创建会话请求。
type CreateConversationRequest struct {
	// Type: 1-单聊 2-群聊 3-频道。
	Type int8 `json:"type" binding:"required"`
	// PeerUserId 单聊对端（Type=1 必填）。
	PeerUserID int64  `json:"peerUserId,string"`
	Name       string `json:"name"`
	Topic      string `json:"topic"`
	// Visibility: 1-public 2-private（频道用）。
	Visibility int8 `json:"visibility"`
}

// ToCommand 转换为命令。
func (r CreateConversationRequest) ToCommand(workspaceID, operatorUserID int64) messagingcmd.CreateConversationCommand {
	return messagingcmd.CreateConversationCommand{
		WorkspaceID:    workspaceID,
		OperatorUserID: operatorUserID,
		Type:           r.Type,
		PeerUserID:     r.PeerUserID,
		Name:           r.Name,
		Topic:          r.Topic,
		Visibility:     r.Visibility,
	}
}

// AddMemberRequest 加入会话成员请求。
type AddMemberRequest struct {
	// MemberType: 1-user 2-bot，缺省按 user。
	MemberType int8  `json:"memberType"`
	MemberID   int64 `json:"memberId,string" binding:"required"`
	// Role: 2-admin 3-member，缺省按 member。
	Role int8 `json:"role"`
}

// ToCommand 转换为命令。
func (r AddMemberRequest) ToCommand(workspaceID, operatorUserID, conversationID int64) messagingcmd.AddMemberCommand {
	memberType := r.MemberType
	if memberType == 0 {
		memberType = 1
	}
	return messagingcmd.AddMemberCommand{
		WorkspaceID:    workspaceID,
		OperatorUserID: operatorUserID,
		ConversationID: conversationID,
		MemberType:     memberType,
		MemberID:       r.MemberID,
		Role:           r.Role,
	}
}
