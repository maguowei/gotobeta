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
	"github.com/maguowei/gotobeta/internal/modules/realtime/infra/ticket"
	"github.com/maguowei/gotobeta/internal/pkg/event"
	"github.com/maguowei/gotobeta/internal/pkg/imevent"
	"github.com/maguowei/gotobeta/internal/pkg/imrt"
)

const defaultTicketTTL = 30 * time.Second

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
// kv 可为 nil（ticket 退化为单机内存）；members 由 messaging 模块注入。
func New(cfg *config.Config, kv *cache.RedisKV, members imrt.MemberLookup, bus Subscriber, logger *slog.Logger) (*Module, error) {
	var ticketKV ticket.KV
	if kv != nil {
		ticketKV = kv
	}
	ticketStore := ticket.NewStore(ticketKV, ticketTTL(cfg))
	connHub := hub.New()

	ticketSvc := realtimesvc.NewTicketService(ticketStore)
	gateway := ws.NewGateway(ticketStore, connHub, nil, logger)

	dispatcher := NewDispatcher(connHub, members, logger)
	bus.Subscribe(imevent.MessageCreated, dispatcher.OnMessageCreated)

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
	if cfg.IM.WSTicketTTL == "" {
		return defaultTicketTTL
	}
	d, err := time.ParseDuration(cfg.IM.WSTicketTTL)
	if err != nil || d <= 0 {
		return defaultTicketTTL
	}
	return d
}
