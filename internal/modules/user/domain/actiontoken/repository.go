package actiontoken

import (
	"context"
	"time"
)

// Repository 定义一次性动作 token 聚合持久化端口。
type Repository interface {
	Create(ctx context.Context, token *ActionToken) error
	Consume(ctx context.Context, tokenHash string, purpose string, now time.Time) (*ActionToken, error)
}
