// Package realtime 装配实时网关（WS ticket + 进程内 Hub + 事件分发）。
package realtime

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/maguowei/gotobeta/internal/infra/cache"
	"github.com/maguowei/gotobeta/internal/infra/config"
	realtimehandler "github.com/maguowei/gotobeta/internal/modules/realtime/adapter/http/handler"
	realtimerouter "github.com/maguowei/gotobeta/internal/modules/realtime/adapter/http/router"
	"github.com/maguowei/gotobeta/internal/modules/realtime/adapter/ws"
	realtimesvc "github.com/maguowei/gotobeta/internal/modules/realtime/application/service"
	"github.com/maguowei/gotobeta/internal/modules/realtime/infra/hub"
	"github.com/maguowei/gotobeta/internal/modules/realtime/infra/presence"
	"github.com/maguowei/gotobeta/internal/modules/realtime/infra/ticket"
	"github.com/maguowei/gotobeta/internal/pkg/event"
	"github.com/maguowei/gotobeta/internal/pkg/imevent"
	"github.com/maguowei/gotobeta/internal/pkg/imrt"
)

const (
	defaultTicketTTL   = 30 * time.Second
	defaultPresenceTTL = 30 * time.Second
)

// Subscriber 是事件总线的订阅端口（由 eventbus.InProc 实现）。
type Subscriber interface {
	Subscribe(name string, h event.Handler)
}

// Module 持有装配好的 realtime HTTP 入口。
type Module struct {
	ticketHandler *realtimehandler.TicketHandler
	gateway       *ws.Gateway
}

// New 完成 realtime 模块装配，并把分发器订阅到事件总线。
//
// kv 可为 nil（ticket/presence 退化为单机内存）；members/reader 由 messaging 模块注入。
func New(cfg *config.Config, kv *cache.RedisKV, members imrt.MemberLookup, reader imrt.ReadReporter, bus Subscriber, logger *slog.Logger) (*Module, error) {
	var ticketKV ticket.KV
	var presenceKV presence.KV
	if kv != nil {
		ticketKV = kv
		presenceKV = kv
	}
	ticketStore := ticket.NewStore(ticketKV, ticketTTL(cfg))
	presenceStore := presence.NewStore(presenceKV, presenceTTL(cfg))
	connHub := hub.New()

	ticketSvc := realtimesvc.NewTicketService(ticketStore)
	ephemeral := NewEphemeral(connHub, members, reader, logger)
	presenceReporter := NewPresence(connHub, presenceStore, members, logger)
	gateway := ws.NewGateway(ticketStore, connHub, ephemeral, presenceReporter, logger)

	dispatcher := NewDispatcher(connHub, members, logger)
	bus.Subscribe(imevent.MessageCreated, dispatcher.OnMessageCreated)
	bus.Subscribe(imevent.ReadUpdated, dispatcher.OnReadUpdated)

	return &Module{
		ticketHandler: realtimehandler.NewTicketHandler(ticketSvc),
		gateway:       gateway,
	}, nil
}

// Mount 把 ticket 与 WS 路由挂到给定路由组。authMiddlewares 仅作用于 POST /ws/ticket。
func (m *Module) Mount(rg *gin.RouterGroup, authMiddlewares ...gin.HandlerFunc) {
	realtimerouter.RegisterRoutes(rg, m.ticketHandler, m.gateway, authMiddlewares...)
}

func ticketTTL(cfg *config.Config) time.Duration {
	return parseDuration(cfg.IM.WSTicketTTL, defaultTicketTTL)
}

func presenceTTL(cfg *config.Config) time.Duration {
	return parseDuration(cfg.IM.PresenceTTL, defaultPresenceTTL)
}

func parseDuration(raw string, fallback time.Duration) time.Duration {
	if raw == "" {
		return fallback
	}
	d, err := time.ParseDuration(raw)
	if err != nil || d <= 0 {
		return fallback
	}
	return d
}
