// Package ws 是 realtime 模块的 WebSocket 入站适配器（升级、心跳、帧编解码）。
package ws

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"github.com/maguowei/gotobeta/internal/modules/realtime/application/port"
	"github.com/maguowei/gotobeta/internal/pkg/imrt"
	loggerx "github.com/maguowei/gotobeta/internal/pkg/logger"
)

// EphemeralHandler 处理上行的 ephemeral 帧（typing/read），由 realtime 应用层实现。
// 第一期可为 nil（M4 只做 signal 下行）。
type EphemeralHandler interface {
	Typing(ctx context.Context, userID, conversationID int64)
	Read(ctx context.Context, userID, conversationID, readSeq int64)
}

// PresenceReporter 处理连接上线/下线，用于在线状态广播。可为 nil。
type PresenceReporter interface {
	OnConnect(ctx context.Context, userID int64)
	OnDisconnect(ctx context.Context, userID int64)
}

// Gateway 处理 WS 升级与连接生命周期。
type Gateway struct {
	tickets   port.TicketStore
	hub       imrt.Registry
	upgrader  websocket.Upgrader
	ephemeral EphemeralHandler
	presence  PresenceReporter
	logger    *slog.Logger
}

// NewGateway 创建网关。ephemeral 与 presence 可为 nil。
func NewGateway(tickets port.TicketStore, h imrt.Registry, ephemeral EphemeralHandler, presence PresenceReporter, logger *slog.Logger) *Gateway {
	return &Gateway{
		tickets: tickets,
		hub:     h,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			// 跨域校验交由网关/反向代理与 ticket 一次性鉴权兜底。
			CheckOrigin: func(*http.Request) bool { return true },
		},
		ephemeral: ephemeral,
		presence:  presence,
		logger:    logger,
	}
}

// Handle 处理 GET /ws?ticket=：校验 ticket → 升级 → 注册 → 读写泵。
func (g *Gateway) Handle(c *gin.Context) {
	token := c.Query("ticket")
	userID, err := g.tickets.Consume(c.Request.Context(), token)
	if err != nil {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	wsConn, err := g.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		// Upgrade 已写过响应头，这里只记日志。
		loggerx.WithError(c.Request.Context(), g.logger, "ws upgrade failed", err)
		return
	}

	conn := newConn(userID, wsConn)
	g.hub.Register(userID, conn)
	conn.Send(mustEncode(Frame{T: TypeAuthOK, UID: userID}))
	if g.presence != nil {
		g.presence.OnConnect(c.Request.Context(), userID)
	}

	go conn.writePump()
	g.readPump(c.Request.Context(), conn)

	g.hub.Unregister(userID, conn)
	conn.close()
	if g.presence != nil {
		g.presence.OnDisconnect(c.Request.Context(), userID)
	}
}

// readPump 阻塞读取上行帧，处理 ping/typing/read，直到连接关闭。
func (g *Gateway) readPump(ctx context.Context, conn *Conn) {
	conn.ws.SetReadLimit(4096)
	_ = conn.ws.SetReadDeadline(time.Now().Add(pongWait))
	conn.ws.SetPongHandler(func(string) error {
		return conn.ws.SetReadDeadline(time.Now().Add(pongWait))
	})
	for {
		_, raw, err := conn.ws.ReadMessage()
		if err != nil {
			return
		}
		f, err := decodeFrame(raw)
		if err != nil {
			continue
		}
		switch f.T {
		case TypePing:
			conn.Send(pongFrame())
		case TypeTyping:
			if g.ephemeral != nil && f.CID != 0 {
				g.ephemeral.Typing(ctx, conn.userID, f.CID)
			}
		case TypeRead:
			if g.ephemeral != nil && f.CID != 0 {
				g.ephemeral.Read(ctx, conn.userID, f.CID, f.ReadSeq)
			}
		}
	}
}
