package query

// ListChangesQuery 增量拉取会话变更流入参。
type ListChangesQuery struct {
	WorkspaceID    int64
	OperatorUserID int64
	ConversationID int64
	AfterChangeSeq int64
	Limit          int
}
