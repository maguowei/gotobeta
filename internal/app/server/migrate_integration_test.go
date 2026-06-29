//go:build integration

package server

import (
	"context"
	"log/slog"
	"sync"
	"testing"

	"github.com/maguowei/gotobeta/internal/infra/config"
	"github.com/maguowei/gotobeta/internal/testutil"
)

// TestRunMigrateConcurrentWithLock 验证并发迁移在 MySQL 命名锁下被串行化：
// 两个 goroutine 同时迁移同一库，均应成功（Ent 自动迁移幂等），不因并发 DDL 冲突报错。
func TestRunMigrateConcurrentWithLock(t *testing.T) {
	ctx := context.Background()
	mysqlC := testutil.StartMySQL(ctx, t)

	cfg := &config.Config{
		Database: config.DatabaseConfig{
			Driver: "mysql",
			DSN:    mysqlC.DSN,
		},
	}
	logger := slog.New(slog.NewTextHandler(testWriter{t}, nil))

	const n = 3
	var wg sync.WaitGroup
	errs := make([]error, n)
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func(idx int) {
			defer wg.Done()
			errs[idx] = runMigrate(ctx, cfg, logger)
		}(i)
	}
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Fatalf("并发迁移 #%d 失败: %v", i, err)
		}
	}

	// 锁释放后再次迁移仍应成功（幂等）。
	if err := runMigrate(ctx, cfg, logger); err != nil {
		t.Fatalf("二次迁移失败: %v", err)
	}
}

// testWriter 把日志转发到测试输出。
type testWriter struct{ t *testing.T }

func (w testWriter) Write(p []byte) (int, error) {
	w.t.Logf("%s", p)
	return len(p), nil
}
