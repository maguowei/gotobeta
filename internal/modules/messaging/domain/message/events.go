package message

import (
	"time"

	"github.com/maguowei/gotobeta/internal/pkg/event"
)

// EventMessageCreated 是消息创建领域事件名。
const EventMessageCreated = "messaging.message.created"

// CreatedEvent 在消息成功持久化后发布，供 realtime 分发与 AI 扩展订阅。
type CreatedEvent struct {
	event.BaseEvent
	WorkspaceID    int64
	ConversationID int64
	MessageID      int64
	Seq            int64
	SenderType     SenderType
	SenderID       int64
	ContentType    ContentType
}

// NewCreatedEvent 构造消息创建事件。
func NewCreatedEvent(workspaceID, conversationID, messageID, seq int64, senderType SenderType, senderID int64, contentType ContentType, occurredAt time.Time) CreatedEvent {
	return CreatedEvent{
		BaseEvent:      event.NewBaseEvent(EventMessageCreated, occurredAt),
		WorkspaceID:    workspaceID,
		ConversationID: conversationID,
		MessageID:      messageID,
		Seq:            seq,
		SenderType:     senderType,
		SenderID:       senderID,
		ContentType:    contentType,
	}
}
