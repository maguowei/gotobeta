package trace

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

// SetGlobalPropagator 全局设置 W3C TraceContext + Baggage 复合 propagator。
// 由 bootstrap 调用一次；业务代码取用走 otel.GetTextMapPropagator()。
func SetGlobalPropagator() {
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))
}
