package conversation

import (
	"testing"
	"time"
)

func TestDMKeySymmetric(t *testing.T) {
	t.Parallel()
	a := DMKey(1, 100, 200)
	b := DMKey(1, 200, 100)
	if a != b {
		t.Fatalf("DMKey 应与顺序无关: %q != %q", a, b)
	}
	if got := DMKey(1, 100, 200); got != "1:100#200" {
		t.Fatalf("DMKey 格式错误: %q", got)
	}
	if DMKey(1, 100, 200) == DMKey(2, 100, 200) {
		t.Fatal("不同 workspace 应产生不同 DMKey")
	}
}

func TestNewDMRejectsSelf(t *testing.T) {
	t.Parallel()
	if _, err := NewDM(1, 1, 100, 100, 100); err == nil {
		t.Fatal("单聊自己应报错")
	}
	c, err := NewDM(1, 1, 100, 200, 100)
	if err != nil {
		t.Fatalf("意外错误: %v", err)
	}
	if c.Type() != TypeDM || c.DMKey() == nil || *c.DMKey() != "1:100#200" {
		t.Fatalf("单聊聚合状态错误: %+v", c)
	}
	if c.MemberCount() != 2 {
		t.Fatalf("单聊成员数应为 2, got %d", c.MemberCount())
	}
}

func TestNewGroupAndChannelValidation(t *testing.T) {
	t.Parallel()
	if _, err := NewGroup(1, 1, "  ", 100); err == nil {
		t.Fatal("空群名应报错")
	}
	if _, err := NewChannel(1, 1, "general", Visibility(9), 100); err == nil {
		t.Fatal("非法可见性应报错")
	}
	ch, err := NewChannel(1, 1, "general", VisibilityPublic, 100)
	if err != nil {
		t.Fatalf("意外错误: %v", err)
	}
	if ch.Type() != TypeChannel || ch.Visibility() != VisibilityPublic {
		t.Fatalf("频道状态错误: %+v", ch)
	}
}

func TestApplyMessageAndArchive(t *testing.T) {
	t.Parallel()
	c, _ := NewGroup(1, 1, "g", 100)
	now := time.Now()
	c.ApplyMessage(5, 999, "hello", now)
	if c.LastSeq() != 5 || c.LastMsgID() != 999 || c.LastMsgDigest() != "hello" {
		t.Fatalf("ApplyMessage 未推进游标: %+v", c)
	}
	if c.LastMsgAt() == nil || !c.LastMsgAt().Equal(now) {
		t.Fatal("ApplyMessage 未设置 lastMsgAt")
	}
	if err := c.Archive(); err != nil {
		t.Fatalf("归档失败: %v", err)
	}
	if c.Status() != StatusArchived {
		t.Fatalf("状态应为归档, got %d", c.Status())
	}
}

func TestMemberMarkReadMonotonicAndUnread(t *testing.T) {
	t.Parallel()
	m := NewMember(1, 1, MemberUser, 100, RoleMember)
	if m.Unread(10) != 10 {
		t.Fatalf("初始未读应为 10, got %d", m.Unread(10))
	}
	now := time.Now()
	if !m.MarkRead(5, now) {
		t.Fatal("首次推进已读应返回 true")
	}
	if m.Unread(10) != 5 {
		t.Fatalf("推进后未读应为 5, got %d", m.Unread(10))
	}
	if m.MarkRead(3, now) {
		t.Fatal("回退已读应返回 false")
	}
	if m.ReadSeq() != 5 {
		t.Fatalf("已读水位不应回退, got %d", m.ReadSeq())
	}
	if m.Unread(4) != 0 {
		t.Fatalf("lastSeq 小于已读时未读应为 0, got %d", m.Unread(4))
	}
}
