// Package command 定义 messaging 模块写用例入参（CQRS Command）。
package command

// CreateConversationCommand 创建会话/频道入参。
//
// Type=1 单聊时使用 PeerUserID 与发起人去重；Type=2/3 使用 Name 与 Visibility。
type CreateConversationCommand struct {
	WorkspaceID    int64
	OperatorUserID int64
	Type           int8
	// PeerUserID 单聊对端用户（Type=1 必填）。
	PeerUserID int64
	Name       string
	Topic      string
	Visibility int8
}

// AddMemberCommand 向会话加入成员入参。
type AddMemberCommand struct {
	WorkspaceID    int64
	OperatorUserID int64
	ConversationID int64
	MemberType     int8
	MemberID       int64
	Role           int8
}

// RemoveMemberCommand 从会话移除成员入参。
type RemoveMemberCommand struct {
	WorkspaceID    int64
	OperatorUserID int64
	ConversationID int64
	MemberType     int8
	MemberID       int64
}
