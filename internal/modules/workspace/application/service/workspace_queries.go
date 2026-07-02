package service

import (
	"context"
	stderrors "errors"

	workspacequery "github.com/maguowei/gotobeta/internal/modules/workspace/application/query"
	workspaceresult "github.com/maguowei/gotobeta/internal/modules/workspace/application/result"
	"github.com/maguowei/gotobeta/internal/modules/workspace/domain/membership"
	"github.com/maguowei/gotobeta/internal/pkg/apperr"
)

// ListMyWorkspaces 返回用户加入的全部工作区。
func (s *WorkspaceService) ListMyWorkspaces(ctx context.Context, q workspacequery.ListMyWorkspacesQuery) ([]*workspaceresult.WorkspaceResult, error) {
	rows, err := s.workspaces.ListByMemberUser(ctx, q.UserID)
	if err != nil {
		return nil, apperr.WrapInternal("查询工作区失败", err)
	}
	items := make([]*workspaceresult.WorkspaceResult, 0, len(rows))
	for _, w := range rows {
		items = append(items, toWorkspaceResult(w))
	}
	return items, nil
}

// ListRoles 返回工作区内角色（要求调用方是工作区成员）。
func (s *WorkspaceService) ListRoles(ctx context.Context, q workspacequery.ListRolesQuery) ([]*workspaceresult.RoleResult, error) {
	if _, err := s.memberships.FindByWorkspaceUser(ctx, q.WorkspaceID, q.OperatorUserID); err != nil {
		if stderrors.Is(err, membership.ErrNotFound) {
			return nil, apperr.Forbidden("不是该工作区成员")
		}
		return nil, apperr.WrapInternal("查询成员失败", err)
	}
	roles, err := s.rbac.ListRoles(ctx, q.WorkspaceID)
	if err != nil {
		return nil, apperr.WrapInternal("查询角色失败", err)
	}
	items := make([]*workspaceresult.RoleResult, 0, len(roles))
	for _, r := range roles {
		items = append(items, &workspaceresult.RoleResult{ID: r.ID(), Code: r.Code(), Name: r.Name()})
	}
	return items, nil
}
