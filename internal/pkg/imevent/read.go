package imevent

import (
	"time"

	"github.com/maguowei/gotobeta/internal/pkg/event"
)

// ReadUpdated 是已读水位更新事件名。
const ReadUpdated = "messaging.read.updated"

// ReadUpdatedEvent 在用户已读水位推进后发布，供 realtime 向本人其他端与会话成员对齐。
type ReadUpdatedEvent struct {
	event.BaseEvent
	WorkspaceID    int64
	ConversationID int64
	UserID         int64
	ReadSeq        int64
}

// NewReadUpdatedEvent 构造已读水位更新事件。
func NewReadUpdatedEvent(workspaceID, conversationID, userID, readSeq int64, occurredAt time.Time) ReadUpdatedEvent {
	return ReadUpdatedEvent{
		BaseEvent:      event.NewBaseEvent(ReadUpdated, occurredAt),
		WorkspaceID:    workspaceID,
		ConversationID: conversationID,
		UserID:         userID,
		ReadSeq:        readSeq,
	}
}
