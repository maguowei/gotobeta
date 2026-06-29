package logger

import (
	"context"
	"log/slog"

	otel_trace "go.opentelemetry.io/otel/trace"

	"github.com/maguowei/gotobeta/internal/pkg/requestctx"
)

type contextHandler struct {
	inner slog.Handler
}

func newContextHandler(inner slog.Handler) *contextHandler {
	return &contextHandler{inner: inner}
}

func (h *contextHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.inner.Enabled(ctx, level)
}

func (h *contextHandler) Handle(ctx context.Context, record slog.Record) error {
	if requestID := requestctx.GetRequestID(ctx); requestID != "" {
		record.AddAttrs(slog.String("requestId", requestID))
	}
	if sc := otel_trace.SpanContextFromContext(ctx); sc.IsValid() {
		record.AddAttrs(
			slog.String("traceId", sc.TraceID().String()),
			slog.String("spanId", sc.SpanID().String()),
		)
	}
	return h.inner.Handle(ctx, record)
}

func (h *contextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return newContextHandler(h.inner.WithAttrs(attrs))
}

func (h *contextHandler) WithGroup(name string) slog.Handler {
	return newContextHandler(h.inner.WithGroup(name))
}
