package query

// PullMessagesQuery 增量拉取会话消息入参。
type PullMessagesQuery struct {
	WorkspaceID    int64
	OperatorUserID int64
	ConversationID int64
	AfterSeq       int64
	Limit          int
}
