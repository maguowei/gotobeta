// Package workspace 装配工作区与 IAM（动态 RBAC + ACL）模块。
package workspace

import (
	"log/slog"

	"github.com/gin-gonic/gin"

	"github.com/maguowei/gotobeta/internal/ent"
	"github.com/maguowei/gotobeta/internal/infra/config"
	"github.com/maguowei/gotobeta/internal/infra/entdb"
	"github.com/maguowei/gotobeta/internal/infra/localid"
	workspacehandler "github.com/maguowei/gotobeta/internal/modules/workspace/adapter/http/handler"
	workspacerouter "github.com/maguowei/gotobeta/internal/modules/workspace/adapter/http/router"
	workspacesvc "github.com/maguowei/gotobeta/internal/modules/workspace/application/service"
	workspaceauthz "github.com/maguowei/gotobeta/internal/modules/workspace/infra/authz"
	workspacepersist "github.com/maguowei/gotobeta/internal/modules/workspace/infra/persistence"
	"github.com/maguowei/gotobeta/internal/pkg/authz"
)

// Module 持有装配好的 workspace HTTP 入口与权限裁决器。
type Module struct {
	handler *workspacehandler.WorkspaceHandler
	checker authz.Checker
}

// New 完成 workspace 模块装配（repo -> checker -> service -> handler）。
func New(client *ent.Client, logger *slog.Logger, _ *config.Config) (*Module, error) {
	wsRepo := workspacepersist.NewWorkspaceRepository(client, logger)
	memRepo := workspacepersist.NewMembershipRepository(client, logger)
	rbacRepo := workspacepersist.NewRBACRepository(client, logger)
	aclRepo := workspacepersist.NewACLRepository(client, logger)

	checker := workspaceauthz.NewChecker(rbacRepo, aclRepo)
	svc := workspacesvc.NewWorkspaceService(
		wsRepo, memRepo, rbacRepo, checker,
		localid.New(), entdb.NewEntTxRunner(client), logger,
	)

	return &Module{
		handler: workspacehandler.NewWorkspaceHandler(svc),
		checker: checker,
	}, nil
}

// Mount 把工作区路由挂到给定路由组。
func (m *Module) Mount(rg *gin.RouterGroup, middlewares ...gin.HandlerFunc) {
	workspacerouter.RegisterRoutes(rg, m.handler, middlewares...)
}

// Checker 暴露权限裁决器，供 messaging/media 等模块经组合根注入。
func (m *Module) Checker() authz.Checker {
	return m.checker
}
