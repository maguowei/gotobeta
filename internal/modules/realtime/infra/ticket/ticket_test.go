package ticket

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestIssueConsumeOnce(t *testing.T) {
	t.Parallel()
	s := NewStore(nil, time.Minute)
	token, err := s.Issue(context.Background(), 42)
	if err != nil {
		t.Fatalf("签发失败: %v", err)
	}
	uid, err := s.Consume(context.Background(), token)
	if err != nil {
		t.Fatalf("消费失败: %v", err)
	}
	if uid != 42 {
		t.Fatalf("userID 错误: %d", uid)
	}
	// 二次消费应失败（一次性）。
	if _, err := s.Consume(context.Background(), token); !errors.Is(err, ErrInvalidTicket) {
		t.Fatalf("二次消费应失败, got %v", err)
	}
}

func TestConsumeExpired(t *testing.T) {
	t.Parallel()
	s := NewStore(nil, time.Nanosecond)
	token, _ := s.Issue(context.Background(), 7)
	time.Sleep(time.Millisecond)
	if _, err := s.Consume(context.Background(), token); !errors.Is(err, ErrInvalidTicket) {
		t.Fatalf("过期 ticket 应失败, got %v", err)
	}
}

func TestConsumeUnknown(t *testing.T) {
	t.Parallel()
	s := NewStore(nil, time.Minute)
	if _, err := s.Consume(context.Background(), "nope"); !errors.Is(err, ErrInvalidTicket) {
		t.Fatalf("未知 ticket 应失败, got %v", err)
	}
	if _, err := s.Consume(context.Background(), ""); !errors.Is(err, ErrInvalidTicket) {
		t.Fatalf("空 ticket 应失败, got %v", err)
	}
}
