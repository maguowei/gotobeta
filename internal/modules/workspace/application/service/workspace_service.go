// Package service 编排 workspace 模块用例（工作区、成员、动态 RBAC）。
package service

import (
	"log/slog"

	workspaceresult "github.com/maguowei/gotobeta/internal/modules/workspace/application/result"
	"github.com/maguowei/gotobeta/internal/modules/workspace/domain/membership"
	"github.com/maguowei/gotobeta/internal/modules/workspace/domain/rbac"
	"github.com/maguowei/gotobeta/internal/modules/workspace/domain/workspace"
	"github.com/maguowei/gotobeta/internal/pkg/authz"
	"github.com/maguowei/gotobeta/internal/pkg/idgen"
	"github.com/maguowei/gotobeta/internal/pkg/persistence"
)

// WorkspaceService 编排工作区相关用例。
type WorkspaceService struct {
	workspaces  workspace.Repository
	memberships membership.Repository
	rbac        rbac.Repository
	checker     authz.Checker
	idGenerator idgen.Generator
	txRunner    persistence.TxRunner
	logger      *slog.Logger
}

// NewWorkspaceService 创建服务。
func NewWorkspaceService(
	workspaces workspace.Repository,
	memberships membership.Repository,
	rbacRepo rbac.Repository,
	checker authz.Checker,
	idGenerator idgen.Generator,
	txRunner persistence.TxRunner,
	logger *slog.Logger,
) *WorkspaceService {
	return &WorkspaceService{
		workspaces:  workspaces,
		memberships: memberships,
		rbac:        rbacRepo,
		checker:     checker,
		idGenerator: idGenerator,
		txRunner:    txRunner,
		logger:      logger,
	}
}

func toWorkspaceResult(w *workspace.Workspace) *workspaceresult.WorkspaceResult {
	return &workspaceresult.WorkspaceResult{
		ID:          w.ID(),
		Slug:        w.Slug(),
		Name:        w.Name(),
		OwnerUserID: w.OwnerUserID(),
		Status:      int8(w.Status()),
		CreatedAt:   w.CreatedAt(),
	}
}

func toMemberResult(m *membership.Member) *workspaceresult.MemberResult {
	return &workspaceresult.MemberResult{
		WorkspaceID: m.WorkspaceID(),
		UserID:      m.UserID(),
		Status:      int8(m.Status()),
		JoinedAt:    m.JoinedAt(),
	}
}
