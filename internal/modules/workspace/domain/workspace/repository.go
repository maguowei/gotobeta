package workspace

import "context"

// Repository 定义 Workspace 聚合的仓储接口。
type Repository interface {
	Create(ctx context.Context, w *Workspace) error
	FindByID(ctx context.Context, id int64) (*Workspace, error)
	FindBySlug(ctx context.Context, slug string) (*Workspace, error)
	Save(ctx context.Context, w *Workspace) error
	// ListByMemberUser 返回某用户加入的全部工作区。
	ListByMemberUser(ctx context.Context, userID int64) ([]*Workspace, error)
}
