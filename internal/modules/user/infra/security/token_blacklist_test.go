package security

import (
	"context"
	"testing"
	"time"
)

// fakeKV 是内存版 blacklistKV，记录写入并支持 TTL<=0 不写入语义由调用方保证。
type fakeKV struct {
	store map[string]string
}

func newFakeKV() *fakeKV { return &fakeKV{store: map[string]string{}} }

func (k *fakeKV) Set(_ context.Context, key, value string, _ time.Duration) error {
	k.store[key] = value
	return nil
}

func (k *fakeKV) Get(_ context.Context, key string) (string, bool, error) {
	v, ok := k.store[key]
	return v, ok, nil
}

func TestTokenBlacklistRevokeThenRevoked(t *testing.T) {
	bl := NewTokenBlacklist(newFakeKV())
	jti := "abc123"

	revoked, err := bl.IsRevoked(context.Background(), jti)
	if err != nil {
		t.Fatalf("IsRevoked error = %v", err)
	}
	if revoked {
		t.Fatal("吊销前不应命中黑名单")
	}

	if err := bl.Revoke(context.Background(), jti, time.Minute); err != nil {
		t.Fatalf("Revoke error = %v", err)
	}

	revoked, err = bl.IsRevoked(context.Background(), jti)
	if err != nil {
		t.Fatalf("IsRevoked error = %v", err)
	}
	if !revoked {
		t.Fatal("吊销后应命中黑名单")
	}
}

func TestTokenBlacklistRevokeExpiredIsNoop(t *testing.T) {
	kv := newFakeKV()
	bl := NewTokenBlacklist(kv)
	if err := bl.Revoke(context.Background(), "expired", -time.Second); err != nil {
		t.Fatalf("Revoke error = %v", err)
	}
	if len(kv.store) != 0 {
		t.Fatal("剩余有效期<=0 时不应写入黑名单")
	}
}

func TestTokenBlacklistNilKVDegrades(t *testing.T) {
	bl := NewTokenBlacklist(nil)
	if err := bl.Revoke(context.Background(), "x", time.Minute); err != nil {
		t.Fatalf("nil KV 下 Revoke 应降级无错: %v", err)
	}
	revoked, err := bl.IsRevoked(context.Background(), "x")
	if err != nil {
		t.Fatalf("nil KV 下 IsRevoked 应降级无错: %v", err)
	}
	if revoked {
		t.Fatal("黑名单不可用时应 fail-open 返回未吊销")
	}
}
