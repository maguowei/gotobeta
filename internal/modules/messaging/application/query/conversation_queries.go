// Package query 定义 messaging 模块读用例入参（CQRS Query）。
package query

// ListConversationsQuery 列出我加入的会话入参。
type ListConversationsQuery struct {
	WorkspaceID int64
	UserID      int64
}

// ListMembersQuery 列出会话成员入参。
type ListMembersQuery struct {
	WorkspaceID    int64
	OperatorUserID int64
	ConversationID int64
}
