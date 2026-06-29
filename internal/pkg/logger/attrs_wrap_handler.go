package logger

import (
	"context"
	"log/slog"
)

type attrsWrapHandler struct {
	inner slog.Handler
}

func newAttrsWrapHandler(inner slog.Handler) *attrsWrapHandler {
	return &attrsWrapHandler{inner: inner}
}

func (h *attrsWrapHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.inner.Enabled(ctx, level)
}

func (h *attrsWrapHandler) Handle(ctx context.Context, record slog.Record) error {
	if record.NumAttrs() == 0 {
		return h.inner.Handle(ctx, record)
	}

	attrs := make([]any, 0, record.NumAttrs())
	record.Attrs(func(attr slog.Attr) bool {
		attrs = append(attrs, attr)

		return true
	})

	next := slog.NewRecord(record.Time, record.Level, record.Message, record.PC)
	next.AddAttrs(slog.Group("attrs", attrs...))

	return h.inner.Handle(ctx, next)
}

func (h *attrsWrapHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return newAttrsWrapHandler(h.inner.WithAttrs(attrs))
}

func (h *attrsWrapHandler) WithGroup(name string) slog.Handler {
	return newAttrsWrapHandler(h.inner.WithGroup(name))
}
