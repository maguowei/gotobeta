// Package ws 是 realtime 模块的 WebSocket 入站适配器（升级、心跳、帧编解码）。
package ws

import (
	"context"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
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
	// Refresh 在连接存活期间周期续期在线状态 TTL，防误判离线。
	Refresh(ctx context.Context, userID int64)
}

// GatewayConfig 是网关装配参数。
type GatewayConfig struct {
	AllowedOrigins          []string           // WS 跨域来源白名单（为空仅放行同源/无 Origin）
	OnOverflow              func(userID int64) // 连接写缓冲溢出断连时回调（供指标计数），可为 nil
	PresenceRefreshInterval time.Duration      // 在线状态续期间隔（应小于 presence TTL），<=0 禁用
	WriteWait               time.Duration      // 单帧写超时，<=0 用默认 10s
	PongWait                time.Duration      // 读超时（等待 pong），<=0 用默认 60s
	ReadLimit               int64              // 单帧读上限（字节），<=0 用默认 4096
}

// Gateway 处理 WS 升级与连接生命周期。
type Gateway struct {
	tickets         port.TicketStore
	hub             imrt.Registry
	upgrader        websocket.Upgrader
	ephemeral       EphemeralHandler
	presence        PresenceReporter
	logger          *slog.Logger
	onOverflow      func(userID int64)
	presenceRefresh time.Duration
	timeouts        wsTimeouts
}

// NewGateway 创建网关。ephemeral 与 presence 可为 nil。
func NewGateway(tickets port.TicketStore, h imrt.Registry, ephemeral EphemeralHandler, presence PresenceReporter, logger *slog.Logger, cfg GatewayConfig) *Gateway {
	return &Gateway{
		tickets: tickets,
		hub:     h,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			// 跨域校验按白名单 + 同源策略，叠加 ticket 一次性鉴权。
			CheckOrigin: originChecker(cfg.AllowedOrigins),
		},
		ephemeral:       ephemeral,
		presence:        presence,
		logger:          logger,
		onOverflow:      cfg.OnOverflow,
		presenceRefresh: cfg.PresenceRefreshInterval,
		timeouts: wsTimeouts{
			writeWait: cfg.WriteWait,
			pongWait:  cfg.PongWait,
			readLimit: cfg.ReadLimit,
		}.withDefaults(),
	}
}

// originChecker 返回 WS 跨域校验函数：
//   - 无 Origin 头（多为原生客户端）放行，鉴权由 ticket 兜底；
//   - 配置了白名单时，仅放行白名单内的 Origin；
//   - 未配置白名单时，仅放行同源请求（Origin.Host == 请求 Host）。
func originChecker(allowed []string) func(*http.Request) bool {
	set := make(map[string]struct{}, len(allowed))
	for _, o := range allowed {
		set[o] = struct{}{}
	}
	return func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		if origin == "" {
			return true
		}
		if len(set) > 0 {
			_, ok := set[origin]
			return ok
		}
		u, err := url.Parse(origin)
		if err != nil {
			return false
		}
		return strings.EqualFold(u.Host, r.Host)
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
	// 协议版本协商（spec 2.5）：缺省视为 v1，便于未声明版本的老客户端兼容。
	clientVersion := CurrentProtocolVersion
	if v := c.Query("v"); v != "" {
		parsed, perr := strconv.Atoi(v)
		if perr != nil {
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}
		clientVersion = parsed
	}

	wsConn, err := g.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		// Upgrade 已写过响应头，这里只记日志。
		loggerx.WithError(c.Request.Context(), g.logger, "ws upgrade failed", err)
		return
	}

	// 升级后才能用 WS close 帧表达版本不兼容（HTTP 响应头已被 Upgrade 占用）。
	if !VersionSupported(clientVersion) {
		_ = wsConn.WriteControl(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(closeUnsupportedVersion, "unsupported protocol version"),
			time.Now().Add(g.timeouts.writeWait),
		)
		_ = wsConn.Close()
		g.logger.WarnContext(c.Request.Context(), "ws 协议版本不兼容，拒绝接入",
			slog.Int64("userId", userID), slog.Int("clientVersion", clientVersion))
		return
	}

	var overflow func()
	if g.onOverflow != nil {
		overflow = func() { g.onOverflow(userID) }
	}
	conn := newConn(userID, wsConn, sendBufferSize, overflow, g.timeouts)
	if !g.hub.Register(userID, conn) {
		// 达到连接上限：已完成 WS 握手，按协议用 1013(Try Again Later) 关闭而非 HTTP 503。
		_ = wsConn.WriteControl(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseTryAgainLater, "connection limit reached"),
			time.Now().Add(g.timeouts.writeWait),
		)
		_ = wsConn.Close()
		g.logger.WarnContext(c.Request.Context(), "ws 连接达上限，拒绝接入", slog.Int64("userId", userID))
		return
	}
	conn.Send(mustEncode(Frame{T: TypeAuthOK, UID: userID, PV: CurrentProtocolVersion}))
	if g.presence != nil {
		g.presence.OnConnect(c.Request.Context(), userID)
		if g.presenceRefresh > 0 {
			go g.refreshPresence(c.Request.Context(), conn, userID)
		}
	}

	go conn.writePump()
	g.readPump(c.Request.Context(), conn)

	g.hub.Unregister(userID, conn)
	conn.Close()
	if g.presence != nil {
		g.presence.OnDisconnect(c.Request.Context(), userID)
	}
}

// refreshPresence 在连接存活期间周期续期在线状态 TTL，连接关闭或 ctx 取消时退出。
// 续期间隔小于 presence TTL，避免心跳间隔（pingPeriod）大于 TTL 时被误判离线。
func (g *Gateway) refreshPresence(ctx context.Context, conn *Conn, userID int64) {
	ticker := time.NewTicker(g.presenceRefresh)
	defer ticker.Stop()
	for {
		select {
		case <-conn.closed:
			return
		case <-ctx.Done():
			return
		case <-ticker.C:
			g.presence.Refresh(ctx, userID)
		}
	}
}

// readPump 阻塞读取上行帧，处理 ping/typing/read，直到连接关闭。
func (g *Gateway) readPump(ctx context.Context, conn *Conn) {
	conn.ws.SetReadLimit(conn.to.readLimit)
	_ = conn.ws.SetReadDeadline(time.Now().Add(conn.to.pongWait))
	conn.ws.SetPongHandler(func(string) error {
		return conn.ws.SetReadDeadline(time.Now().Add(conn.to.pongWait))
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
