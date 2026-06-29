package server

import (
	"context"

	"errors"
	"fmt"
	"log/slog"

	entschema "entgo.io/ent/dialect/sql/schema"

	"github.com/maguowei/gotobeta/internal/infra/config"

	"github.com/maguowei/gotobeta/internal/infra/entdb"
)

// migrateLockName 是 MySQL 命名建议锁键，多副本/Job 并发迁移时互斥。
const migrateLockName = "gotobeta_migrate"

// migrateLockTimeoutSeconds 是获取迁移锁的等待上限（秒）。
const migrateLockTimeoutSeconds = 30

func runMigrate(ctx context.Context, cfg *config.Config, logger *slog.Logger) (err error) {
	client, sqlDB, err := entdb.NewEntClient(&cfg.Database)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := client.Close(); closeErr != nil {
			err = errors.Join(err, fmt.Errorf("close ent client: %w", closeErr))
		}
	}()
	defer func() {
		if closeErr := sqlDB.Close(); closeErr != nil {
			err = errors.Join(err, fmt.Errorf("close sql db: %w", closeErr))
		}
	}()

	// 取一条独占连接持有 MySQL 命名锁：多实例并发迁移时仅一个执行 DDL，
	// 避免 Ent 自动迁移并发产生重复建表/加索引冲突。
	conn, err := sqlDB.Conn(ctx)
	if err != nil {
		return fmt.Errorf("acquire migrate conn: %w", err)
	}
	defer func() {
		if closeErr := conn.Close(); closeErr != nil {
			err = errors.Join(err, fmt.Errorf("close migrate conn: %w", closeErr))
		}
	}()

	var locked int
	if scanErr := conn.QueryRowContext(ctx, "SELECT GET_LOCK(?, ?)", migrateLockName, migrateLockTimeoutSeconds).Scan(&locked); scanErr != nil {
		return fmt.Errorf("acquire migrate lock: %w", scanErr)
	}
	if locked != 1 {
		// 未拿到锁：其他实例正在迁移，跳过（幂等）。
		logger.WarnContext(ctx, "未获取迁移锁，跳过迁移", slog.String("lock", migrateLockName))
		return nil
	}
	defer func() {
		// 用同一连接释放锁；连接关闭也会自动释放，这里显式释放更清晰。
		if _, relErr := conn.ExecContext(context.WithoutCancel(ctx), "SELECT RELEASE_LOCK(?)", migrateLockName); relErr != nil {
			err = errors.Join(err, fmt.Errorf("release migrate lock: %w", relErr))
		}
	}()

	return client.Schema.Create(ctx, entschema.WithForeignKeys(false))
}
