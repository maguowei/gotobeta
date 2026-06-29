package realtime

import (
	"context"
	"log/slog"

	"github.com/maguowei/gotobeta/internal/modules/realtime/adapter/ws"
	"github.com/maguowei/gotobeta/internal/modules/realtime/infra/hub"
	"github.com/maguowei/gotobeta/internal/pkg/imrt"
	loggerx "github.com/maguowei/gotobeta/internal/pkg/logger"
)

// Ephemeral 处理 WS 上行 ephemeral 帧（typing/read），实现 ws.EphemeralHandler。
type Ephemeral struct {
	hub     *hub.Hub
	members imrt.MemberLookup
	reader  imrt.ReadReporter
	logger  *slog.Logger
}

// NewEphemeral 创建 ephemeral 处理器。
func NewEphemeral(h *hub.Hub, members imrt.MemberLookup, reader imrt.ReadReporter, logger *slog.Logger) *Ephemeral {
	return &Ephemeral{hub: h, members: members, reader: reader, logger: logger}
}

// Typing 把 typing 广播给会话其他成员（不回送自己），不落库、不占 seq。
func (e *Ephemeral) Typing(ctx context.Context, userID, conversationID int64) {
	userIDs, err := e.members.ConversationUserIDs(ctx, conversationID)
	if err != nil {
		loggerx.WithError(ctx, e.logger, "typing lookup members failed", err, slog.Int64("conversationId", conversationID))
		return
	}
	frame := ws.TypingFrame(conversationID, userID)
	for _, uid := range userIDs {
		if uid == userID {
			continue
		}
		e.hub.Push(uid, frame)
	}
}

// Read 把 WS 上行 read 帧回流到 messaging 上报已读水位（再由事件驱动多端对齐）。
func (e *Ephemeral) Read(ctx context.Context, userID, conversationID, readSeq int64) {
	if err := e.reader.ReportRead(ctx, conversationID, userID, readSeq); err != nil {
		loggerx.WithError(ctx, e.logger, "report read failed", err, slog.Int64("conversationId", conversationID))
	}
}
