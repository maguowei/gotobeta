package logger

import (
	"context"
	"log/slog"
)

type loggerCtxKey struct{}

// ToContext 把 logger 写入 ctx，供下游 FromContext 取出。
func ToContext(ctx context.Context, l *slog.Logger) context.Context {
	return context.WithValue(ctx, loggerCtxKey{}, l)
}

// FromContext 优先返回 ctx 中携带的 logger（HTTP 中间件注入的请求级 logger），
// 否则返回 slog.Default()。
func FromContext(ctx context.Context) *slog.Logger {
	if l, ok := ctx.Value(loggerCtxKey{}).(*slog.Logger); ok && l != nil {
		return l
	}
	return slog.Default()
}
