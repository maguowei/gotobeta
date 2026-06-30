package message

import (
	"errors"
	"strings"
	"testing"
	"time"
)

// TestNewBotSenderAndGetters 校验机器人发送者路径与 getter 透传。
func TestNewBotSenderAndGetters(t *testing.T) {
	t.Parallel()
	content := map[string]any{"text": "hi"}
	m, err := New(11, 22, 3, SenderBot, 100, "client-1", ContentText, content, 99)
	if err != nil {
		t.Fatalf("意外错误: %v", err)
	}
	if m.ID() != 11 || m.ConversationID() != 22 || m.Seq() != 3 {
		t.Fatalf("基础字段错误: %+v", m)
	}
	if m.SenderType() != SenderBot || m.SenderID() != 100 {
		t.Fatalf("发送者字段错误: %+v", m)
	}
	if m.ContentType() != ContentText || m.Content()["text"] != "hi" {
		t.Fatalf("内容字段错误: %+v", m)
	}
	if m.ReplyToMsgID() != 99 {
		t.Fatalf("回复字段错误: %d", m.ReplyToMsgID())
	}
	if m.Status() != StatusNormal || m.ServerTime().IsZero() {
		t.Fatalf("状态/时间错误: %+v", m)
	}
	if m.Metadata() == nil || m.CreatedAt().IsZero() || m.UpdatedAt().IsZero() {
		t.Fatalf("metadata/时间戳错误: %+v", m)
	}
}

// TestNewEmptyClientMsgID 校验空 clientMsgID 时指针为 nil。
func TestNewEmptyClientMsgID(t *testing.T) {
	t.Parallel()
	m, err := New(1, 1, 1, SenderUser, 9, "", ContentImage, map[string]any{"url": "x"}, 0)
	if err != nil {
		t.Fatalf("意外错误: %v", err)
	}
	if m.ClientMsgID() != nil {
		t.Fatalf("空 clientMsgID 应为 nil, got %v", *m.ClientMsgID())
	}
}

// TestNewSendableContentTypes 表驱动校验可发送内容类型分支。
func TestNewSendableContentTypes(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		ct      ContentType
		content map[string]any
		wantErr bool
	}{
		{"image", ContentImage, map[string]any{"url": "x"}, false},
		{"file", ContentFile, map[string]any{"url": "x"}, false},
		{"voice", ContentVoice, map[string]any{"url": "x"}, false},
		{"card", ContentCard, map[string]any{"title": "t"}, false},
		{"recall-not-sendable", ContentRecall, map[string]any{"k": "v"}, true},
		{"system-not-sendable", ContentSystem, map[string]any{"k": "v"}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := New(1, 1, 1, SenderUser, 9, "c1", tc.ct, tc.content, 0)
			if tc.wantErr && err == nil {
				t.Fatalf("内容类型 %d 应报错", tc.ct)
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("内容类型 %d 意外错误: %v", tc.ct, err)
			}
		})
	}
}

// TestNewSystem 校验系统条目构造，含 nil content 兜底。
func TestNewSystem(t *testing.T) {
	t.Parallel()
	m := NewSystem(5, 9, 7, ContentRecall, map[string]any{"target": 1})
	if m.ID() != 5 || m.ConversationID() != 9 || m.Seq() != 7 {
		t.Fatalf("系统条目基础字段错误: %+v", m)
	}
	if m.SenderType() != SenderSystem || m.SenderID() != 0 {
		t.Fatalf("系统条目发送者应为 system/0: %+v", m)
	}
	if m.ContentType() != ContentRecall || m.Content()["target"] != 1 {
		t.Fatalf("系统条目内容错误: %+v", m)
	}
	if m.Status() != StatusNormal || m.Metadata() == nil {
		t.Fatalf("系统条目状态/metadata 错误: %+v", m)
	}

	// nil content 应兜底为空 map。
	m2 := NewSystem(1, 1, 1, ContentSystem, nil)
	if m2.Content() == nil {
		t.Fatal("nil content 应兜底为空 map")
	}
}

