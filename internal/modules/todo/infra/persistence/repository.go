package persistence

import (
	"context"
	"log/slog"

	"github.com/maguowei/gotobeta/internal/ent"
	enttodo "github.com/maguowei/gotobeta/internal/ent/todo"
	"github.com/maguowei/gotobeta/internal/infra/entdb"
	"github.com/maguowei/gotobeta/internal/modules/todo/domain/todo"
)

// TodoRepository 是 Todo 聚合仓储的 Ent 实现。
type TodoRepository struct {
	client *ent.Client
	logger *slog.Logger
}

// NewTodoRepository 创建仓储。
func NewTodoRepository(client *ent.Client, logger *slog.Logger) *TodoRepository {
	return &TodoRepository{client: client, logger: logger}
}

// Create 保存 Todo。
func (r *TodoRepository) Create(ctx context.Context, t *todo.Todo) error {
	client := entdb.ClientFromCtx(ctx, r.client)
	_, err := client.Todo.
		Create().
		SetBizID(t.ID()).
		SetTitle(t.Title().String()).
		SetStatus(string(t.Status())).
		SetVersion(t.Version()).
		SetCreatedAt(t.CreatedAt()).
		SetUpdatedAt(t.UpdatedAt()).
		Save(ctx)

	return err
}

// FindByID 按业务 ID 查找。
func (r *TodoRepository) FindByID(ctx context.Context, id int64) (*todo.Todo, error) {
	client := entdb.ClientFromCtx(ctx, r.client)
	row, err := client.Todo.Query().Where(enttodo.BizID(id)).Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, todo.ErrNotFound
		}
		return nil, err
	}
	return toEntity(row), nil
}

// Save 更新 Todo。
func (r *TodoRepository) Save(ctx context.Context, t *todo.Todo) error {
	client := entdb.ClientFromCtx(ctx, r.client)
	affected, err := client.Todo.
		Update().
		Where(enttodo.BizID(t.ID()), enttodo.Version(t.Version())).
		SetTitle(t.Title().String()).
		SetStatus(string(t.Status())).
		SetVersion(t.Version() + 1).
		SetUpdatedAt(t.UpdatedAt()).
		Save(ctx)
	if err != nil {
		return err
	}
	if affected == 0 {
		// 版本条件未命中：记录不存在或已被并发修改，用 Exist 区分两种语义。
		exists, existErr := client.Todo.Query().Where(enttodo.BizID(t.ID())).Exist(ctx)
		if existErr != nil {
			return existErr
		}
		if exists {
			return todo.ErrConflict
		}
		return todo.ErrNotFound
	}

	return nil
}

// Delete 按业务 ID 删除。
func (r *TodoRepository) Delete(ctx context.Context, id int64) error {
	client := entdb.ClientFromCtx(ctx, r.client)
	affected, err := client.Todo.Delete().Where(enttodo.BizID(id)).Exec(ctx)
	if err != nil {
		return err
	}
	if affected == 0 {
		return todo.ErrNotFound
	}

	return nil
}

// List 返回 Todo 列表。
func (r *TodoRepository) List(ctx context.Context) ([]*todo.Todo, error) {
	client := entdb.ClientFromCtx(ctx, r.client)
	rows, err := client.Todo.Query().Order(enttodo.ByCreatedAt()).All(ctx)
	if err != nil {
		return nil, err
	}

	items := make([]*todo.Todo, 0, len(rows))
	for _, row := range rows {
		items = append(items, toEntity(row))
	}

	return items, nil
}

func toEntity(row *ent.Todo) *todo.Todo {
	return todo.UnmarshalFromDB(
		row.BizID,
		row.Title,
		todo.Status(row.Status),
		row.Version,
		row.CreatedAt,
		row.UpdatedAt,
	)
}
