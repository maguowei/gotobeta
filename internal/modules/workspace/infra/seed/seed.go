// Package seed 提供平台级 RBAC 模板的幂等初始化（workspace_id=0）。
//
// 建工作区时 CreateWorkspace 会按编码查平台权限并绑定到租户角色，
// 因此平台权限目录必须先由 datainit 初始化。
package seed

import (
	"context"
	"errors"

	"github.com/maguowei/gotobeta/internal/modules/workspace/domain/rbac"
	"github.com/maguowei/gotobeta/internal/pkg/idgen"
)

// SeedPlatformTemplates 幂等初始化平台权限目录与模板角色（workspace_id=0）。
func SeedPlatformTemplates(ctx context.Context, repo rbac.Repository, idGen idgen.Generator) error {
	for _, tmpl := range rbac.DefaultPermissionTemplates() {
		if _, err := repo.FindPermissionByCode(ctx, 0, tmpl.Code); err == nil {
			continue
		} else if !errors.Is(err, rbac.ErrPermissionNotFound) {
			return err
		}
		id, err := idGen.NextID(ctx)
		if err != nil {
			return err
		}
		if err := repo.CreatePermission(ctx, rbac.NewPermission(id, 0, tmpl.Code, tmpl.Name, tmpl.ResourceType, tmpl.ActionKey)); err != nil {
			return err
		}
	}

	for _, tmpl := range rbac.DefaultRoleTemplates() {
		if _, err := repo.FindRoleByCode(ctx, 0, tmpl.Code); err == nil {
			continue
		} else if !errors.Is(err, rbac.ErrRoleNotFound) {
			return err
		}
		id, err := idGen.NextID(ctx)
		if err != nil {
			return err
		}
		if err := repo.CreateRole(ctx, rbac.NewRole(id, 0, tmpl.Code, tmpl.Name)); err != nil {
			return err
		}
	}
	return nil
}