// TestRecallDeletedStatus 校验非正常状态（已删除）不可撤回。
func TestRecallDeletedStatus(t *testing.T) {
	t.Parallel()
	now := time.Now()
	m := UnmarshalFromDB(1, 1, 1, SenderUser, 9, nil, ContentText, map[string]any{"text": "x"}, 0,
		StatusDeleted, now, nil, nil, now, now)
	if err := m.Recall(now, time.Minute); !errors.Is(err, ErrNotRecallable) {
		t.Fatalf("已删除消息撤回应返回 ErrNotRecallable, got %v", err)
	}
}

// TestEditSuccess 校验文本消息在窗口内可原地编辑并记录 editedAt。
func TestEditSuccess(t *testing.T) {
	t.Parallel()
	m, _ := New(1, 1, 1, SenderUser, 9, "c1", ContentText, map[string]any{"text": "old"}, 0)
	now := m.ServerTime().Add(time.Second)
	if err := m.Edit(map[string]any{"text": "new"}, now, time.Minute); err != nil {
		t.Fatalf("编辑应成功: %v", err)
	}
	if m.Content()["text"] != "new" {
		t.Fatalf("内容应更新为 new, got %v", m.Content()["text"])
	}
	if m.EditedAt() == nil || !m.EditedAt().Equal(now) {
		t.Fatalf("editedAt 应为 %v, got %v", now, m.EditedAt())
	}
	if !m.UpdatedAt().Equal(now) {
		t.Fatal("updatedAt 应同步为编辑时间")
	}
}

// TestEditRejects 表驱动覆盖编辑的各拒绝分支。
func TestEditRejects(t *testing.T) {
	t.Parallel()
	now := time.Now()
	cases := []struct {
		name    string
		msg     *Message
		content map[string]any
		window  time.Duration
		wantErr error
	}{
		{
			"非文本不可编辑",
			UnmarshalFromDB(1, 1, 1, SenderUser, 9, nil, ContentImage, map[string]any{"url": "x"}, 0, StatusNormal, now, nil, nil, now, now),
			map[string]any{"text": "new"}, time.Minute, ErrNotEditable,
		},
		{
			"已撤回不可编辑",
			UnmarshalFromDB(1, 1, 1, SenderUser, 9, nil, ContentText, map[string]any{"text": "x"}, 0, StatusRecalled, now, nil, nil, now, now),
			map[string]any{"text": "new"}, time.Minute, ErrNotEditable,
		},
		{
			"超窗不可编辑",
			UnmarshalFromDB(1, 1, 1, SenderUser, 9, nil, ContentText, map[string]any{"text": "x"}, 0, StatusNormal, now.Add(-time.Hour), nil, nil, now, now),
			map[string]any{"text": "new"}, time.Minute, ErrEditWindowExpired,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if err := tc.msg.Edit(tc.content, now, tc.window); !errors.Is(err, tc.wantErr) {
				t.Fatalf("应返回 %v, got %v", tc.wantErr, err)
			}
		})
	}
}

// TestEditEmptyTextRejected 校验编辑后文本为空被拒绝。
func TestEditEmptyTextRejected(t *testing.T) {
	t.Parallel()
	m, _ := New(1, 1, 1, SenderUser, 9, "c1", ContentText, map[string]any{"text": "old"}, 0)
	if err := m.Edit(map[string]any{"text": ""}, m.ServerTime(), time.Minute); err == nil {
		t.Fatal("空文本编辑应被拒绝")
	}
}

