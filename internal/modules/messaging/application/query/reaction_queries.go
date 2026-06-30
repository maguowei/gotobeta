package query

// ListReactionsQuery 列举某条消息的表情回应入参。
type ListReactionsQuery struct {
	WorkspaceID    int64
	ConversationID int64
	MessageID      int64
	OperatorUserID int64
}
