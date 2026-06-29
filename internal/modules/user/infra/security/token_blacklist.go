package security

import (
	"context"
	"time"
)

// blacklistKV 是吊销黑名单依赖的键值存储（由 infra/cache.RedisKV 实现）。
// nil 时黑名单降级为不可用：写入 no-op、查询恒返回未吊销。
type blacklistKV interface {
	Set(ctx context.Context, key, value string, ttl time.Duration) error
	Get(ctx context.Context, key string) (value string, found bool, err error)
}

const revokedKeyPrefix = "jwt:revoked:"

// TokenBlacklist 基于 Redis 维护已吊销 access token 的 jti 集合。
// 每个 jti 的 key 仅保留到 token 自然过期，过期后自动清理，避免无限增长。
type TokenBlacklist struct {
	kv blacklistKV
}

// NewTokenBlacklist 创建吊销黑名单；kv 为 nil 时黑名单不可用（吊销降级）。
func NewTokenBlacklist(kv blacklistKV) *TokenBlacklist {
	return &TokenBlacklist{kv: kv}
}

// Revoke 把 jti 加入黑名单，TTL 为 token 剩余有效期；ttl<=0 表示已过期，无需记录。
func (b *TokenBlacklist) Revoke(ctx context.Context, jti string, ttl time.Duration) error {
	if b == nil || b.kv == nil || jti == "" || ttl <= 0 {
		return nil
	}
	return b.kv.Set(ctx, revokedKeyPrefix+jti, "1", ttl)
}

// IsRevoked 查询 jti 是否已被吊销；黑名单不可用时返回未吊销（fail-open，避免误杀正常请求）。
func (b *TokenBlacklist) IsRevoked(ctx context.Context, jti string) (bool, error) {
	if b == nil || b.kv == nil || jti == "" {
		return false, nil
	}
	_, found, err := b.kv.Get(ctx, revokedKeyPrefix+jti)
	if err != nil {
		return false, err
	}
	return found, nil
}
