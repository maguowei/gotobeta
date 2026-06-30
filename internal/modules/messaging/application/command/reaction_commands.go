package command

// AddReactionCommand 添加表情回应入参。
type AddReactionCommand struct {
	WorkspaceID    int64
	ConversationID int64
	MessageID      int64
	OperatorUserID int64
	Emoji          string
}

// RemoveReactionCommand 取消表情回应入参（仅能取消本人回应）。
type RemoveReactionCommand struct {
	WorkspaceID    int64
	ConversationID int64
	MessageID      int64
	OperatorUserID int64
	Emoji          string
}
