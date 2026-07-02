package realtime

import (
	"context"
	"log/slog"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"

	"github.com/maguowei/gotobeta/internal/modules/realtime/adapter/ws"
	"github.com/maguowei/gotobeta/internal/modules/realtime/infra/hub"
	"github.com/maguowei/gotobeta/internal/pkg/event"
	"github.com/maguowei/gotobeta/internal/pkg/imevent"
	"github.com/maguowei/gotobeta/internal/pkg/imrt"
	loggerx "github.com/maguowei/gotobeta/internal/pkg/logger"
)

// dispatchTracer 是推送链路的 tracer；全局 provider 未配置时为 noop。
var dispatchTracer = otel.Tracer("realtime")

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

// fanout 查会话成员并向在线成员广播帧，统一记录失败日志与推送计数。
func (d *Dispatcher) fanout(ctx context.Context, conversationID int64, frame []byte) error {
	userIDs, err := d.members.ConversationUserIDs(ctx, conversationID)
	if err != nil {
		d.incPush("error")
		loggerx.WithError(ctx, d.logger, "dispatch lookup members failed", err, slog.Int64("conversationId", conversationID))
		return err
	}
	d.hub.Broadcast(userIDs, frame)
	d.incPush("success")
	return nil
}

// OnMessageCreated 处理消息创建事件：查会话成员 → 向在线成员推 signal 帧。
func (d *Dispatcher) OnMessageCreated(ctx context.Context, e event.Event) error {
	evt, ok := e.(imevent.MessageCreatedEvent)
	if !ok {
		return nil
	}
	// 进程内事件总线同步传递 ctx，从中起子 span 串联“发消息→推送”链路。
	ctx, span := dispatchTracer.Start(ctx, "realtime.dispatch")
	defer span.End()
	span.SetAttributes(
		attribute.Int64("conversation_id", evt.ConversationID),
		attribute.Int64("seq", evt.Seq),
	)
	return d.fanout(ctx, evt.ConversationID, ws.SignalFrame(evt.ConversationID, evt.Seq))
}

// OnReadUpdated 处理已读水位更新事件：向会话成员（含本人其他端）推 read 帧对齐。
func (d *Dispatcher) OnReadUpdated(ctx context.Context, e event.Event) error {
	evt, ok := e.(imevent.ReadUpdatedEvent)
	if !ok {
		return nil
	}
	return d.fanout(ctx, evt.ConversationID, ws.ReadFrame(evt.ConversationID, evt.UserID, evt.ReadSeq))
}

// OnReactionUpdated 处理表情回应变更事件：向会话在线成员推 reaction 帧同步。
func (d *Dispatcher) OnReactionUpdated(ctx context.Context, e event.Event) error {
	evt, ok := e.(imevent.ReactionUpdatedEvent)
	if !ok {
		return nil
	}
	return d.fanout(ctx, evt.ConversationID, ws.ReactionFrame(evt.ConversationID, evt.MessageID, evt.UserID, evt.Emoji, evt.Action))
}

// OnMessageEdited 处理消息编辑事件：向会话在线成员推 edit 帧，携带新内容原地替换。
func (d *Dispatcher) OnMessageEdited(ctx context.Context, e event.Event) error {
	evt, ok := e.(imevent.MessageEditedEvent)
	if !ok {
		return nil
	}
	return d.fanout(ctx, evt.ConversationID, ws.EditFrame(evt.ConversationID, evt.MessageID, evt.Content))
}
