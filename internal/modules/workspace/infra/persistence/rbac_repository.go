package persistence

import (
	"context"
	"log/slog"
	"time"

	"github.com/maguowei/gotobeta/internal/ent"
	entperm "github.com/maguowei/gotobeta/internal/ent/rbacpermission"
	entrole "github.com/maguowei/gotobeta/internal/ent/rbacrole"
	entrp "github.com/maguowei/gotobeta/internal/ent/rbacrolepermission"
	entur "github.com/maguowei/gotobeta/internal/ent/rbacuserrole"
	"github.com/maguowei/gotobeta/internal/infra/entdb"
	"github.com/maguowei/gotobeta/internal/modules/workspace/domain/rbac"
)

// RBACRepository 是动态 RBAC 仓储的 Ent 实现。
type RBACRepository struct {
	client *ent.Client
	logger *slog.Logger
}

// NewRBACRepository 创建仓储。
func NewRBACRepository(client *ent.Client, logger *slog.Logger) *RBACRepository {
	return &RBACRepository{client: client, logger: logger}
}

// CreateRole 新增角色。
func (r *RBACRepository) CreateRole(ctx context.Context, role *rbac.Role) error {
	client := entdb.ClientFromCtx(ctx, r.client)
	return client.RbacRole.Create().
		SetBizID(role.ID()).
		SetWorkspaceID(role.WorkspaceID()).
		SetCode(role.Code()).
		SetName(role.Name()).
		SetRoleType(role.RoleType()).
		SetStatus(role.Status()).
		Exec(ctx)
}

// FindRoleByCode 按编码查角色。
func (r *RBACRepository) FindRoleByCode(ctx context.Context, workspaceID int64, code string) (*rbac.Role, error) {
	client := entdb.ClientFromCtx(ctx, r.client)
	row, err := client.RbacRole.Query().
		Where(entrole.WorkspaceID(workspaceID), entrole.Code(code)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, rbac.ErrRoleNotFound
		}
		return nil, err
	}
	return rbac.UnmarshalRole(row.BizID, row.WorkspaceID, row.Code, row.Name, row.RoleType, row.Status), nil
}

// ListRoles 返回工作区角色（含平台模板 workspace_id=0）。
func (r *RBACRepository) ListRoles(ctx context.Context, workspaceID int64) ([]*rbac.Role, error) {
	client := entdb.ClientFromCtx(ctx, r.client)
	rows, err := client.RbacRole.Query().Where(entrole.WorkspaceID(workspaceID)).All(ctx)
	if err != nil {
		return nil, err
	}
	items := make([]*rbac.Role, 0, len(rows))
	for _, row := range rows {
		items = append(items, rbac.UnmarshalRole(row.BizID, row.WorkspaceID, row.Code, row.Name, row.RoleType, row.Status))
	}
	return items, nil
}

// CreatePermission 新增权限定义。
func (r *RBACRepository) CreatePermission(ctx context.Context, p *rbac.Permission) error {
	client := entdb.ClientFromCtx(ctx, r.client)
	return client.RbacPermission.Create().
		SetBizID(p.ID()).
		SetWorkspaceID(p.WorkspaceID()).
		SetCode(p.Code()).
		SetName(p.Name()).
		SetResourceType("").
		SetActionKey("").
		Exec(ctx)
}

// FindPermissionByCode 按编码查权限。
func (r *RBACRepository) FindPermissionByCode(ctx context.Context, workspaceID int64, code string) (*rbac.Permission, error) {
	client := entdb.ClientFromCtx(ctx, r.client)
	row, err := client.RbacPermission.Query().
		Where(entperm.WorkspaceID(workspaceID), entperm.Code(code)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, rbac.ErrPermissionNotFound
		}
		return nil, err
	}
	return rbac.UnmarshalPermission(row.BizID, row.WorkspaceID, row.Code, row.Name, row.ResourceType, row.ActionKey, row.Status), nil
}

// ListPermissions 返回权限目录。
func (r *RBACRepository) ListPermissions(ctx context.Context, workspaceID int64) ([]*rbac.Permission, error) {
	client := entdb.ClientFromCtx(ctx, r.client)
	rows, err := client.RbacPermission.Query().Where(entperm.WorkspaceID(workspaceID)).All(ctx)
	if err != nil {
		return nil, err
	}
	items := make([]*rbac.Permission, 0, len(rows))
	for _, row := range rows {
		items = append(items, rbac.UnmarshalPermission(row.BizID, row.WorkspaceID, row.Code, row.Name, row.ResourceType, row.ActionKey, row.Status))
	}
	return items, nil
}

