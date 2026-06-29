package logger

import (
	"context"
	"log/slog"
)

// errorAttrer 是错误自述结构化字段的接口（由 errors.DomainError 实现）。
type errorAttrer interface {
	LogAttrs() []slog.Attr
}

// WithError 是结构化错误日志的唯一入口。
//   - 第一参数 ctx 必传：编译器保证 contextHandler 能注入 traceId/requestId
//   - err 实现 LogAttrs 时自动展开；否则落入 "error" 字段
//   - 多余 attrs 追加在末尾
func WithError(ctx context.Context, l *slog.Logger, msg string, err error, attrs ...slog.Attr) {
	if l == nil {
		l = slog.Default()
	}
	if err == nil {
		//nolint:sloglint // WithError 是日志封装入口，必须透传调用方的业务消息。
		l.LogAttrs(ctx, slog.LevelError, msg, attrs...)
		return
	}

	merged := make([]slog.Attr, 0, len(attrs)+4)
	if a, ok := err.(errorAttrer); ok {
		merged = append(merged, a.LogAttrs()...)
	} else {
		merged = append(merged, slog.String("error", err.Error()))
	}
	merged = append(merged, attrs...)
	//nolint:sloglint // WithError 是日志封装入口，必须透传调用方的业务消息。
	l.LogAttrs(ctx, slog.LevelError, msg, merged...)
}
