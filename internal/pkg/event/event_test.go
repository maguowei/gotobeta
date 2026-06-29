package event_test

import (
	"testing"
	"time"

	"github.com/maguowei/gotobeta/internal/pkg/event"
)

// fakeEvent 内嵌 BaseEvent 验证契约可被实现。
type fakeEvent struct {
	event.BaseEvent
	payload string
}

func TestBaseEventImplementsEvent(t *testing.T) {
	now := time.Date(2026, 6, 29, 10, 0, 0, 0, time.UTC)
	var e event.Event = fakeEvent{
		BaseEvent: event.NewBaseEvent("test.created", now),
		payload:   "x",
	}

	if e.Name() != "test.created" {
		t.Fatalf("Name() = %q, want test.created", e.Name())
	}
	if !e.OccurredAt().Equal(now) {
		t.Fatalf("OccurredAt() = %v, want %v", e.OccurredAt(), now)
	}
}
