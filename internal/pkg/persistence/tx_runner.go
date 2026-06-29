package persistence

import "context"

// TxRunner 提供数据库事务执行能力（由基础设施层实现，注入应用层）。
type TxRunner interface {
	RunInTx(ctx context.Context, fn func(ctx context.Context) error) error
}
