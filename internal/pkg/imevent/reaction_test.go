package imevent

import (
	"testing"
	"time"
)

func TestNewReactionUpdatedEvent(t *testing.T) {
	at := time.Now()
	e := NewReactionUpdatedEvent(7, 100, 8001, 9, "👍", ReactionActionAdd, at)
	if e.Name() != ReactionUpdated {
		t.Fatalf("事件名错误: %q", e.Name())
	}
	if !e.OccurredAt().Equal(at) {
		t.Fatalf("发生时间错误: %v", e.OccurredAt())
	}
	if e.WorkspaceID != 7 || e.ConversationID != 100 || e.MessageID != 8001 || e.UserID != 9 {
		t.Fatalf("事件字段错误: %+v", e)
	}
	if e.Emoji != "👍" || e.Action != ReactionActionAdd {
		t.Fatalf("表情/动作字段错误: %+v", e)
	}
}
