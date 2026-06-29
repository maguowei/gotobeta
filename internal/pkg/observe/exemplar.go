package observe

import (
	"context"

	"github.com/prometheus/client_golang/prometheus"
	otel_trace "go.opentelemetry.io/otel/trace"
)

// ObserveWithTraceID 在有 trace 时写入 exemplar，否则普通 observe。
// 使用 comma-ok 类型断言确保在 Observer 不支持 exemplar 时优雅降级。
func ObserveWithTraceID(ctx context.Context, observer prometheus.Observer, value float64) {
	if sc := otel_trace.SpanContextFromContext(ctx); sc.IsValid() {
		if eo, ok := observer.(prometheus.ExemplarObserver); ok {
			eo.ObserveWithExemplar(value, prometheus.Labels{"traceID": sc.TraceID().String()})
			return
		}
	}
	observer.Observe(value)
}
