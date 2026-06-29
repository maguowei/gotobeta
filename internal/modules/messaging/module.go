// Package messaging 装配会话/消息模块（读扩散 timeline + 每会话 seq）。
package messaging

import (
	"log/slog"

	"github.com/gin-gonic/gin"

	"github.com/maguowei/gotobeta/internal/ent"
	"github.com/maguowei/gotobeta/internal/infra/config"
	"github.com/maguowei/gotobeta/internal/infra/entdb"
	"github.com/maguowei/gotobeta/internal/infra/localid"
	messaginghandler "github.com/maguowei/gotobeta/internal/modules/messaging/adapter/http/handler"
	messagingrouter "github.com/maguowei/gotobeta/internal/modules/messaging/adapter/http/router"
	messagingsvc "github.com/maguowei/gotobeta/internal/modules/messaging/application/service"
	messagingpersist "github.com/maguowei/gotobeta/internal/modules/messaging/infra/persistence"
	"github.com/maguowei/gotobeta/internal/pkg/authz"
)

// Module 持有装配好的 messaging HTTP 入口。
type Module struct {
	handler *messaginghandler.ConversationHandler
}

// New 完成 messaging 模块装配（repo -> service -> handler）。
//
// checker 由组合根从 workspace 模块注入，实现跨模块鉴权而不直接 import workspace。
func New(client *ent.Client, logger *slog.Logger, _ *config.Config, checker authz.Checker) (*Module, error) {
	convRepo := messagingpersist.NewConversationRepository(client, logger)
	svc := messagingsvc.NewConversationService(
		convRepo, checker, localid.New(), entdb.NewEntTxRunner(client), logger,
	)
	return &Module{
		handler: messaginghandler.NewConversationHandler(svc),
	}, nil
}

// Mount 把会话路由挂到给定路由组。
func (m *Module) Mount(rg *gin.RouterGroup, middlewares ...gin.HandlerFunc) {
	messagingrouter.RegisterRoutes(rg, m.handler, middlewares...)
}
