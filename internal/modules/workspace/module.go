// Package workspace 装配工作区与 IAM（动态 RBAC + ACL）模块。
package workspace

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/maguowei/gotobeta/internal/ent"
	"github.com/maguowei/gotobeta/internal/infra/cache"
	"github.com/maguowei/gotobeta/internal/infra/config"
	"github.com/maguowei/gotobeta/internal/infra/entdb"
	"github.com/maguowei/gotobeta/internal/infra/localid"
	workspacehandler "github.com/maguowei/gotobeta/internal/modules/workspace/adapter/http/handler"
	workspacerouter "github.com/maguowei/gotobeta/internal/modules/workspace/adapter/http/router"
	workspacesvc "github.com/maguowei/gotobeta/internal/modules/workspace/application/service"
	"github.com/maguowei/gotobeta/internal/modules/workspace/domain/rbac"
	workspaceauthz "github.com/maguowei/gotobeta/internal/modules/workspace/infra/authz"
	workspacepersist "github.com/maguowei/gotobeta/internal/modules/workspace/infra/persistence"
	"github.com/maguowei/gotobeta/internal/pkg/authz"
)

// defaultPermCacheTTL 是权限动作集缓存的兜底 TTL（精准失效靠版本号，TTL 仅作上限）。
const defaultPermCacheTTL = time.Hour

// Module 持有装配好的 workspace HTTP 入口与权限裁决器。
type Module struct {
	handler *workspacehandler.WorkspaceHandler
	checker authz.Checker
}

// New 完成 workspace 模块装配（repo -> checker -> service -> handler）。
// kv 可为 nil（无 Redis）：此时权限解析直查 DB，不做版本化缓存。
func New(client *ent.Client, logger *slog.Logger, _ *config.Config, kv *cache.RedisKV) (*Module, error) {
	generator := localid.New()
	wsRepo := workspacepersist.NewWorkspaceRepository(client, logger)
	memRepo := workspacepersist.NewMembershipRepository(client, logger)
	rbacRepo := workspacepersist.NewRBACRepository(client, logger, generator)
	aclRepo := workspacepersist.NewACLRepository(client, logger)

	// 权限裁决走带版本化缓存的解析器（kv 可用时）；写操作仍用原始 rbacRepo。
	var resolver rbac.Repository = rbacRepo
	if kv != nil {
		resolver = workspaceauthz.NewCachedResolver(rbacRepo, kv, defaultPermCacheTTL)
	}
	checker := workspaceauthz.NewChecker(resolver, aclRepo)
	svc := workspacesvc.NewWorkspaceService(
		wsRepo, memRepo, rbacRepo, checker,
		generator, entdb.NewEntTxRunner(client), logger,
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
