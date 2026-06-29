package command

// SendMessageCommand 发送消息入参。
type SendMessageCommand struct {
	WorkspaceID    int64
	ConversationID int64
	SenderUserID   int64
	ClientMsgID    string
	ContentType    int8
	Content        map[string]any
	ReplyToMsgID   int64
}

// RecallMessageCommand 撤回消息入参。
type RecallMessageCommand struct {
	WorkspaceID    int64
	ConversationID int64
	OperatorUserID int64
	MessageID      int64
}
