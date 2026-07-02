package persistence

import (
	"testing"
	"time"

	"github.com/maguowei/gotobeta/internal/ent"
	"github.com/maguowei/gotobeta/internal/modules/messaging/domain/messagechange"
)

func TestMessageChangeToEntity(t *testing.T) {
	now := time.Now()
	row := &ent.MessageChange{
		BizID: 11, ConversationID: 100, ChangeSeq: 5, ChangeType: 3,
		MessageID: 8001, ActorID: 9, Payload: map[string]any{"emoji": "👍"}, CreatedAt: now,
	}
	c := messageChangeToEntity(row)
	if c.ID() != 11 || c.ConversationID() != 100 || c.ChangeSeq() != 5 {
		t.Fatalf("映射基础字段错误: %+v", c)
	}
	if c.Type() != messagechange.ChangeReactionAdd || c.MessageID() != 8001 || c.ActorID() != 9 {
		t.Fatalf("映射类型/目标错误: %+v", c)
	}
	if c.Payload()["emoji"] != "👍" || !c.CreatedAt().Equal(now) {
		t.Fatalf("映射 payload/时间错误: %+v", c)
	}
}
