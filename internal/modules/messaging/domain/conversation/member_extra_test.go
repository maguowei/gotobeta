package conversation

import (
	"testing"
	"time"
)

// TestNewMemberDefaults 校验成员构造的初始状态与 getter。
func TestNewMemberDefaults(t *testing.T) {
	t.Parallel()
	m := NewMember(5, 9, MemberBot, 100, RoleOwner)
	if m.ID() != 5 || m.ConversationID() != 9 || m.MemberType() != MemberBot {
		t.Fatalf("成员基础字段错误: %+v", m)
	}
	if m.MemberID() != 100 || m.Role() != RoleOwner {
		t.Fatalf("成员主体/角色错误: %+v", m)
	}
	if m.ReadSeq() != 0 || m.LastReadAt() != nil {
		t.Fatalf("初始已读水位应为 0/nil: %+v", m)
	}
	if m.IsMuted() || m.IsPinned() {
		t.Fatal("初始静音/置顶应为 false")
	}
	if m.Status() != MemberActive {
		t.Fatalf("初始状态应为 Active, got %d", m.Status())
	}
	if m.JoinedAt().IsZero() || m.CreatedAt().IsZero() || m.UpdatedAt().IsZero() {
		t.Fatal("时间戳不应为零值")
	}
}

// TestUnmarshalMemberFromDB 校验成员从 DB 重建全字段透传。
func TestUnmarshalMemberFromDB(t *testing.T) {
	t.Parallel()
	now := time.Now()
	readAt := now.Add(time.Minute)
	m := UnmarshalMemberFromDB(1, 2, MemberUser, 100, RoleAdmin, 42, &readAt,
		true, true, MemberLeft, now, now, now)
	if m.ID() != 1 || m.ConversationID() != 2 || m.MemberType() != MemberUser || m.MemberID() != 100 {
		t.Fatalf("重建基础字段错误: %+v", m)
	}
	if m.Role() != RoleAdmin || m.ReadSeq() != 42 {
		t.Fatalf("重建角色/已读错误: %+v", m)
	}
	if m.LastReadAt() == nil || !m.LastReadAt().Equal(readAt) {
		t.Fatal("重建 lastReadAt 错误")
	}
	if !m.IsMuted() || !m.IsPinned() {
		t.Fatal("重建静音/置顶应为 true")
	}
	if m.Status() != MemberLeft {
		t.Fatalf("重建状态错误, got %d", m.Status())
	}
	if !m.JoinedAt().Equal(now) || !m.CreatedAt().Equal(now) || !m.UpdatedAt().Equal(now) {
		t.Fatal("重建时间戳错误")
	}
}

// TestMemberLeave 校验退出的幂等：首次成功、再次返回 false。
func TestMemberLeave(t *testing.T) {
	t.Parallel()
	m := NewMember(1, 1, MemberUser, 100, RoleMember)
	if !m.Leave() {
		t.Fatal("首次退出应返回 true")
	}
	if m.Status() != MemberLeft {
		t.Fatalf("退出后状态应为 Left, got %d", m.Status())
	}
	if m.Leave() {
		t.Fatal("已退出再次退出应返回 false")
	}
}

// TestMemberMarkReadSetsTimestamp 校验推进已读水位会更新 lastReadAt。
func TestMemberMarkReadSetsTimestamp(t *testing.T) {
	t.Parallel()
	m := NewMember(1, 1, MemberUser, 100, RoleMember)
	at := time.Now()
	if !m.MarkRead(7, at) {
		t.Fatal("推进已读应返回 true")
	}
	if m.LastReadAt() == nil || !m.LastReadAt().Equal(at) {
		t.Fatal("MarkRead 应设置 lastReadAt")
	}
}
