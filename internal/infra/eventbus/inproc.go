// Package eventbus 提供领域事件总线的基础设施实现。
//
// 第一期为单进程部署，采用进程内同步派发；后续可平滑替换为
// Redis Streams / Kafka（sarama SDK 的唯一归口即本包）实现，
// 上层 application 只依赖 internal/pkg/event 端口，无需改动。
package eventbus

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/maguowei/gotobeta/internal/infra/metrics"
	"github.com/maguowei/gotobeta/internal/pkg/event"
)

// inprocComponent 是进程内总线在事件指标 component label 中的取值。
const inprocComponent = "eventbus_inproc"

// InProc 是进程内事件总线：订阅者按事件名注册，Publish 同步派发。
// 推送是尽力而为——某个 handler 出错只记录日志，不影响其他订阅者，
// 也不让 Publish 失败（避免破坏已提交的主业务事务）。
type InProc struct {
	mu       sync.RWMutex
	handlers map[string][]event.Handler
	logger   *slog.Logger
	metrics  *metrics.Collectors
}

// NewInProc 创建进程内事件总线。collectors 可为 nil（指标可选，nil 时静默跳过）。
func NewInProc(logger *slog.Logger, collectors *metrics.Collectors) *InProc {
	return &InProc{
		handlers: make(map[string][]event.Handler),
		logger:   logger,
		metrics:  collectors,
	}
}

// Subscribe 为指定事件名注册处理器。应在组合根接线阶段完成，运行期不再变更。
func (b *InProc) Subscribe(name string, h event.Handler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers[name] = append(b.handlers[name], h)
}

// Publish 同步派发事件给其订阅者。始终返回 nil：派发失败不应反噬主流程。
func (b *InProc) Publish(ctx context.Context, events ...event.Event) error {
	for _, e := range events {
		b.mu.RLock()
		handlers := b.handlers[e.Name()]
		b.mu.RUnlock()

		for _, h := range handlers {
			start := time.Now()
			err := h(ctx, e)
			status := "processed"
			if err != nil {
				status = "error"
				b.logger.WarnContext(ctx, "event handler failed",
					slog.String("event", e.Name()),
					slog.String("error", err.Error()),
				)
			}
			b.metrics.ObserveEventBus(ctx, inprocComponent, e.Name(), status, time.Since(start))
		}
	}
	return nil
}
