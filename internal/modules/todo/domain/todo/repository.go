package todo

import "context"

// Repository 定义 Todo 聚合的仓储接口。
type Repository interface {
	Create(ctx context.Context, todo *Todo) error
	FindByID(ctx context.Context, id int64) (*Todo, error)
	Save(ctx context.Context, todo *Todo) error
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context) ([]*Todo, error)
}
