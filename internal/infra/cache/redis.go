package cache

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/maguowei/gotobeta/internal/infra/config"
	"github.com/maguowei/gotobeta/internal/pkg/health"
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

// RedisHealthChecker 返回探活 Redis 的健康检查器（PING），供 readyz 注册。
func RedisHealthChecker(client *redis.Client) health.Checker {
	return health.CheckerFunc(func(ctx context.Context) error {
		return PingRedis(ctx, client)
	})
}

// CloseRedis 关闭 Redis client。
func CloseRedis(client *redis.Client) error {
	if client == nil {
		return nil
	}
	return client.Close()
}
