package persistence

import (
	"context"
	"log/slog"
	"time"

	"github.com/maguowei/gotobeta/internal/ent"
	entacl "github.com/maguowei/gotobeta/internal/ent/rbacaclentry"
	"github.com/maguowei/gotobeta/internal/infra/entdb"
	"github.com/maguowei/gotobeta/internal/modules/workspace/domain/acl"
)

// ACLRepository 是 ACL 仓储的 Ent 实现。
type ACLRepository struct {
	client *ent.Client
	logger *slog.Logger
}

// NewACLRepository 创建仓储。
func NewACLRepository(client *ent.Client, logger *slog.Logger) *ACLRepository {
	return &ACLRepository{client: client, logger: logger}
}

// Grant 新增 ACL 例外授权。
func (r *ACLRepository) Grant(ctx context.Context, e *acl.Entry) error {
	client := entdb.ClientFromCtx(ctx, r.client)
	builder := client.RbacAclEntry.Create().
		SetBizID(e.ID()).
		SetWorkspaceID(e.WorkspaceID()).
		SetSubjectType(e.SubjectType()).
		SetSubjectID(e.SubjectID()).
		SetResourceType(e.ResourceType()).
		SetResourceID(e.ResourceID()).
		SetActionCode(e.ActionCode()).
		SetEffect(int8(e.Effect())).
		SetReason(e.Reason()).
		SetSourceType(e.SourceType()).
		SetCreatedBy(e.CreatedBy())
	if exp := e.ExpiresAt(); exp != nil {
		builder.SetExpiresAt(*exp)
	}
	return builder.Exec(ctx)
}

// Revoke 删除 ACL 授权。
func (r *ACLRepository) Revoke(ctx context.Context, id int64) error {
	client := entdb.ClientFromCtx(ctx, r.client)
	affected, err := client.RbacAclEntry.Delete().Where(entacl.BizID(id)).Exec(ctx)
	if err != nil {
		return err
	}
	if affected == 0 {
		return acl.ErrNotFound
	}
	return nil
}

// FindDecisive 返回作用于 (用户或其角色, 资源, 动作) 的有效裁决条目，拒绝优先。
func (r *ACLRepository) FindDecisive(ctx context.Context, workspaceID, userID int64, roleIDs []int64, resourceType, resourceID, actionCode string, now time.Time) (*acl.Entry, error) {
	client := entdb.ClientFromCtx(ctx, r.client)

	subjectMatch := entacl.Or(
		entacl.And(entacl.SubjectTypeEQ(acl.SubjectUser), entacl.SubjectID(userID)),
		entacl.And(entacl.SubjectTypeEQ(acl.SubjectRole), entacl.SubjectIDIn(roleIDs...)),
	)
	rows, err := client.RbacAclEntry.Query().
		Where(
			entacl.WorkspaceID(workspaceID),
			entacl.ResourceType(resourceType),
			entacl.ResourceID(resourceID),
			entacl.ActionCode(actionCode),
			entacl.Or(entacl.ExpiresAtIsNil(), entacl.ExpiresAtGT(now)),
			subjectMatch,
		).
		All(ctx)
	if err != nil {
		return nil, err
	}

	var decisive *acl.Entry
	for _, row := range rows {
		entry := aclToEntity(row)
		if entry.Effect() == acl.EffectDeny {
			return entry, nil // 拒绝优先，立即返回
		}
		if decisive == nil {
			decisive = entry
		}
	}
	return decisive, nil
}

func aclToEntity(row *ent.RbacAclEntry) *acl.Entry {
	return acl.UnmarshalEntry(
		row.BizID, row.WorkspaceID, row.SubjectType, row.SubjectID,
		row.ResourceType, row.ResourceID, row.ActionCode, acl.Effect(row.Effect),
		row.Reason, row.SourceType, row.ExpiresAt, row.CreatedBy, row.CreatedAt,
	)
}
