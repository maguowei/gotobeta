package ticket

import (
	"context"
	"errors"
	"testing"
	"time"
)

// fakeKV 用单层 map 模拟 Redis 的 Set / GetDel 一次性消费语义。
type fakeKV struct {
	data    map[string]string
	setErr  error
	getErr  error
	lastTTL time.Duration
}

func newFakeKV() *fakeKV { return &fakeKV{data: make(map[string]string)} }

func (k *fakeKV) Set(_ context.Context, key, value string, ttl time.Duration) error {
	if k.setErr != nil {
		return k.setErr
	}
	k.lastTTL = ttl
	k.data[key] = value
	return nil
}

func (k *fakeKV) GetDel(_ context.Context, key string) (string, bool, error) {
	if k.getErr != nil {
		return "", false, k.getErr
	}
	v, ok := k.data[key]
	if !ok {
		return "", false, nil
	}
	delete(k.data, key)
	return v, true, nil
}

func TestKVIssueConsumeOnce(t *testing.T) {
	t.Parallel()
	kv := newFakeKV()
	s := NewStore(kv, time.Minute)

	token, err := s.Issue(context.Background(), 42)
	if err != nil {
		t.Fatalf("签发失败: %v", err)
	}
	if kv.data["ws:ticket:"+token] != "42" || kv.lastTTL != time.Minute {
		t.Fatalf("Issue 应写入带 TTL 的 key: data=%v ttl=%v", kv.data, kv.lastTTL)
	}

	uid, err := s.Consume(context.Background(), token)
	if err != nil {
		t.Fatalf("消费失败: %v", err)
	}
	if uid != 42 {
		t.Fatalf("userID 错误: %d", uid)
	}
	// GetDel 一次性消费：二次应为 ErrInvalidTicket。
	if _, err := s.Consume(context.Background(), token); !errors.Is(err, ErrInvalidTicket) {
		t.Fatalf("二次消费应失败, got %v", err)
	}
}

func TestKVConsumeUnknown(t *testing.T) {
	t.Parallel()
	s := NewStore(newFakeKV(), time.Minute)
	if _, err := s.Consume(context.Background(), "missing"); !errors.Is(err, ErrInvalidTicket) {
		t.Fatalf("未知 ticket 应失败, got %v", err)
	}
}

func TestKVConsumeMalformedValue(t *testing.T) {
	t.Parallel()
	kv := newFakeKV()
	kv.data["ws:ticket:bad"] = "not-a-number"
	s := NewStore(kv, time.Minute)
	if _, err := s.Consume(context.Background(), "bad"); !errors.Is(err, ErrInvalidTicket) {
		t.Fatalf("非数字 value 应视为非法 ticket, got %v", err)
	}
}

func TestKVIssueSetError(t *testing.T) {
	t.Parallel()
	boom := errors.New("set boom")
	s := NewStore(&fakeKV{data: map[string]string{}, setErr: boom}, time.Minute)
	if _, err := s.Issue(context.Background(), 1); !errors.Is(err, boom) {
		t.Fatalf("Set 错误应包装透传, got %v", err)
	}
}

func TestKVConsumeGetError(t *testing.T) {
	t.Parallel()
	boom := errors.New("get boom")
	s := NewStore(&fakeKV{data: map[string]string{}, getErr: boom}, time.Minute)
	if _, err := s.Consume(context.Background(), "any"); !errors.Is(err, boom) {
		t.Fatalf("GetDel 错误应包装透传, got %v", err)
	}
}

func TestNewStoreDefaultsTTL(t *testing.T) {
	t.Parallel()
	if got := NewStore(nil, 0).ttl; got != 30*time.Second {
		t.Fatalf("默认 TTL 应为 30s, got %v", got)
	}
	if got := NewStore(nil, -time.Second).ttl; got != 30*time.Second {
		t.Fatalf("负 TTL 应退化为 30s, got %v", got)
	}
}
