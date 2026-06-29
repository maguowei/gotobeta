package realtime

import (
	"context"
	"log/slog"

	"github.com/maguowei/gotobeta/internal/modules/realtime/adapter/ws"
	"github.com/maguowei/gotobeta/internal/modules/realtime/infra/hub"
	"github.com/maguowei/gotobeta/internal/pkg/event"
	"github.com/maguowei/gotobeta/internal/pkg/imevent"
	"github.com/maguowei/gotobeta/internal/pkg/imrt"
	loggerx "github.com/maguowei/gotobeta/internal/pkg/logger"
)

// Dispatcher 订阅领域事件并向在线成员扇出实时信号。
type Dispatcher struct {
	hub     *hub.Hub
	members imrt.MemberLookup
	logger  *slog.Logger
}

// NewDispatcher 创建分发器。
func NewDispatcher(h *hub.Hub, members imrt.MemberLookup, logger *slog.Logger) *Dispatcher {
	return &Dispatcher{hub: h, members: members, logger: logger}
}

// OnMessageCreated 处理消息创建事件：查会话成员 → 向在线成员推 signal 帧。
func (d *Dispatcher) OnMessageCreated(ctx context.Context, e event.Event) error {
	evt, ok := e.(imevent.MessageCreatedEvent)
	if !ok {
		return nil
	}
	userIDs, err := d.members.ConversationUserIDs(ctx, evt.ConversationID)
	if err != nil {
		loggerx.WithError(ctx, d.logger, "dispatch lookup members failed", err, slog.Int64("conversationId", evt.ConversationID))
		return err
	}
	d.hub.Broadcast(userIDs, ws.SignalFrame(evt.ConversationID, evt.Seq))
	return nil
}
