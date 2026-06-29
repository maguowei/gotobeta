package presence

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestInProcMarkOnlineOffline(t *testing.T) {
	t.Parallel()
	s := NewStore(nil, time.Minute)

	// 上线写入进程内 map。
	if err := s.MarkOnline(context.Background(), 7); err != nil {
		t.Fatalf("标记上线失败: %v", err)
	}
	if _, ok := s.online[7]; !ok {
		t.Fatal("上线后应记录在 online map")
	}

	// 离线删除进程内 map。
	if err := s.MarkOffline(context.Background(), 7); err != nil {
		t.Fatalf("标记离线失败: %v", err)
	}
	if _, ok := s.online[7]; ok {
		t.Fatal("离线后应从 online map 删除")
	}
}

func TestNewStoreDefaultsTTL(t *testing.T) {
	t.Parallel()
	// 非正 TTL 退化为默认值。
	if got := NewStore(nil, 0).ttl; got != 30*time.Second {
		t.Fatalf("默认 TTL 应为 30s, got %v", got)
	}
	if got := NewStore(nil, -time.Second).ttl; got != 30*time.Second {
		t.Fatalf("负 TTL 应退化为 30s, got %v", got)
	}
}

// fakeKV 记录 Set/Del 调用，用于断言 Redis 路径。
type fakeKV struct {
	setKey, setVal string
	setTTL         time.Duration
	delKey         string
	setErr, delErr error
}

func (k *fakeKV) Set(_ context.Context, key, value string, ttl time.Duration) error {
	k.setKey, k.setVal, k.setTTL = key, value, ttl
	return k.setErr
}

func (k *fakeKV) Del(_ context.Context, key string) error {
	k.delKey = key
	return k.delErr
}

func TestMarkOnlineUsesKV(t *testing.T) {
	t.Parallel()
	kv := &fakeKV{}
	s := NewStore(kv, time.Minute)

	if err := s.MarkOnline(context.Background(), 42); err != nil {
		t.Fatalf("标记上线失败: %v", err)
	}
	if kv.setKey != "presence:42" || kv.setVal != "1" || kv.setTTL != time.Minute {
		t.Fatalf("Set 参数错误: key=%q val=%q ttl=%v", kv.setKey, kv.setVal, kv.setTTL)
	}
	// kv 非 nil 时不写进程内 map。
	if len(s.online) != 0 {
		t.Fatal("kv 模式不应写进程内 map")
	}
}

func TestMarkOfflineUsesKV(t *testing.T) {
	t.Parallel()
	kv := &fakeKV{}
	s := NewStore(kv, time.Minute)

	if err := s.MarkOffline(context.Background(), 42); err != nil {
		t.Fatalf("标记离线失败: %v", err)
	}
	if kv.delKey != "presence:42" {
		t.Fatalf("Del key 错误: %q", kv.delKey)
	}
}

func TestKVErrorsPropagate(t *testing.T) {
	t.Parallel()
	setErr := errors.New("set boom")
	delErr := errors.New("del boom")
	s := NewStore(&fakeKV{setErr: setErr, delErr: delErr}, time.Minute)

	if err := s.MarkOnline(context.Background(), 1); !errors.Is(err, setErr) {
		t.Fatalf("Set 错误应透传, got %v", err)
	}
	if err := s.MarkOffline(context.Background(), 1); !errors.Is(err, delErr) {
		t.Fatalf("Del 错误应透传, got %v", err)
	}
}
