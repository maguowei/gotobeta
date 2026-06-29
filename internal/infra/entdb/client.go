package entdb

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"

	entsql "entgo.io/ent/dialect/sql"

	"github.com/maguowei/gotobeta/internal/ent"
	"github.com/maguowei/gotobeta/internal/infra/config"
)

const startupPingTimeout = 5 * time.Second

// NewEntClient 创建 Ent 客户端。
func NewEntClient(cfg *config.DatabaseConfig) (*ent.Client, *sql.DB, error) {
	dsn, err := ensureMySQLTimeZone(cfg.DSN)
	if err != nil {
		return nil, nil, err
	}

	db, err := sql.Open(cfg.Driver, dsn)
	if err != nil {
		return nil, nil, fmt.Errorf("open database: %w", err)
	}

	if cfg.MaxOpenConns > 0 {
		db.SetMaxOpenConns(cfg.MaxOpenConns)
	}
	if cfg.MaxIdleConns > 0 {
		db.SetMaxIdleConns(cfg.MaxIdleConns)
	}
	if cfg.ConnMaxLifetime > 0 {
		db.SetConnMaxLifetime(time.Duration(cfg.ConnMaxLifetime) * time.Second)
	}
	if cfg.ConnMaxIdleTime > 0 {
		db.SetConnMaxIdleTime(time.Duration(cfg.ConnMaxIdleTime) * time.Second)
	}

	ctx, cancel := context.WithTimeout(context.Background(), startupPingTimeout)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, nil, fmt.Errorf("ping database: %w", err)
	}

	driver := entsql.OpenDB(cfg.Driver, db)
	client := ent.NewClient(ent.Driver(driver))
	// DataScope 兜底：查询层统一注入 workspace_id 过滤（详见 WorkspaceScopeInterceptor）。
	client.Intercept(WorkspaceScopeInterceptor())

	return client, db, nil
}

func ensureMySQLTimeZone(dsn string) (string, error) {
	params := url.Values{}
	baseDSN := dsn

	if strings.Contains(dsn, "?") {
		parts := strings.SplitN(dsn, "?", 2)
		baseDSN = parts[0]
		parsed, err := url.ParseQuery(parts[1])
		if err != nil {
			return "", err
		}
		params = parsed
	}

	if params.Get("parseTime") == "" {
		params.Set("parseTime", "true")
	}

	if params.Get("loc") == "" {
		params.Set("loc", "Local")
	}

	return baseDSN + "?" + params.Encode(), nil
}