// BindRolePermission 绑定角色与权限（幂等）。
func (r *RBACRepository) BindRolePermission(ctx context.Context, workspaceID, roleID, permissionID int64) error {
	client := entdb.ClientFromCtx(ctx, r.client)
	exists, err := client.RbacRolePermission.Query().
		Where(entrp.RoleID(roleID), entrp.PermissionID(permissionID)).
		Exist(ctx)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	return client.RbacRolePermission.Create().
		SetWorkspaceID(workspaceID).
		SetRoleID(roleID).
		SetPermissionID(permissionID).
		Exec(ctx)
}

// AssignRole 给用户分配角色（幂等）。
func (r *RBACRepository) AssignRole(ctx context.Context, ur *rbac.UserRole) error {
	client := entdb.ClientFromCtx(ctx, r.client)
	exists, err := client.RbacUserRole.Query().
		Where(entur.WorkspaceID(ur.WorkspaceID()), entur.UserID(ur.UserID()), entur.RoleID(ur.RoleID())).
		Exist(ctx)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	builder := client.RbacUserRole.Create().
		SetWorkspaceID(ur.WorkspaceID()).
		SetUserID(ur.UserID()).
		SetRoleID(ur.RoleID()).
		SetSourceType(ur.SourceType())
	if ur.EffectiveEndAt() != nil {
		builder.SetEffectiveEndAt(*ur.EffectiveEndAt())
	}
	return builder.Exec(ctx)
}

// RevokeRole 撤销用户角色。
func (r *RBACRepository) RevokeRole(ctx context.Context, workspaceID, userID, roleID int64) error {
	client := entdb.ClientFromCtx(ctx, r.client)
	_, err := client.RbacUserRole.Delete().
		Where(entur.WorkspaceID(workspaceID), entur.UserID(userID), entur.RoleID(roleID)).
		Exec(ctx)
	return err
}

// ResolveUserActions 解析用户在工作区内的全部有效权限编码集合。
// 链路：user_roles(未过期) → role_permissions → permissions.code。
func (r *RBACRepository) ResolveUserActions(ctx context.Context, workspaceID, userID int64) (map[string]struct{}, error) {
	client := entdb.ClientFromCtx(ctx, r.client)
	now := time.Now()

	urRows, err := client.RbacUserRole.Query().
		Where(
			entur.WorkspaceID(workspaceID),
			entur.UserID(userID),
			entur.Or(entur.EffectiveEndAtIsNil(), entur.EffectiveEndAtGT(now)),
		).
		All(ctx)
	if err != nil {
		return nil, err
	}
	if len(urRows) == 0 {
		return map[string]struct{}{}, nil
	}
	roleIDs := make([]int64, 0, len(urRows))
	for _, ur := range urRows {
		roleIDs = append(roleIDs, ur.RoleID)
	}

	rpRows, err := client.RbacRolePermission.Query().
		Where(entrp.RoleIDIn(roleIDs...)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	if len(rpRows) == 0 {
		return map[string]struct{}{}, nil
	}
	permIDs := make([]int64, 0, len(rpRows))
	for _, rp := range rpRows {
		permIDs = append(permIDs, rp.PermissionID)
	}

	permRows, err := client.RbacPermission.Query().
		Where(entperm.BizIDIn(permIDs...), entperm.StatusEQ(1)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	set := make(map[string]struct{}, len(permRows))
	for _, p := range permRows {
		set[p.Code] = struct{}{}
	}
	return set, nil
}

// ListUserRoleIDs 返回用户在工作区内的有效角色 ID 列表。
func (r *RBACRepository) ListUserRoleIDs(ctx context.Context, workspaceID, userID int64) ([]int64, error) {
	client := entdb.ClientFromCtx(ctx, r.client)
	now := time.Now()
	rows, err := client.RbacUserRole.Query().
		Where(
			entur.WorkspaceID(workspaceID),
			entur.UserID(userID),
			entur.Or(entur.EffectiveEndAtIsNil(), entur.EffectiveEndAtGT(now)),
		).
		All(ctx)
	if err != nil {
		return nil, err
	}
	ids := make([]int64, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.RoleID)
	}
	return ids, nil
}

// HasRoleCode 判断用户是否拥有某角色编码。
func (r *RBACRepository) HasRoleCode(ctx context.Context, workspaceID, userID int64, code string) (bool, error) {
	client := entdb.ClientFromCtx(ctx, r.client)
	role, err := client.RbacRole.Query().
		Where(entrole.WorkspaceID(workspaceID), entrole.Code(code)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}
	return client.RbacUserRole.Query().
		Where(entur.WorkspaceID(workspaceID), entur.UserID(userID), entur.RoleID(role.BizID)).
		Exist(ctx)
}