// TestDigestAllTypes 表驱动覆盖所有摘要分支。
func TestDigestAllTypes(t *testing.T) {
	t.Parallel()
	now := time.Now()
	build := func(ct ContentType, status Status, content map[string]any) *Message {
		return UnmarshalFromDB(1, 1, 1, SenderUser, 9, nil, ct, content, 0, status, now, nil, nil, now, now)
	}
	cases := []struct {
		name string
		msg  *Message
		want string
	}{
		{"recalled", build(ContentText, StatusRecalled, map[string]any{"text": "x"}), "撤回了一条消息"},
		{"text", build(ContentText, StatusNormal, map[string]any{"text": "hello"}), "hello"},
		{"image", build(ContentImage, StatusNormal, map[string]any{"url": "x"}), "[图片]"},
		{"file", build(ContentFile, StatusNormal, map[string]any{"url": "x"}), "[文件]"},
		{"voice", build(ContentVoice, StatusNormal, map[string]any{"url": "x"}), "[语音]"},
		{"card", build(ContentCard, StatusNormal, map[string]any{"title": "t"}), "[卡片]"},
		{"system-default-empty", build(ContentSystem, StatusNormal, map[string]any{"k": "v"}), ""},
		{"text-missing-key", build(ContentText, StatusNormal, map[string]any{"k": "v"}), ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := tc.msg.Digest(); got != tc.want {
				t.Fatalf("摘要错误: want %q got %q", tc.want, got)
			}
		})
	}
}

// TestDigestTruncateBoundary 校验恰好等于上限时不截断。
func TestDigestTruncateBoundary(t *testing.T) {
	t.Parallel()
	now := time.Now()
	exact := strings.Repeat("字", 60)
	m := UnmarshalFromDB(1, 1, 1, SenderUser, 9, nil, ContentText, map[string]any{"text": exact}, 0,
		StatusNormal, now, nil, nil, now, now)
	if got := m.Digest(); len([]rune(got)) != 60 {
		t.Fatalf("恰好 60 runes 不应截断, got %d", len([]rune(got)))
	}
}

// TestUnmarshalFromDB 校验消息从 DB 重建全字段透传与 nil 兜底。
func TestUnmarshalFromDB(t *testing.T) {
	t.Parallel()
	now := time.Now()
	server := now.Add(time.Minute)
	client := "client-1"
	content := map[string]any{"text": "hi"}
	meta := map[string]any{"k": "v"}

	m := UnmarshalFromDB(11, 22, 3, SenderBot, 100, &client, ContentText, content, 99,
		StatusRecalled, server, nil, meta, now, now)
	if m.ID() != 11 || m.ConversationID() != 22 || m.Seq() != 3 {
		t.Fatalf("重建基础字段错误: %+v", m)
	}
	if m.SenderType() != SenderBot || m.SenderID() != 100 {
		t.Fatalf("重建发送者错误: %+v", m)
	}
	if m.ClientMsgID() == nil || *m.ClientMsgID() != client {
		t.Fatalf("重建 clientMsgID 错误: %+v", m.ClientMsgID())
	}
	if m.ContentType() != ContentText || m.Content()["text"] != "hi" {
		t.Fatalf("重建内容错误: %+v", m)
	}
	if m.ReplyToMsgID() != 99 || m.Status() != StatusRecalled {
		t.Fatalf("重建回复/状态错误: %+v", m)
	}
	if !m.ServerTime().Equal(server) {
		t.Fatal("重建 serverTime 错误")
	}
	if m.Metadata()["k"] != "v" {
		t.Fatalf("重建 metadata 错误: %+v", m.Metadata())
	}
	if !m.CreatedAt().Equal(now) || !m.UpdatedAt().Equal(now) {
		t.Fatal("重建时间戳错误")
	}

	// nil content / metadata 应兜底为空 map。
	m2 := UnmarshalFromDB(1, 1, 1, SenderUser, 9, nil, ContentText, nil, 0,
		StatusNormal, now, nil, nil, now, now)
	if m2.Content() == nil || m2.Metadata() == nil {
		t.Fatal("nil content/metadata 应兜底为空 map")
	}
}
