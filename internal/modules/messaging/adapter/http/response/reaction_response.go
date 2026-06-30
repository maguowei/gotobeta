package response

import messagingresult "github.com/maguowei/gotobeta/internal/modules/messaging/application/result"

// ReactionResponse 是表情回应响应。
type ReactionResponse struct {
	MessageID int64  `json:"messageId,string"`
	UserID    int64  `json:"userId,string"`
	Emoji     string `json:"emoji"`
}

// ToReactionListResponse 批量转换表情回应。
func ToReactionListResponse(items []*messagingresult.ReactionResult) []ReactionResponse {
	out := make([]ReactionResponse, 0, len(items))
	for _, item := range items {
		out = append(out, ReactionResponse{
			MessageID: item.MessageID,
			UserID:    item.UserID,
			Emoji:     item.Emoji,
		})
	}
	return out
}
