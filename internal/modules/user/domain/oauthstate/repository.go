package oauthstate

import (
	"context"
	"time"
)

// Repository 定义 OAuth state 聚合持久化端口。
type Repository interface {
	Create(ctx context.Context, state *OAuthState) error
	Consume(ctx context.Context, provider string, stateHash string, now time.Time) (*OAuthState, error)
}
