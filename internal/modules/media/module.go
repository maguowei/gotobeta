// Package media 装配附件模块（S3 兼容对象存储预签名直传与提交）。
package media

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/maguowei/gotobeta/internal/ent"
	"github.com/maguowei/gotobeta/internal/infra/config"
	"github.com/maguowei/gotobeta/internal/infra/localid"
	"github.com/maguowei/gotobeta/internal/infra/objstore"
	mediahandler "github.com/maguowei/gotobeta/internal/modules/media/adapter/http/handler"
	mediarouter "github.com/maguowei/gotobeta/internal/modules/media/adapter/http/router"
	mediaport "github.com/maguowei/gotobeta/internal/modules/media/application/port"
	mediasvc "github.com/maguowei/gotobeta/internal/modules/media/application/service"
	mediapersist "github.com/maguowei/gotobeta/internal/modules/media/infra/persistence"
	"github.com/maguowei/gotobeta/internal/pkg/authz"
)

const defaultPresignTTL = 15 * time.Minute

// Module 持有装配好的 media HTTP 入口。
type Module struct {
	handler *mediahandler.AttachmentHandler
}

// New 完成 media 模块装配（repo + presigner -> service -> handler）。
//
// presigner 由组合根注入（可为 nil，未配置对象存储时附件接口返回未启用）；
// checker 由 workspace 模块注入。
func New(client *ent.Client, logger *slog.Logger, cfg *config.Config, presigner *objstore.MinioPresigner, checker authz.Checker) (*Module, error) {
	repo := mediapersist.NewAttachmentRepository(client, logger)

	var p mediaport.Presigner
	if presigner != nil {
		p = presigner
	}
	svc := mediasvc.NewAttachmentService(repo, p, checker, localid.New(), presignTTL(cfg), logger)

	return &Module{handler: mediahandler.NewAttachmentHandler(svc)}, nil
}

// Mount 把附件路由挂到给定路由组。
func (m *Module) Mount(rg *gin.RouterGroup, middlewares ...gin.HandlerFunc) {
	mediarouter.RegisterRoutes(rg, m.handler, middlewares...)
}

func presignTTL(cfg *config.Config) time.Duration {
	if cfg.ObjStore.PresignTTL == "" {
		return defaultPresignTTL
	}
	d, err := time.ParseDuration(cfg.ObjStore.PresignTTL)
	if err != nil || d <= 0 {
		return defaultPresignTTL
	}
	return d
}
