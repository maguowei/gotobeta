package persistence

import (
	"context"

	"github.com/maguowei/gotobeta/internal/ent"
	entver "github.com/maguowei/gotobeta/internal/ent/rbacpermissionversion"
	"github.com/maguowei/gotobeta/internal/infra/entdb"
	"github.com/maguowei/gotobeta/internal/modules/workspace/domain/rbac"
)

// BumpUserVersion 递增用户权限版本号并返回新值（无记录则创建为 1）。
// 用于权限变更后精准失效缓存键 perm:user:{ws}:{uid}:v{version}。
func (r *RBACRepository) BumpUserVersion(ctx context.Context, workspaceID, userID int64) (int64, error) {
	client := entdb.ClientFromCtx(ctx, r.client)
	bump := func() (int, error) {
		return client.RbacPermissionVersion.Update().
			Where(
				entver.WorkspaceID(workspaceID),
				entver.SubjectType(rbac.SubjectTypeUser),
				entver.SubjectID(userID),
			).
			AddVersion(1).
			Save(ctx)
	}

	affected, err := bump()
	if err != nil {
		return 0, err
	}
	if affected == 0 {
		// 首次：创建版本记录，初始为 1。并发首次创建会触发唯一约束冲突，
		// 此时记录已被另一事务创建，退回递增分支保证版本单调，避免命令整体回滚。
		err := client.RbacPermissionVersion.Create().
			SetWorkspaceID(workspaceID).
			SetSubjectType(rbac.SubjectTypeUser).
			SetSubjectID(userID).
			SetVersion(1).
			Exec(ctx)
		switch {
		case err == nil:
			return 1, nil
		case ent.IsConstraintError(err):
			if _, err := bump(); err != nil {
				return 0, err
			}
		default:
			return 0, err
		}
	}
	return r.GetUserVersion(ctx, workspaceID, userID)
}

// GetUserVersion 返回用户当前权限版本号，无记录时返回 0。
func (r *RBACRepository) GetUserVersion(ctx context.Context, workspaceID, userID int64) (int64, error) {
	client := entdb.ClientFromCtx(ctx, r.client)
	row, err := client.RbacPermissionVersion.Query().
		Where(
			entver.WorkspaceID(workspaceID),
			entver.SubjectType(rbac.SubjectTypeUser),
			entver.SubjectID(userID),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return 0, nil
		}
		return 0, err
	}
	return row.Version, nil
}

// RecordChange 写入一条授权变更审计日志。
func (r *RBACRepository) RecordChange(ctx context.Context, entry rbac.ChangeLogEntry) error {
	client := entdb.ClientFromCtx(ctx, r.client)
	bizID, err := r.idgen.NextID(ctx)
	if err != nil {
		return err
	}
	builder := client.RbacPermissionChangeLog.Create().
		SetBizID(bizID).
		SetWorkspaceID(entry.WorkspaceID).
		SetChangeType(entry.ChangeType).
		SetTargetType(entry.TargetType).
		SetTargetID(entry.TargetID).
		SetOperatorID(entry.OperatorID).
		SetRequestID(entry.RequestID).
		SetReason(entry.Reason)
	if entry.Before != nil {
		builder.SetBeforeJSON(entry.Before)
	}
	if entry.After != nil {
		builder.SetAfterJSON(entry.After)
	}
	return builder.Exec(ctx)
}
