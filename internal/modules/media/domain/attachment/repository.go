package attachment

import "context"

// Repository 定义附件聚合的仓储接口。
type Repository interface {
	Create(ctx context.Context, a *Attachment) error
	FindByID(ctx context.Context, id int64) (*Attachment, error)
	Save(ctx context.Context, a *Attachment) error
}
