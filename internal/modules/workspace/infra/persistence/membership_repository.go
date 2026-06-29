package persistence

import (
	"context"
	"log/slog"

	"github.com/maguowei/gotobeta/internal/ent"
	entmember "github.com/maguowei/gotobeta/internal/ent/workspacemember"
	"github.com/maguowei/gotobeta/internal/infra/entdb"
	"github.com/maguowei/gotobeta/internal/modules/workspace/domain/membership"
)

// MembershipRepository 是成员关系仓储的 Ent 实现。
type MembershipRepository struct {
	client *ent.Client
	logger *slog.Logger
}

// NewMembershipRepository 创建仓储。
func NewMembershipRepository(client *ent.Client, logger *slog.Logger) *MembershipRepository {
	return &MembershipRepository{client: client, logger: logger}
}

// Add 新增成员关系。
func (r *MembershipRepository) Add(ctx context.Context, m *membership.Member) error {
	client := entdb.ClientFromCtx(ctx, r.client)
	_, err := client.WorkspaceMember.Create().
		SetBizID(m.ID()).
		SetWorkspaceID(m.WorkspaceID()).
		SetUserID(m.UserID()).
		SetStatus(int8(m.Status())).
		SetJoinedAt(m.JoinedAt()).
		SetCreatedAt(m.CreatedAt()).
		SetUpdatedAt(m.UpdatedAt()).
		Save(ctx)
	if err != nil {
		if ent.IsConstraintError(err) {
			return membership.ErrAlreadyMember
		}
		return err
	}
	return nil
}

// FindByWorkspaceUser 查找成员关系。
func (r *MembershipRepository) FindByWorkspaceUser(ctx context.Context, workspaceID, userID int64) (*membership.Member, error) {
	client := entdb.ClientFromCtx(ctx, r.client)
	row, err := client.WorkspaceMember.Query().
		Where(entmember.WorkspaceID(workspaceID), entmember.UserID(userID)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, membership.ErrNotFound
		}
		return nil, err
	}
	return memberToEntity(row), nil
}

// ListByUser 返回用户的全部成员关系。
func (r *MembershipRepository) ListByUser(ctx context.Context, userID int64) ([]*membership.Member, error) {
	client := entdb.ClientFromCtx(ctx, r.client)
	rows, err := client.WorkspaceMember.Query().Where(entmember.UserID(userID)).All(ctx)
	if err != nil {
		return nil, err
	}
	return membersToEntities(rows), nil
}

// ListByWorkspace 返回工作区的全部成员关系。
func (r *MembershipRepository) ListByWorkspace(ctx context.Context, workspaceID int64) ([]*membership.Member, error) {
	client := entdb.ClientFromCtx(ctx, r.client)
	rows, err := client.WorkspaceMember.Query().Where(entmember.WorkspaceID(workspaceID)).All(ctx)
	if err != nil {
		return nil, err
	}
	return membersToEntities(rows), nil
}

func memberToEntity(row *ent.WorkspaceMember) *membership.Member {
	return membership.UnmarshalFromDB(
		row.BizID, row.WorkspaceID, row.UserID,
		membership.Status(row.Status), row.JoinedAt, row.CreatedAt, row.UpdatedAt,
	)
}

func membersToEntities(rows []*ent.WorkspaceMember) []*membership.Member {
	items := make([]*membership.Member, 0, len(rows))
	for _, row := range rows {
		items = append(items, memberToEntity(row))
	}
	return items
}
