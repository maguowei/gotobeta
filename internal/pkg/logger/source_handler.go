package logger

import (
	"context"
	"log/slog"
	"runtime"
)

// sourceHandler 在 record 上附加 source(file:line:func) 属性。
// 启用条件由 Options.AddSource 决定；slog 自身也有 AddSource 选项，
// 但其只走 HandlerOptions 路径；本 handler 适用于多 handler 复合时保持一致。
type sourceHandler struct {
	inner slog.Handler
}

func newSourceHandler(inner slog.Handler) *sourceHandler {
	return &sourceHandler{inner: inner}
}

func (h *sourceHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.inner.Enabled(ctx, level)
}

func (h *sourceHandler) Handle(ctx context.Context, record slog.Record) error {
	if record.PC != 0 {
		fs := runtime.CallersFrames([]uintptr{record.PC})
		f, _ := fs.Next()
		if f.File != "" {
			record.AddAttrs(slog.Group("source",
				slog.String("file", f.File),
				slog.Int("line", f.Line),
				slog.String("func", f.Function),
			))
		}
	}
	return h.inner.Handle(ctx, record)
}

func (h *sourceHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return newSourceHandler(h.inner.WithAttrs(attrs))
}

func (h *sourceHandler) WithGroup(name string) slog.Handler {
	return newSourceHandler(h.inner.WithGroup(name))
}
