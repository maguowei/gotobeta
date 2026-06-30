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

// EditMessageCommand 编辑消息入参（原地更新文本内容）。
type EditMessageCommand struct {
	WorkspaceID    int64
	ConversationID int64
	OperatorUserID int64
	MessageID      int64
	Content        map[string]any
}

// ReportReadCommand 已读水位上报入参。workspaceID 由用例从会话推导。
type ReportReadCommand struct {
	ConversationID int64
	UserID         int64
	ReadSeq        int64
}
