package result

import "time"

// MessageResult 是消息视图。
type MessageResult struct {
	MessageID      int64          `json:"messageId"`
	ConversationID int64          `json:"conversationId"`
	Seq            int64          `json:"seq"`
	SenderType     int8           `json:"senderType"`
	SenderID       int64          `json:"senderId"`
	ContentType    int8           `json:"contentType"`
	Content        map[string]any `json:"content"`
	ReplyToMsgID   int64          `json:"replyToMsgId"`
	Status         int8           `json:"status"`
	ServerTime     time.Time      `json:"serverTime"`
	EditedAt       *time.Time     `json:"editedAt,omitempty"`
}
