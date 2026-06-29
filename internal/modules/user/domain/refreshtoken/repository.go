package refreshtoken

import (
	"context"
	"time"
)

// Repository 定义 refresh token 聚合持久化端口。
type Repository interface {
	Create(ctx context.Context, token *RefreshToken) error
	FindByHash(ctx context.Context, tokenHash string, now time.Time) (*RefreshToken, error)
	Revoke(ctx context.Context, tokenID string, replacedByTokenID string, reason string, now time.Time) error
}
