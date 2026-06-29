// Package imevent 定义 IM 跨模块共享的领域事件契约。
//
// messaging 模块产生事件并经 event.Publisher 发布，realtime 等模块订阅消费，
// 双方只依赖本包，避免跨模块直接 import 领域包（符合分层边界）。
package imevent

import (
	"time"

	"github.com/maguowei/gotobeta/internal/pkg/event"
)

// MessageCreated 是消息创建事件名。
const MessageCreated = "messaging.message.created"

// MessageCreatedEvent 在消息成功持久化后发布，供 realtime 分发与 AI 扩展订阅。
type MessageCreatedEvent struct {
	event.BaseEvent
	WorkspaceID    int64
	ConversationID int64
	MessageID      int64
	Seq            int64
	SenderType     int8
	SenderID       int64
	ContentType    int8
}

// NewMessageCreatedEvent 构造消息创建事件。
func NewMessageCreatedEvent(workspaceID, conversationID, messageID, seq int64, senderType int8, senderID int64, contentType int8, occurredAt time.Time) MessageCreatedEvent {
	return MessageCreatedEvent{
		BaseEvent:      event.NewBaseEvent(MessageCreated, occurredAt),
		WorkspaceID:    workspaceID,
		ConversationID: conversationID,
		MessageID:      messageID,
		Seq:            seq,
		SenderType:     senderType,
		SenderID:       senderID,
		ContentType:    contentType,
	}
}
