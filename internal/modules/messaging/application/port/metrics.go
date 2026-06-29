package port

import (
	"context"
	"time"
)

// MessageMetrics 是消息用例的可观测性端口（由 infra/metrics.Collectors 实现）。
// 应用层通过该接口埋点，不直接依赖全局基础设施；可为 nil（不埋点）。
type MessageMetrics interface {
	// ObserveSeqAlloc 记录一次每会话 seq 分配耗时。
	ObserveSeqAlloc(ctx context.Context, d time.Duration)
	// ObserveMessageLatency 记录一次发消息端到端处理耗时。
	ObserveMessageLatency(ctx context.Context, d time.Duration)
}
