package messagechange

import (
	"errors"
	"testing"
	"time"
)

func TestNewValidatesChangeType(t *testing.T) {
	t.Parallel()
	if _, err := New(1, 100, 5, ChangeType(99), 8001, 9, nil); !errors.Is(err, ErrInvalidChangeType) {
		t.Fatalf("非法 change_type 应返回 ErrInvalidChangeType, got %v", err)
	}
}

func TestNewAndGetters(t *testing.T) {
	t.Parallel()
	payload := map[string]any{"emoji": "👍"}
	c, err := New(11, 100, 5, ChangeReactionAdd, 8001, 9, payload)
	if err != nil {
		t.Fatalf("意外错误: %v", err)
	}
	if c.ID() != 11 || c.ConversationID() != 100 || c.ChangeSeq() != 5 {
		t.Fatalf("基础字段错误: %+v", c)
	}
	if c.Type() != ChangeReactionAdd || c.MessageID() != 8001 || c.ActorID() != 9 {
		t.Fatalf("类型/目标字段错误: %+v", c)
	}
	if c.Payload()["emoji"] != "👍" || c.CreatedAt().IsZero() {
		t.Fatalf("payload/时间错误: %+v", c)
	}
}

func TestNewNilPayloadDefaultsEmpty(t *testing.T) {
	t.Parallel()
	c, _ := New(1, 1, 1, ChangeCreated, 1, 0, nil)
	if c.Payload() == nil {
		t.Fatal("nil payload 应兜底为空 map")
	}
}

func TestUnmarshalFromDB(t *testing.T) {
	t.Parallel()
	now := time.Now()
	c := UnmarshalFromDB(11, 100, 5, ChangeEdited, 8001, 9, map[string]any{"content": "x"}, now)
	if c.ID() != 11 || c.Type() != ChangeEdited || !c.CreatedAt().Equal(now) {
		t.Fatalf("重建字段错误: %+v", c)
	}
	// nil payload 兜底。
	c2 := UnmarshalFromDB(1, 1, 1, ChangeCreated, 1, 0, nil, now)
	if c2.Payload() == nil {
		t.Fatal("nil payload 应兜底为空 map")
	}
}
