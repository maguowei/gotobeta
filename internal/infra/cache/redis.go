package cache

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/maguowei/gotobeta/internal/infra/config"
)

// NewRedisClient 创建 Redis client。redis.enabled=false 时返回 nil。
func NewRedisClient(cfg config.RedisConfig) (*redis.Client, error) {
	if !cfg.Enabled {
		return nil, nil
	}
	dialTimeout, err := time.ParseDuration(cfg.DialTimeout)
	if err != nil {
		return nil, err
	}
	readTimeout, err := time.ParseDuration(cfg.ReadTimeout)
	if err != nil {
		return nil, err
	}
	writeTimeout, err := time.ParseDuration(cfg.WriteTimeout)
	if err != nil {
		return nil, err
	}
	return redis.NewClient(&redis.Options{
		Addr:         cfg.Addr,
		Password:     cfg.Password,
		DB:           cfg.DB,
		DialTimeout:  dialTimeout,
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
	}), nil
}

// PingRedis 检查 Redis 连接是否可用。
func PingRedis(ctx context.Context, client *redis.Client) error {
	if client == nil {
		return nil
	}
	return client.Ping(ctx).Err()
}

// CloseRedis 关闭 Redis client。
func CloseRedis(client *redis.Client) error {
	if client == nil {
		return nil
	}
	return client.Close()
}
