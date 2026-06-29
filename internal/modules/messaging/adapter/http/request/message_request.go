package request

import messagingcmd "github.com/maguowei/gotobeta/internal/modules/messaging/application/command"

// SendMessageRequest 发送消息请求。
type SendMessageRequest struct {
	ClientMsgID string `json:"clientMsgId" binding:"required"`
	// ContentType: 1-text 2-image 3-file 4-voice 20-card。
	ContentType int8 `json:"contentType" binding:"required"`
	// Content 为 content blocks 结构化消息体。
	Content      map[string]any `json:"content" binding:"required"`
	ReplyToMsgID int64          `json:"replyToMsgId,string"`
}

// ToCommand 转换为命令。
func (r SendMessageRequest) ToCommand(workspaceID, conversationID, senderUserID int64) messagingcmd.SendMessageCommand {
	return messagingcmd.SendMessageCommand{
		WorkspaceID:    workspaceID,
		ConversationID: conversationID,
		SenderUserID:   senderUserID,
		ClientMsgID:    r.ClientMsgID,
		ContentType:    r.ContentType,
		Content:        r.Content,
		ReplyToMsgID:   r.ReplyToMsgID,
	}
}
