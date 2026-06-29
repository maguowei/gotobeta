// Package presence 记录用户在线状态（Redis TTL，缺省退化为单机内存）。
package presence

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// KV 是 presence 依赖的通用键值存储（由 infra/cache.RedisKV 实现），nil 时退化进程内。
type KV interface {
	Set(ctx context.Context, key, value string, ttl time.Duration) error
	Del(ctx context.Context, key string) error
}

const keyPrefix = "presence:"

// Store 记录在线状态。kv 为 nil 时使用进程内 map（单机）。
type Store struct {
	kv  KV
	ttl time.Duration

	mu     sync.Mutex
	online map[int64]struct{}
}

// NewStore 创建在线状态存储。
func NewStore(kv KV, ttl time.Duration) *Store {
	if ttl <= 0 {
		ttl = 30 * time.Second
	}
	return &Store{kv: kv, ttl: ttl, online: make(map[int64]struct{})}
}

// MarkOnline 标记用户在线（Redis 写带 TTL 的 key）。
func (s *Store) MarkOnline(ctx context.Context, userID int64) error {
	if s.kv != nil {
		return s.kv.Set(ctx, key(userID), "1", s.ttl)
	}
	s.mu.Lock()
	s.online[userID] = struct{}{}
	s.mu.Unlock()
	return nil
}

// Refresh 续期在线状态 TTL，防止心跳期间 key 过期被误判离线。
// Redis 模式重写带 TTL 的 key；内存模式无 TTL，no-op。
func (s *Store) Refresh(ctx context.Context, userID int64) error {
	if s.kv != nil {
		return s.kv.Set(ctx, key(userID), "1", s.ttl)
	}
	return nil
}

// MarkOffline 标记用户离线。
func (s *Store) MarkOffline(ctx context.Context, userID int64) error {
	if s.kv != nil {
		return s.kv.Del(ctx, key(userID))
	}
	s.mu.Lock()
	delete(s.online, userID)
	s.mu.Unlock()
	return nil
}

func key(userID int64) string { return fmt.Sprintf("%s%d", keyPrefix, userID) }
