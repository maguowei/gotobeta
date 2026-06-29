package response

import (
	"time"

	messagingresult "github.com/maguowei/gotobeta/internal/modules/messaging/application/result"
)

// MessageResponse 是消息响应。
type MessageResponse struct {
	MessageID      int64          `json:"messageId,string"`
	ConversationID int64          `json:"conversationId,string"`
	Seq            int64          `json:"seq"`
	SenderType     int8           `json:"senderType"`
	SenderID       int64          `json:"senderId,string"`
	ContentType    int8           `json:"contentType"`
	Content        map[string]any `json:"content"`
	ReplyToMsgID   int64          `json:"replyToMsgId,string"`
	Status         int8           `json:"status"`
	ServerTime     string         `json:"serverTime"`
}

// ToMessageResponse 转换消息结果。
func ToMessageResponse(out *messagingresult.MessageResult) MessageResponse {
	return MessageResponse{
		MessageID:      out.MessageID,
		ConversationID: out.ConversationID,
		Seq:            out.Seq,
		SenderType:     out.SenderType,
		SenderID:       out.SenderID,
		ContentType:    out.ContentType,
		Content:        out.Content,
		ReplyToMsgID:   out.ReplyToMsgID,
		Status:         out.Status,
		ServerTime:     out.ServerTime.Format(time.DateTime),
	}
}

// ToMessageListResponse 批量转换消息。
func ToMessageListResponse(items []*messagingresult.MessageResult) []MessageResponse {
	out := make([]MessageResponse, 0, len(items))
	for _, item := range items {
		out = append(out, ToMessageResponse(item))
	}
	return out
}
