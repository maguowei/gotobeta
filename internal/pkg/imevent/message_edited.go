package imevent

import (
	"time"

	"github.com/maguowei/gotobeta/internal/pkg/event"
)

// MessageEdited 是消息编辑事件名。
const MessageEdited = "messaging.message.edited"

// MessageEditedEvent 在消息内容原地编辑后发布，供 realtime 向会话在线成员推送新内容。
//
// 编辑不占新 seq、不进 timeline，靠本事件携带新 content 让在线端原地替换。
type MessageEditedEvent struct {
	event.BaseEvent
	WorkspaceID    int64
	ConversationID int64
	MessageID      int64
	Content        map[string]any
	EditedAt       time.Time
}

// NewMessageEditedEvent 构造消息编辑事件。
func NewMessageEditedEvent(workspaceID, conversationID, messageID int64, content map[string]any, editedAt time.Time) MessageEditedEvent {
	return MessageEditedEvent{
		BaseEvent:      event.NewBaseEvent(MessageEdited, editedAt),
		WorkspaceID:    workspaceID,
		ConversationID: conversationID,
		MessageID:      messageID,
		Content:        content,
		EditedAt:       editedAt,
	}
}
