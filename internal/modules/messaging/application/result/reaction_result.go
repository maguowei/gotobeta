package result

// ReactionResult 是表情回应视图。
type ReactionResult struct {
	MessageID int64  `json:"messageId"`
	UserID    int64  `json:"userId"`
	Emoji     string `json:"emoji"`
}
