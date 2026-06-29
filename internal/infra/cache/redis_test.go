package cache

import (
	"context"
	"testing"
	"time"

	"github.com/maguowei/gotobeta/internal/infra/config"
)

func TestNewRedisClientReturnsNilWhenDisabled(t *testing.T) {
	client, err := NewRedisClient(config.RedisConfig{Enabled: false})
	if err != nil {
		t.Fatalf("NewRedisClient() error = %v", err)
	}
	if client != nil {
		t.Fatalf("client = %v, want nil when disabled", client)
	}
}

func TestRedisHealthCheckerReportsError(t *testing.T) {
	// 指向不可达地址 + 短 dial 超时：探活应快速返回错误（不挂起）。
	client, err := NewRedisClient(config.RedisConfig{
		Enabled:      true,
		Addr:         "127.0.0.1:1",
		DialTimeout:  "100ms",
		ReadTimeout:  "100ms",
		WriteTimeout: "100ms",
	})
	if err != nil {
		t.Fatalf("NewRedisClient() error = %v", err)
	}
	t.Cleanup(func() { _ = CloseRedis(client) })

	checker := RedisHealthChecker(client)
	if checker == nil {
		t.Fatal("checker 不应为 nil")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := checker.Check(ctx); err == nil {
		t.Fatal("不可达 Redis 探活应返回错误")
	}
}

func TestNewRedisClientUsesConfig(t *testing.T) {
	client, err := NewRedisClient(config.RedisConfig{
		Enabled:      true,
		Addr:         "127.0.0.1:6379",
		Password:     "secret",
		DB:           2,
		DialTimeout:  "5s",
		ReadTimeout:  "3s",
		WriteTimeout: "2s",
	})
	if err != nil {
		t.Fatalf("NewRedisClient() error = %v", err)
	}
	t.Cleanup(func() { _ = CloseRedis(client) })

	opts := client.Options()
	if opts.Addr != "127.0.0.1:6379" || opts.Password != "secret" || opts.DB != 2 {
		t.Fatalf("redis options = %+v", opts)
	}
	if opts.DialTimeout != 5*time.Second || opts.ReadTimeout != 3*time.Second || opts.WriteTimeout != 2*time.Second {
		t.Fatalf("timeouts = %v/%v/%v", opts.DialTimeout, opts.ReadTimeout, opts.WriteTimeout)
	}
}
