package sentry

import (
	"errors"
	"fmt"
	"time"

	sentrysdk "github.com/getsentry/sentry-go"

	"github.com/maguowei/gotobeta/internal/infra/config"
)

// Init 初始化 Sentry。
func Init(cfg *config.SentryConfig) error {
	if cfg == nil || !cfg.Enabled {
		return nil
	}

	if cfg.DSN == "" {
		return errors.New("sentry DSN is required when enabled")
	}

	if err := sentrysdk.Init(sentrysdk.ClientOptions{
		Dsn:         cfg.DSN,
		Environment: cfg.Env,
	}); err != nil {
		return fmt.Errorf("initialize sentry: %w", err)
	}

	return nil
}

// Flush 刷新缓冲区。返回 true 表示在 timeout 内全部刷写完毕；
// 返回 false 表示超时，缓冲事件可能丢失，调用方应记录或聚合为错误。
func Flush() bool {
	return sentrysdk.Flush(2 * time.Second)
}
