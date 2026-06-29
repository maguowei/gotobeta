package identity

import "context"

// Repository 定义第三方身份聚合持久化端口。
type Repository interface {
	Find(ctx context.Context, provider string, providerUserID string) (*Identity, error)
	Upsert(ctx context.Context, ident *Identity) error
	List(ctx context.Context, userID int64) ([]*Identity, error)
	Delete(ctx context.Context, userID int64, provider string) error
}
