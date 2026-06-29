// Package messaging 装配会话/消息模块（读扩散 timeline + 每会话 seq）。
package messaging

import (
	"context"
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/maguowei/gotobeta/internal/ent"
	"github.com/maguowei/gotobeta/internal/infra/config"
	"github.com/maguowei/gotobeta/internal/infra/entdb"
	"github.com/maguowei/gotobeta/internal/infra/localid"
	messaginghandler "github.com/maguowei/gotobeta/internal/modules/messaging/adapter/http/handler"
	messagingrouter "github.com/maguowei/gotobeta/internal/modules/messaging/adapter/http/router"
	messagingcmd "github.com/maguowei/gotobeta/internal/modules/messaging/application/command"
	messagingport "github.com/maguowei/gotobeta/internal/modules/messaging/application/port"
	messagingsvc "github.com/maguowei/gotobeta/internal/modules/messaging/application/service"
	messagingpersist "github.com/maguowei/gotobeta/internal/modules/messaging/infra/persistence"
	"github.com/maguowei/gotobeta/internal/modules/messaging/infra/seqalloc"
	"github.com/maguowei/gotobeta/internal/pkg/authz"
	"github.com/maguowei/gotobeta/internal/pkg/event"
	httpmiddleware "github.com/maguowei/gotobeta/internal/pkg/httpx/middleware"
	"github.com/maguowei/gotobeta/internal/pkg/imrt"
)

// 撤回窗口默认值（解析失败时回退）。
const defaultRecallWindow = 2 * time.Minute

// Module 持有装配好的 messaging HTTP 入口。
type Module struct {
	convHandler   *messaginghandler.ConversationHandler
	msgHandler    *messaginghandler.MessageHandler
	convSvc       *messagingsvc.ConversationService
	msgSvc        *messagingsvc.MessageService
	sendRateLimit gin.HandlerFunc
}

// New 完成 messaging 模块装配（repo -> service -> handler）。
//
// checker 由组合根从 workspace 模块注入；publisher 为进程内事件总线，用于消息创建事件分发。
// metrics 可为 nil（不埋点），由组合根注入 *metrics.Collectors。
func New(client *ent.Client, logger *slog.Logger, cfg *config.Config, checker authz.Checker, publisher event.Publisher, metrics messagingport.MessageMetrics) (*Module, error) {
	convRepo := messagingpersist.NewConversationRepository(client, logger)
	msgRepo := messagingpersist.NewMessageRepository(client, logger)
	seqAllocator := seqalloc.NewDBAllocator(client)
	txRunner := entdb.NewEntTxRunner(client)
	idGen := localid.New()

	convSvc := messagingsvc.NewConversationService(convRepo, checker, idGen, txRunner, logger)
	msgSvc := messagingsvc.NewMessageService(
		msgRepo, convRepo, seqAllocator, checker, publisher, idGen, txRunner,
		recallWindow(cfg), cfg.IM.MessagePageSize, logger, metrics,
	)

	sendLimiter := httpmiddleware.NewLimiter(cfg.IM.MessageRatePerMinute, cfg.IM.MessageRateBurst)

	return &Module{
		convHandler:   messaginghandler.NewConversationHandler(convSvc),
		msgHandler:    messaginghandler.NewMessageHandler(msgSvc),
		convSvc:       convSvc,
		msgSvc:        msgSvc,
		sendRateLimit: sendLimiter.Middleware(messagingrouter.UserRateKey),
	}, nil
}

// MemberLookup 暴露会话成员查询端口，供 realtime 模块经组合根注入。
func (m *Module) MemberLookup() imrt.MemberLookup {
	return m.convSvc
}

// ReadReporter 暴露已读上报端口，供 realtime 处理 WS 上行 read 帧时回流。
func (m *Module) ReadReporter() imrt.ReadReporter {
	return readReporter{svc: m.msgSvc}
}

// readReporter 适配 MessageService 到 imrt.ReadReporter。
type readReporter struct {
	svc *messagingsvc.MessageService
}

func (r readReporter) ReportRead(ctx context.Context, conversationID, userID, readSeq int64) error {
	return r.svc.ReportRead(ctx, messagingcmd.ReportReadCommand{
		ConversationID: conversationID,
		UserID:         userID,
		ReadSeq:        readSeq,
	})
}

// Mount 把会话与消息路由挂到给定路由组。
func (m *Module) Mount(rg *gin.RouterGroup, middlewares ...gin.HandlerFunc) {
	messagingrouter.RegisterRoutes(rg, m.convHandler, m.msgHandler, m.sendRateLimit, middlewares...)
}

func recallWindow(cfg *config.Config) time.Duration {
	if cfg.IM.RecallWindow == "" {
		return defaultRecallWindow
	}
	d, err := time.ParseDuration(cfg.IM.RecallWindow)
	if err != nil || d <= 0 {
		return defaultRecallWindow
	}
	return d
}
