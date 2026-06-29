package service

import (
	"context"
	stderrors "errors"
	"log/slog"

	workspacecmd "github.com/maguowei/gotobeta/internal/modules/workspace/application/command"
	workspaceresult "github.com/maguowei/gotobeta/internal/modules/workspace/application/result"
	"github.com/maguowei/gotobeta/internal/modules/workspace/domain/membership"
	"github.com/maguowei/gotobeta/internal/modules/workspace/domain/rbac"
	"github.com/maguowei/gotobeta/internal/modules/workspace/domain/workspace"
	"github.com/maguowei/gotobeta/internal/pkg/apperr"
	"github.com/maguowei/gotobeta/internal/pkg/authz"
	loggerx "github.com/maguowei/gotobeta/internal/pkg/logger"
)

// CreateWorkspace 创建工作区：建工作区 + 复制平台模板角色并绑定权限 + 把 owner 加入并赋 owner 角色。
func (s *WorkspaceService) CreateWorkspace(ctx context.Context, cmd workspacecmd.CreateWorkspaceCommand) (*workspaceresult.WorkspaceResult, error) {
	// slug 唯一性预校验（最终一致由唯一索引兜底）。
	if _, err := s.workspaces.FindBySlug(ctx, cmd.Slug); err == nil {
		return nil, apperr.Conflict("工作区标识已被占用")
	} else if !stderrors.Is(err, workspace.ErrNotFound) {
		return nil, wrapInfrastructureError("查询工作区失败", err)
	}

	wsID, err := s.idGenerator.NextID(ctx)
	if err != nil {
		return nil, wrapInfrastructureError("生成工作区 ID 失败", err)
	}
	ws, err := workspace.New(wsID, cmd.Slug, cmd.Name, cmd.OwnerUserID)
	if err != nil {
		return nil, err
	}

	err = s.txRunner.RunInTx(ctx, func(txCtx context.Context) error {
		if err := s.workspaces.Create(txCtx, ws); err != nil {
			if stderrors.Is(err, workspace.ErrSlugTaken) {
				return apperr.Conflict("工作区标识已被占用")
			}
			return wrapInfrastructureError("保存工作区失败", err)
		}
		rolesByCode, err := s.seedWorkspaceRoles(txCtx, wsID)
		if err != nil {
			return err
		}
		// owner 加入成员并赋予 owner 角色。
		memID, err := s.idGenerator.NextID(txCtx)
		if err != nil {
			return wrapInfrastructureError("生成成员 ID 失败", err)
		}
		mem, err := membership.New(memID, wsID, cmd.OwnerUserID)
		if err != nil {
			return err
		}
		if err := s.memberships.Add(txCtx, mem); err != nil {
			return wrapInfrastructureError("加入成员失败", err)
		}
		if err := s.rbac.AssignRole(txCtx, rbac.NewUserRole(wsID, cmd.OwnerUserID, rolesByCode[rbac.RoleOwner])); err != nil {
			return wrapInfrastructureError("分配 owner 角色失败", err)
		}
		return nil
	})
	if err != nil {
		loggerx.WithError(ctx, s.logger, "create workspace failed", err, slog.String("slug", cmd.Slug))
		return nil, err
	}

	s.logger.InfoContext(ctx, "workspace created", slog.Int64("workspaceId", wsID), slog.String("slug", cmd.Slug))
	return toWorkspaceResult(ws), nil
}

// seedWorkspaceRoles 从平台模板复制角色到工作区，并按默认矩阵绑定平台权限。
// 返回 角色编码→角色ID 映射。
func (s *WorkspaceService) seedWorkspaceRoles(ctx context.Context, wsID int64) (map[string]int64, error) {
	rolesByCode := make(map[string]int64)
	for _, tmpl := range rbac.DefaultRoleTemplates() {
		roleID, err := s.idGenerator.NextID(ctx)
		if err != nil {
			return nil, wrapInfrastructureError("生成角色 ID 失败", err)
		}
		if err := s.rbac.CreateRole(ctx, rbac.NewRole(roleID, wsID, tmpl.Code, tmpl.Name)); err != nil {
			return nil, wrapInfrastructureError("创建角色失败", err)
		}
		rolesByCode[tmpl.Code] = roleID
	}
	for code, permCodes := range rbac.DefaultRolePermissions() {
		roleID := rolesByCode[code]
		for _, permCode := range permCodes {
			perm, err := s.rbac.FindPermissionByCode(ctx, 0, permCode)
			if err != nil {
				return nil, wrapInfrastructureError("查询平台权限失败", err)
			}
			if err := s.rbac.BindRolePermission(ctx, wsID, roleID, perm.ID()); err != nil {
				return nil, wrapInfrastructureError("绑定角色权限失败", err)
			}
		}
	}
	return rolesByCode, nil
}

