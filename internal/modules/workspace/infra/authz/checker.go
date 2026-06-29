// Package authz 是 workspace 模块对 internal/pkg/authz.Checker 的实现：
// 组合动态 RBAC（动作授权）+ ACL（实例级例外）+ owner 短路，给出最终裁决。
package authz

import (
	"context"
	"time"

	"github.com/maguowei/gotobeta/internal/modules/workspace/domain/acl"
	"github.com/maguowei/gotobeta/internal/modules/workspace/domain/rbac"
	"github.com/maguowei/gotobeta/internal/pkg/apperr"
	pkgauthz "github.com/maguowei/gotobeta/internal/pkg/authz"
)

// Checker 实现 pkgauthz.Checker。
type Checker struct {
	rbacRepo rbac.Repository
	aclRepo  acl.Repository
}

// NewChecker 创建权限裁决器。
func NewChecker(rbacRepo rbac.Repository, aclRepo acl.Repository) *Checker {
	return &Checker{rbacRepo: rbacRepo, aclRepo: aclRepo}
}

// 编译期断言：Checker 满足端口。
var _ pkgauthz.Checker = (*Checker)(nil)

// Check 裁决鉴权请求。允许返回 nil，拒绝返回 apperr.Forbidden。
func (c *Checker) Check(ctx context.Context, req pkgauthz.Request) error {
	now := time.Now()

	// 1. 解析 RBAC 动作集合，判断是否拥有该动作。
	actions, err := c.rbacRepo.ResolveUserActions(ctx, req.WorkspaceID, req.Subject.UserID)
	if err != nil {
		return apperr.Internal("解析权限失败", err)
	}
	_, rbacAllowed := actions[req.Action]

	// 2. owner 短路：拥有 owner 角色视为 RBAC 允许（仍受 ACL 显式拒绝约束）。
	if !rbacAllowed {
		isOwner, err := c.rbacRepo.HasRoleCode(ctx, req.WorkspaceID, req.Subject.UserID, rbac.RoleOwner)
		if err != nil {
			return apperr.Internal("解析角色失败", err)
		}
		rbacAllowed = isOwner
	}

	// 3. 无资源实例：纯工作区级动作授权，直接看 RBAC。
	if req.ResourceID == "" {
		if rbacAllowed {
			return nil
		}
		return apperr.Forbidden("没有执行该操作的权限")
	}

	// 4. 有资源实例：查 ACL 例外（拒绝优先），与 RBAC 组合裁决。
	roleIDs, err := c.rbacRepo.ListUserRoleIDs(ctx, req.WorkspaceID, req.Subject.UserID)
	if err != nil {
		return apperr.Internal("解析角色失败", err)
	}
	decisive, err := c.aclRepo.FindDecisive(ctx, req.WorkspaceID, req.Subject.UserID, roleIDs, req.ResourceType, req.ResourceID, req.Action, now)
	if err != nil {
		return apperr.Internal("解析 ACL 失败", err)
	}
	if acl.Decide(rbacAllowed, decisive) {
		return nil
	}
	return apperr.Forbidden("没有执行该操作的权限")
}
