// Package persistence 是 workspace 模块的仓储 Ent 实现。
package persistence

import (
	"context"
	"log/slog"

	"github.com/maguowei/gotobeta/internal/ent"
	entworkspace "github.com/maguowei/gotobeta/internal/ent/workspace"
	entmember "github.com/maguowei/gotobeta/internal/ent/workspacemember"
	"github.com/maguowei/gotobeta/internal/infra/entdb"
	"github.com/maguowei/gotobeta/internal/modules/workspace/domain/workspace"
)

// WorkspaceRepository 是工作区仓储的 Ent 实现。
type WorkspaceRepository struct {
	client *ent.Client
	logger *slog.Logger
}

// NewWorkspaceRepository 创建仓储。
func NewWorkspaceRepository(client *ent.Client, logger *slog.Logger) *WorkspaceRepository {
	return &WorkspaceRepository{client: client, logger: logger}
}

// Create 保存工作区。
func (r *WorkspaceRepository) Create(ctx context.Context, w *workspace.Workspace) error {
	client := entdb.ClientFromCtx(ctx, r.client)
	_, err := client.Workspace.Create().
		SetBizID(w.ID()).
		SetSlug(w.Slug()).
		SetName(w.Name()).
		SetOwnerUserID(w.OwnerUserID()).
		SetStatus(int8(w.Status())).
		SetSettings(w.Settings()).
		SetCreatedAt(w.CreatedAt()).
		SetUpdatedAt(w.UpdatedAt()).
		Save(ctx)
	if err != nil {
		if ent.IsConstraintError(err) {
			return workspace.ErrSlugTaken
		}
		return err
	}
	return nil
}

// FindByID 按业务 ID 查找。
func (r *WorkspaceRepository) FindByID(ctx context.Context, id int64) (*workspace.Workspace, error) {
	client := entdb.ClientFromCtx(ctx, r.client)
	row, err := client.Workspace.Query().Where(entworkspace.BizID(id)).Only(ctx)
	if err != nil {
		return nil, mapEntNotFound(err, workspace.ErrNotFound)
	}
	return workspaceToEntity(row), nil
}

// FindBySlug 按 slug 查找。
func (r *WorkspaceRepository) FindBySlug(ctx context.Context, slug string) (*workspace.Workspace, error) {
	client := entdb.ClientFromCtx(ctx, r.client)
	row, err := client.Workspace.Query().Where(entworkspace.Slug(slug)).Only(ctx)
	if err != nil {
		return nil, mapEntNotFound(err, workspace.ErrNotFound)
	}
	return workspaceToEntity(row), nil
}

// Save 更新工作区。
func (r *WorkspaceRepository) Save(ctx context.Context, w *workspace.Workspace) error {
	client := entdb.ClientFromCtx(ctx, r.client)
	affected, err := client.Workspace.Update().
		Where(entworkspace.BizID(w.ID())).
		SetName(w.Name()).
		SetStatus(int8(w.Status())).
		SetSettings(w.Settings()).
		SetUpdatedAt(w.UpdatedAt()).
		Save(ctx)
	if err != nil {
		return err
	}
	if affected == 0 {
		return workspace.ErrNotFound
	}
	return nil
}

// ListByMemberUser 返回用户加入的全部工作区（经成员表关联）。
func (r *WorkspaceRepository) ListByMemberUser(ctx context.Context, userID int64) ([]*workspace.Workspace, error) {
	client := entdb.ClientFromCtx(ctx, r.client)
	memberRows, err := client.WorkspaceMember.Query().
		Where(entmember.UserID(userID), entmember.StatusEQ(1)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	if len(memberRows) == 0 {
		return []*workspace.Workspace{}, nil
	}
	wsIDs := make([]int64, 0, len(memberRows))
	for _, m := range memberRows {
		wsIDs = append(wsIDs, m.WorkspaceID)
	}
	rows, err := client.Workspace.Query().
		Where(entworkspace.BizIDIn(wsIDs...)).
		Order(ent.Asc(entworkspace.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	items := make([]*workspace.Workspace, 0, len(rows))
	for _, row := range rows {
		items = append(items, workspaceToEntity(row))
	}
	return items, nil
}

func workspaceToEntity(row *ent.Workspace) *workspace.Workspace {
	return workspace.UnmarshalFromDB(
		row.BizID, row.Slug, row.Name, row.OwnerUserID,
		workspace.Status(row.Status), row.Settings, row.CreatedAt, row.UpdatedAt,
	)
}
