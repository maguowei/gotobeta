package cache

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisKV 是基于 go-redis 的通用键值封装，供需要简单 KV/TTL 语义的业务模块复用，
// 避免业务模块直接 import go-redis（SDK 归口约束）。
type RedisKV struct {
	client *redis.Client
}

// NewRedisKV 包装一个 Redis client；client 为 nil 时返回 nil。
func NewRedisKV(client *redis.Client) *RedisKV {
	if client == nil {
		return nil
	}
	return &RedisKV{client: client}
}

// Set 写入键值并设置 TTL。
func (k *RedisKV) Set(ctx context.Context, key, value string, ttl time.Duration) error {
	return k.client.Set(ctx, key, value, ttl).Err()
}

// Del 删除键（不存在视为成功）。
func (k *RedisKV) Del(ctx context.Context, key string) error {
	return k.client.Del(ctx, key).Err()
}

// GetDel 原子读取并删除键；键不存在时 found=false。
func (k *RedisKV) GetDel(ctx context.Context, key string) (value string, found bool, err error) {
	val, err := k.client.GetDel(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return val, true, nil
}
