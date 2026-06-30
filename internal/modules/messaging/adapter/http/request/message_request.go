package request

import messagingcmd "github.com/maguowei/gotobeta/internal/modules/messaging/application/command"

// SendMessageRequest 发送消息请求。
type SendMessageRequest struct {
	ClientMsgID string `json:"clientMsgId" binding:"required,max=64"`
	// ContentType: 1-text 2-image 3-file 4-voice 20-card。
	ContentType int8 `json:"contentType" binding:"required,oneof=1 2 3 4 20"`
	// Content 为 content blocks 结构化消息体（非空）。
	Content      map[string]any `json:"content" binding:"required,min=1"`
	ReplyToMsgID int64          `json:"replyToMsgId,string" binding:"min=0"`
}

// EditMessageRequest 编辑消息请求（原地更新文本内容）。
type EditMessageRequest struct {
	// Content 为编辑后的 content blocks 结构化消息体（非空）。
	Content map[string]any `json:"content" binding:"required,min=1"`
}

// ToCommand 转换为命令。
func (r EditMessageRequest) ToCommand(workspaceID, conversationID, messageID, operatorUserID int64) messagingcmd.EditMessageCommand {
	return messagingcmd.EditMessageCommand{
		WorkspaceID:    workspaceID,
		ConversationID: conversationID,
		OperatorUserID: operatorUserID,
		MessageID:      messageID,
		Content:        r.Content,
	}
}

// ReportReadRequest 已读上报请求。
type ReportReadRequest struct {
	ReadSeq int64 `json:"readSeq" binding:"required"`
}

// ToCommand 转换为命令。
func (r ReportReadRequest) ToCommand(conversationID, userID int64) messagingcmd.ReportReadCommand {
	return messagingcmd.ReportReadCommand{
		ConversationID: conversationID,
		UserID:         userID,
		ReadSeq:        r.ReadSeq,
	}
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
