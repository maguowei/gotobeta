package message

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func textContent(s string) map[string]any { return map[string]any{"text": s} }

func TestNewValidation(t *testing.T) {
	t.Parallel()
	if _, err := New(1, 1, 1, SenderSystem, 0, "", ContentText, textContent("hi"), 0); err == nil {
		t.Fatal("system 不是可发送类型应报错")
	}
	if _, err := New(1, 1, 1, SenderUser, 9, "c1", ContentText, map[string]any{}, 0); err == nil {
		t.Fatal("空内容应报错")
	}
	if _, err := New(1, 1, 1, SenderUser, 9, "c1", ContentText, textContent(""), 0); err == nil {
		t.Fatal("空文本应报错")
	}
	if _, err := New(1, 1, 1, SenderUser, 9, "c1", ContentType(99), textContent("hi"), 0); err == nil {
		t.Fatal("非法内容类型应报错")
	}
	m, err := New(1, 1, 5, SenderUser, 9, "c1", ContentText, textContent("hi"), 0)
	if err != nil {
		t.Fatalf("意外错误: %v", err)
	}
	if m.Seq() != 5 || m.Status() != StatusNormal || m.ClientMsgID() == nil || *m.ClientMsgID() != "c1" {
		t.Fatalf("消息状态错误: %+v", m)
	}
}

func TestRecallWindow(t *testing.T) {
	t.Parallel()
	m, _ := New(1, 1, 1, SenderUser, 9, "c1", ContentText, textContent("hi"), 0)
	now := m.ServerTime()
	// 超窗。
	if err := m.Recall(now.Add(3*time.Minute), 2*time.Minute); !errors.Is(err, ErrRecallWindowExpired) {
		t.Fatalf("超窗应返回 ErrRecallWindowExpired, got %v", err)
	}
	// 窗内。
	if err := m.Recall(now.Add(1*time.Minute), 2*time.Minute); err != nil {
		t.Fatalf("窗内撤回应成功: %v", err)
	}
	if m.Status() != StatusRecalled {
		t.Fatalf("撤回后状态应为 Recalled, got %d", m.Status())
	}
	// 重复撤回。
	if err := m.Recall(now.Add(1*time.Minute), 2*time.Minute); !errors.Is(err, ErrNotRecallable) {
		t.Fatalf("重复撤回应返回 ErrNotRecallable, got %v", err)
	}
}

func TestDigest(t *testing.T) {
	t.Parallel()
	m, _ := New(1, 1, 1, SenderUser, 9, "c1", ContentText, textContent("hello world"), 0)
	if m.Digest() != "hello world" {
		t.Fatalf("文本摘要错误: %q", m.Digest())
	}
	long, _ := New(2, 1, 2, SenderUser, 9, "c2", ContentText, textContent(strings.Repeat("a", 100)), 0)
	if got := long.Digest(); len([]rune(got)) != 60 {
		t.Fatalf("长文本应截断到 60 runes, got %d", len([]rune(got)))
	}
	img, _ := New(3, 1, 3, SenderUser, 9, "c3", ContentImage, map[string]any{"url": "x"}, 0)
	if img.Digest() != "[图片]" {
		t.Fatalf("图片摘要错误: %q", img.Digest())
	}
}

func TestCreatedEvent(t *testing.T) {
	t.Parallel()
	e := NewCreatedEvent(1, 100, 8001, 5, SenderUser, 9, ContentText, time.Now())
	if e.Name() != EventMessageCreated {
		t.Fatalf("事件名错误: %q", e.Name())
	}
	if e.ConversationID != 100 || e.Seq != 5 {
		t.Fatalf("事件载荷错误: %+v", e)
	}
}
