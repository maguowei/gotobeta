// Package ticket 实现 WS 鉴权 ticket 存储（Redis，缺省退化为进程内 TTL map）。
package ticket

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"
)

// ErrInvalidTicket 表示 ticket 无效、过期或已被消费。
var ErrInvalidTicket = errors.New("ticket: invalid or expired")

const keyPrefix = "ws:ticket:"

// KV 是 ticket 依赖的通用键值存储（由 infra/cache.RedisKV 实现），nil 时退化为进程内。
type KV interface {
	Set(ctx context.Context, key, value string, ttl time.Duration) error
	GetDel(ctx context.Context, key string) (value string, found bool, err error)
}

// Store 实现 port.TicketStore。kv 为 nil 时使用进程内带 TTL 的 map。
type Store struct {
	kv  KV
	ttl time.Duration

	mu  sync.Mutex
	mem map[string]memEntry
}

type memEntry struct {
	userID    int64
	expiresAt time.Time
}

// NewStore 创建 ticket 存储。kv 为 nil 时退化为单机内存实现。
func NewStore(kv KV, ttl time.Duration) *Store {
	if ttl <= 0 {
		ttl = 30 * time.Second
	}
	return &Store{kv: kv, ttl: ttl, mem: make(map[string]memEntry)}
}

// Issue 签发一次性 ticket。
func (s *Store) Issue(ctx context.Context, userID int64) (string, error) {
	token, err := randomToken()
	if err != nil {
		return "", err
	}
	if s.kv != nil {
		if err := s.kv.Set(ctx, keyPrefix+token, strconv.FormatInt(userID, 10), s.ttl); err != nil {
			return "", fmt.Errorf("ticket: 写入 Redis: %w", err)
		}
		return token, nil
	}
	s.mu.Lock()
	s.mem[token] = memEntry{userID: userID, expiresAt: time.Now().Add(s.ttl)}
	s.mu.Unlock()
	return token, nil
}

// Consume 校验并消费 ticket。
func (s *Store) Consume(ctx context.Context, token string) (int64, error) {
	if token == "" {
		return 0, ErrInvalidTicket
	}
	if s.kv != nil {
		val, found, err := s.kv.GetDel(ctx, keyPrefix+token)
		if err != nil {
			return 0, fmt.Errorf("ticket: 读取 Redis: %w", err)
		}
		if !found {
			return 0, ErrInvalidTicket
		}
		userID, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return 0, ErrInvalidTicket
		}
		return userID, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, ok := s.mem[token]
	if !ok {
		return 0, ErrInvalidTicket
	}
	delete(s.mem, token)
	if time.Now().After(entry.expiresAt) {
		return 0, ErrInvalidTicket
	}
	return entry.userID, nil
}

func randomToken() (string, error) {
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("ticket: 生成随机数: %w", err)
	}
	return hex.EncodeToString(buf), nil
}
