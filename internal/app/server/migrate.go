package server

import (
	"context"

	"errors"
	"fmt"

	entschema "entgo.io/ent/dialect/sql/schema"

	"github.com/maguowei/gotobeta/internal/infra/config"

	"github.com/maguowei/gotobeta/internal/infra/entdb"
)

func runMigrate(ctx context.Context, cfg *config.Config) (err error) {
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

	return client.Schema.Create(ctx, entschema.WithForeignKeys(false))
}
