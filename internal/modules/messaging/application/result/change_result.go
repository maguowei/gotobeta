package result

// ChangeResult 是单条变更视图。
type ChangeResult struct {
	ChangeSeq  int64          `json:"changeSeq"`
	ChangeType int8           `json:"changeType"`
	MessageID  int64          `json:"messageId"`
	ActorID    int64          `json:"actorId"`
	Payload    map[string]any `json:"payload"`
}

// ChangesPage 是一页变更流结果。
type ChangesPage struct {
	Changes    []*ChangeResult
	NextCursor int64
	HasMore    bool
}
