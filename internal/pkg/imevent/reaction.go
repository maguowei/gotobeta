package imevent

import (
	"time"

	"github.com/maguowei/gotobeta/internal/pkg/event"
)

// ReactionUpdated 是表情回应变更事件名。
const ReactionUpdated = "messaging.reaction.updated"

// 表情回应动作。
const (
	// ReactionActionAdd 添加回应。
	ReactionActionAdd int8 = 1
	// ReactionActionRemove 取消回应。
	ReactionActionRemove int8 = 2
)

// ReactionUpdatedEvent 在表情回应增删后发布，供 realtime 向会话成员实时同步。
type ReactionUpdatedEvent struct {
	event.BaseEvent
	WorkspaceID    int64
	ConversationID int64
	MessageID      int64
	UserID         int64
	Emoji          string
	Action         int8
}

// NewReactionUpdatedEvent 构造表情回应变更事件。
func NewReactionUpdatedEvent(workspaceID, conversationID, messageID, userID int64, emoji string, action int8, occurredAt time.Time) ReactionUpdatedEvent {
	return ReactionUpdatedEvent{
		BaseEvent:      event.NewBaseEvent(ReactionUpdated, occurredAt),
		WorkspaceID:    workspaceID,
		ConversationID: conversationID,
		MessageID:      messageID,
		UserID:         userID,
		Emoji:          emoji,
		Action:         action,
	}
}
