package request

import messagingcmd "github.com/maguowei/gotobeta/internal/modules/messaging/application/command"

// AddReactionRequest 添加表情回应请求。
type AddReactionRequest struct {
	Emoji string `json:"emoji" binding:"required,max=64"`
}

// ToCommand 转换为命令。
func (r AddReactionRequest) ToCommand(workspaceID, conversationID, messageID, operatorUserID int64) messagingcmd.AddReactionCommand {
	return messagingcmd.AddReactionCommand{
		WorkspaceID:    workspaceID,
		ConversationID: conversationID,
		MessageID:      messageID,
		OperatorUserID: operatorUserID,
		Emoji:          r.Emoji,
	}
}
