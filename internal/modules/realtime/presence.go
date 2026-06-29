package realtime

import (
	"context"
	"log/slog"

	"github.com/maguowei/gotobeta/internal/modules/realtime/adapter/ws"
	"github.com/maguowei/gotobeta/internal/modules/realtime/infra/hub"
	"github.com/maguowei/gotobeta/internal/modules/realtime/infra/presence"
	"github.com/maguowei/gotobeta/internal/pkg/imrt"
	loggerx "github.com/maguowei/gotobeta/internal/pkg/logger"
)

// Presence 处理连接上线/下线：记录在线状态并向会话同伴广播 presence 帧，实现 ws.PresenceReporter。
type Presence struct {
	hub     *hub.Hub
	store   *presence.Store
	members imrt.MemberLookup
	logger  *slog.Logger
}

// NewPresence 创建在线状态处理器。
func NewPresence(h *hub.Hub, store *presence.Store, members imrt.MemberLookup, logger *slog.Logger) *Presence {
	return &Presence{hub: h, store: store, members: members, logger: logger}
}

// OnConnect 标记上线并向同伴广播 online。
func (p *Presence) OnConnect(ctx context.Context, userID int64) {
	if err := p.store.MarkOnline(ctx, userID); err != nil {
		loggerx.WithError(ctx, p.logger, "mark online failed", err, slog.Int64("userId", userID))
	}
	p.broadcast(ctx, userID, true)
}

// OnDisconnect 标记离线并向同伴广播 offline（仅当该用户已无任何连接）。
func (p *Presence) OnDisconnect(ctx context.Context, userID int64) {
	if p.hub.IsOnline(userID) {
		// 多端中仍有其他连接在线，不视为离线。
		return
	}
	if err := p.store.MarkOffline(ctx, userID); err != nil {
		loggerx.WithError(ctx, p.logger, "mark offline failed", err, slog.Int64("userId", userID))
	}
	p.broadcast(ctx, userID, false)
}

func (p *Presence) broadcast(ctx context.Context, userID int64, online bool) {
	peers, err := p.members.UserConversationPeers(ctx, userID)
	if err != nil {
		loggerx.WithError(ctx, p.logger, "presence lookup peers failed", err, slog.Int64("userId", userID))
		return
	}
	p.hub.Broadcast(peers, ws.PresenceFrame(userID, online))
}
