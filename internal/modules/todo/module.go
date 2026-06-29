package todo

import (
	"log/slog"

	"github.com/gin-gonic/gin"

	"github.com/maguowei/gotobeta/internal/ent"
	"github.com/maguowei/gotobeta/internal/infra/config"
	"github.com/maguowei/gotobeta/internal/infra/entdb"
	"github.com/maguowei/gotobeta/internal/infra/localid"
	todohandler "github.com/maguowei/gotobeta/internal/modules/todo/adapter/http/handler"
	todorouter "github.com/maguowei/gotobeta/internal/modules/todo/adapter/http/router"
	todosvc "github.com/maguowei/gotobeta/internal/modules/todo/application/service"
	todopersist "github.com/maguowei/gotobeta/internal/modules/todo/infra/persistence"
)

// Module 持有装配好的 Todo HTTP 入口。
type Module struct {
	handler *todohandler.TodoHandler
}

// New 完成 Todo 模块的全部装配（repo -> service -> handler）。
func New(client *ent.Client, logger *slog.Logger, cfg *config.Config) (*Module, error) {
	repo := todopersist.NewTodoRepository(client, logger)
	svc := todosvc.NewTodoService(
		repo,
		localid.New(),
		entdb.NewEntTxRunner(client),
		logger,
	)
	return &Module{handler: todohandler.NewTodoHandler(svc)}, nil
}

// Mount 把 Todo 路由挂到给定的路由组。
// middlewares 透传给路由组：启用鉴权的服务应传入登录中间件，
// 避免在鉴权服务里暴露公开可写的 demo 业务端点。
func (m *Module) Mount(rg *gin.RouterGroup, middlewares ...gin.HandlerFunc) {
	todorouter.RegisterRoutes(rg, m.handler, middlewares...)
}
