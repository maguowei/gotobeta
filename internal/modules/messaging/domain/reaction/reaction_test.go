package reaction

import (
	"strings"
	"testing"
	"time"
)

func TestNewValidatesEmoji(t *testing.T) {
	if _, err := New(1, 2, 3, 4, ""); err == nil {
		t.Fatal("空 emoji 应被拒绝")
	}
	if _, err := New(1, 2, 3, 4, strings.Repeat("a", 65)); err == nil {
		t.Fatal("超长 emoji 应被拒绝")
	}
}

func TestNewAndGetters(t *testing.T) {
	r, err := New(10, 20, 30, 40, "👍")
	if err != nil {
		t.Fatalf("创建失败: %v", err)
	}
	if r.ID() != 10 || r.ConversationID() != 20 || r.MessageID() != 30 || r.UserID() != 40 || r.Emoji() != "👍" {
		t.Fatalf("字段不符: %+v", r)
	}
	if r.CreatedAt().IsZero() {
		t.Fatal("createdAt 应被初始化")
	}
}

func TestUnmarshalFromDB(t *testing.T) {
	r := UnmarshalFromDB(1, 2, 3, 4, "❤️", time.Now())
	if r.MessageID() != 3 || r.Emoji() != "❤️" {
		t.Fatalf("重建字段不符: %+v", r)
	}
}
