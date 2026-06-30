package persistence

import (
	"testing"
	"time"

	"github.com/maguowei/gotobeta/internal/ent"
)

func TestReactionToEntity(t *testing.T) {
	now := time.Now()
	row := &ent.Reaction{
		BizID:          11,
		ConversationID: 22,
		MessageID:      33,
		UserID:         44,
		Emoji:          "👍",
		CreatedAt:      now,
	}
	got := reactionToEntity(row)
	if got.ID() != 11 || got.ConversationID() != 22 || got.MessageID() != 33 ||
		got.UserID() != 44 || got.Emoji() != "👍" || !got.CreatedAt().Equal(now) {
		t.Fatalf("映射结果不符: %+v", got)
	}
}
