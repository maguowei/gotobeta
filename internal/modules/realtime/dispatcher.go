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

// PushMetrics 是实时推送计数端口（由 infra/metrics.Collectors 实现），可为 nil（不埋点）。
type PushMetrics interface {
	IncPush(result string)
}

// Dispatcher 订阅领域事件并向在线成员扇出实时信号。
type Dispatcher struct {
	hub     *hub.Hub
	members imrt.MemberLookup
	logger  *slog.Logger
	metrics PushMetrics
}

// NewDispatcher 创建分发器。metrics 可为 nil（不埋点）。
func NewDispatcher(h *hub.Hub, members imrt.MemberLookup, logger *slog.Logger, metrics PushMetrics) *Dispatcher {
	return &Dispatcher{hub: h, members: members, logger: logger, metrics: metrics}
}

// incPush 安全地累加推送结果计数。
func (d *Dispatcher) incPush(result string) {
	if d.metrics != nil {
		d.metrics.IncPush(result)
	}
}

// OnMessageCreated 处理消息创建事件：查会话成员 → 向在线成员推 signal 帧。
func (d *Dispatcher) OnMessageCreated(ctx context.Context, e event.Event) error {
	evt, ok := e.(imevent.MessageCreatedEvent)
	if !ok {
		return nil
	}
	userIDs, err := d.members.ConversationUserIDs(ctx, evt.ConversationID)
	if err != nil {
		d.incPush("error")
		loggerx.WithError(ctx, d.logger, "dispatch lookup members failed", err, slog.Int64("conversationId", evt.ConversationID))
		return err
	}
	d.hub.Broadcast(userIDs, ws.SignalFrame(evt.ConversationID, evt.Seq))
	d.incPush("success")
	return nil
}

// OnReadUpdated 处理已读水位更新事件：向会话成员（含本人其他端）推 read 帧对齐。
func (d *Dispatcher) OnReadUpdated(ctx context.Context, e event.Event) error {
	evt, ok := e.(imevent.ReadUpdatedEvent)
	if !ok {
		return nil
	}
	userIDs, err := d.members.ConversationUserIDs(ctx, evt.ConversationID)
	if err != nil {
		d.incPush("error")
		loggerx.WithError(ctx, d.logger, "dispatch lookup members failed", err, slog.Int64("conversationId", evt.ConversationID))
		return err
	}
	d.hub.Broadcast(userIDs, ws.ReadFrame(evt.ConversationID, evt.UserID, evt.ReadSeq))
	d.incPush("success")
	return nil
}