// InviteMember 邀请用户加入工作区并赋予角色。
func (s *WorkspaceService) InviteMember(ctx context.Context, cmd workspacecmd.InviteMemberCommand) (*workspaceresult.MemberResult, error) {
	if err := s.checker.Check(ctx, authz.Request{
		WorkspaceID: cmd.WorkspaceID,
		Subject:     authz.Subject{UserID: cmd.OperatorUserID},
		Action:      rbac.PermMemberInvite,
	}); err != nil {
		return nil, err
	}

	role, err := s.rbac.FindRoleByCode(ctx, cmd.WorkspaceID, cmd.RoleCode)
	if err != nil {
		if stderrors.Is(err, rbac.ErrRoleNotFound) {
			return nil, apperr.InvalidParam("角色不存在")
		}
		return nil, wrapInfrastructureError("查询角色失败", err)
	}

	var mem *membership.Member
	err = s.txRunner.RunInTx(ctx, func(txCtx context.Context) error {
		memID, err := s.idGenerator.NextID(txCtx)
		if err != nil {
			return wrapInfrastructureError("生成成员 ID 失败", err)
		}
		m, err := membership.New(memID, cmd.WorkspaceID, cmd.TargetUserID)
		if err != nil {
			return err
		}
		if err := s.memberships.Add(txCtx, m); err != nil {
			if stderrors.Is(err, membership.ErrAlreadyMember) {
				return apperr.Conflict("用户已是该工作区成员")
			}
			return wrapInfrastructureError("加入成员失败", err)
		}
		if err := s.rbac.AssignRole(txCtx, rbac.NewUserRole(cmd.WorkspaceID, cmd.TargetUserID, role.ID())); err != nil {
			return wrapInfrastructureError("分配角色失败", err)
		}
		mem = m
		return nil
	})
	if err != nil {
		loggerx.WithError(ctx, s.logger, "invite member failed", err, slog.Int64("workspaceId", cmd.WorkspaceID), slog.Int64("targetUserId", cmd.TargetUserID))
		return nil, err
	}
	s.logger.InfoContext(ctx, "member invited", slog.Int64("workspaceId", cmd.WorkspaceID), slog.Int64("targetUserId", cmd.TargetUserID))
	return toMemberResult(mem), nil
}

// AssignRole 给工作区成员分配角色。
func (s *WorkspaceService) AssignRole(ctx context.Context, cmd workspacecmd.AssignRoleCommand) error {
	if err := s.checker.Check(ctx, authz.Request{
		WorkspaceID: cmd.WorkspaceID,
		Subject:     authz.Subject{UserID: cmd.OperatorUserID},
		Action:      rbac.PermRoleManage,
	}); err != nil {
		return err
	}
	role, err := s.rbac.FindRoleByCode(ctx, cmd.WorkspaceID, cmd.RoleCode)
	if err != nil {
		if stderrors.Is(err, rbac.ErrRoleNotFound) {
			return apperr.InvalidParam("角色不存在")
		}
		return wrapInfrastructureError("查询角色失败", err)
	}
	if err := s.rbac.AssignRole(ctx, rbac.NewUserRole(cmd.WorkspaceID, cmd.TargetUserID, role.ID())); err != nil {
		return wrapInfrastructureError("分配角色失败", err)
	}
	s.logger.InfoContext(ctx, "role assigned", slog.Int64("workspaceId", cmd.WorkspaceID), slog.Int64("targetUserId", cmd.TargetUserID), slog.String("role", cmd.RoleCode))
	return nil
}
