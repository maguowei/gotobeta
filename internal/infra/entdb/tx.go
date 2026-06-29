package entdb

import (
	"context"
	"errors"
	"fmt"

	"github.com/maguowei/gotobeta/internal/ent"
)

type txKey struct{}

// EntTxRunner 提供基于 Ent 的事务执行能力。
type EntTxRunner struct {
	client *ent.Client
}

func NewEntTxRunner(client *ent.Client) *EntTxRunner {
	return &EntTxRunner{client: client}
}

// RunInTx 在事务中执行 fn，提交或回滚由结果决定。
func (r *EntTxRunner) RunInTx(ctx context.Context, fn func(ctx context.Context) error) (err error) {
	tx, err := r.client.Tx(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	rollbackNeeded := true
	defer func() {
		if !rollbackNeeded {
			return
		}

		if recovered := recover(); recovered != nil {
			if rbErr := tx.Rollback(); rbErr != nil {
				err = errors.Join(err, fmt.Errorf("rollback tx: %w", rbErr))
			}
			panic(recovered)
		}

		if err != nil {
			if rbErr := tx.Rollback(); rbErr != nil {
				err = errors.Join(err, fmt.Errorf("rollback tx: %w", rbErr))
			}
		}
	}()

	txCtx := context.WithValue(ctx, txKey{}, tx)
	if err = fn(txCtx); err != nil {
		return err
	}

	rollbackNeeded = false
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}

	return nil
}

// ClientFromCtx 从 context 获取事务客户端，若无事务则返回 fallback。
func ClientFromCtx(ctx context.Context, fallback *ent.Client) *ent.Client {
	if tx, ok := ctx.Value(txKey{}).(*ent.Tx); ok {
		return tx.Client()
	}
	return fallback
}
