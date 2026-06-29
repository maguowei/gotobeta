package logger

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"

	sentrysdk "github.com/getsentry/sentry-go"
)

// Options 是日志初始化参数。
type Options struct {
	Level       string
	Path        string
	AppName     string
	AppEnv      string
	ProcessName string // server / worker / migrate / datainit
	AddSource   bool
}

// New 创建应用日志。
func New(opts Options) (*slog.Logger, io.Closer, error) {
	if err := os.MkdirAll(opts.Path, 0o755); err != nil {
		return nil, nil, err
	}

	file, err := os.OpenFile(filepath.Join(opts.Path, "app.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, nil, err
	}

	handlerOpts := &slog.HandlerOptions{Level: parseLevel(opts.Level), AddSource: opts.AddSource}
	handlers := []slog.Handler{
		slog.NewJSONHandler(file, handlerOpts),
		slog.NewTextHandler(os.Stdout, handlerOpts),
	}
	if sentrysdk.CurrentHub().Client() != nil {
		handlers = append(handlers, &sentryHandler{level: slog.LevelError})
	}

	inner := slog.NewMultiHandler(handlers...)
	var h slog.Handler = newContextHandler(inner)
	if opts.AddSource {
		h = newSourceHandler(h)
	}
	h = newAttrsWrapHandler(h)
	return slog.New(h).With(
		slog.String("logType", "app"),
		slog.String("appName", opts.AppName),
		slog.String("appEnv", opts.AppEnv),
		slog.String("process", opts.ProcessName),
	), file, nil
}

// NewAuditLogger 创建审计日志。
func NewAuditLogger(opts Options) (*slog.Logger, io.Closer, error) {
	if err := os.MkdirAll(opts.Path, 0o755); err != nil {
		return nil, nil, err
	}

	file, err := os.OpenFile(filepath.Join(opts.Path, "audit.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, nil, err
	}

	handler := newContextHandler(slog.NewJSONHandler(file, &slog.HandlerOptions{Level: slog.LevelInfo}))
	return slog.New(handler).With(
		slog.String("logType", "audit"),
		slog.String("appName", opts.AppName),
		slog.String("appEnv", opts.AppEnv),
		slog.String("process", opts.ProcessName),
	), file, nil
}

// NewWithWriter 创建写入指定 writer 的日志器，便于单元测试。
func NewWithWriter(opts Options, writer io.Writer) *slog.Logger {
	handler := slog.NewJSONHandler(writer, &slog.HandlerOptions{Level: parseLevel(opts.Level)})

	return slog.New(newAttrsWrapHandler(newContextHandler(handler))).With(
		slog.String("appName", opts.AppName),
		slog.String("appEnv", opts.AppEnv),
	)
}

func parseLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

type sentryHandler struct {
	level  slog.Level
	attrs  []slog.Attr
	groups []string
}

func (h *sentryHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level
}

func (h *sentryHandler) Handle(ctx context.Context, record slog.Record) error {
	hub := sentrysdk.GetHubFromContext(ctx)
	if hub == nil {
		hub = sentrysdk.CurrentHub()
	}
	if hub.Client() == nil {
		return nil
	}

	// 把通过 logger.With(...) 累积的 attrs 与本条 record 的 attrs 合并到 Sentry scope，
	// 保留 process / appName / appEnv 等关键运维上下文；丢弃这些字段会让 Sentry 告警无法定位进程。
	// 注意：attrsWrapHandler 会把 record.Attrs 包成 slog.Group("attrs", ...)，
	// 这里必须递归展开 group，否则 Sentry context 里会出现 `[]slog.Attr` 这种不可读类型。
	logCtx := map[string]any{
		"level":   record.Level.String(),
		"message": record.Message,
	}
	for _, attr := range h.attrs {
		h.addAttr(logCtx, attr)
	}
	record.Attrs(func(attr slog.Attr) bool {
		h.addAttr(logCtx, attr)
		return true
	})

	hub.WithScope(func(scope *sentrysdk.Scope) {
		scope.SetLevel(sentrysdk.LevelError)
		scope.SetContext("log", logCtx)
		hub.CaptureMessage(record.Message)
	})

	return nil
}

// addAttr 把 slog.Attr 平铺到 Sentry context。遇到 Group 时递归展开为嵌套 map，
// 让运维在 Sentry UI 上看到 `db.dsn` 这样的结构而不是 Go 类型的字符串化。
// 特殊处理 attrsWrapHandler 注入的 "attrs" 顶层组：把其内部字段平铺到外层，
// 与 JSON/Text handler 的视觉一致性由各自渲染规则保证，这里追求 Sentry 上的可读性。
func (h *sentryHandler) addAttr(dst map[string]any, attr slog.Attr) {
	value := attr.Value.Resolve()
	if value.Kind() == slog.KindGroup {
		groupAttrs := value.Group()
		if attr.Key == "attrs" && len(h.groups) == 0 {
			for _, child := range groupAttrs {
				h.addAttr(dst, child)
			}
			return
		}
		nested := make(map[string]any, len(groupAttrs))
		for _, child := range groupAttrs {
			h.addAttr(nested, child)
		}
		dst[h.qualify(attr.Key)] = nested
		return
	}
	dst[h.qualify(attr.Key)] = value.Any()
}

func (h *sentryHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return h
	}
	clone := h.clone()
	clone.attrs = append(clone.attrs, attrs...)
	return clone
}

func (h *sentryHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	clone := h.clone()
	clone.groups = append(clone.groups, name)
	return clone
}

func (h *sentryHandler) clone() *sentryHandler {
	return &sentryHandler{
		level:  h.level,
		attrs:  slices.Clone(h.attrs),
		groups: slices.Clone(h.groups),
	}
}

func (h *sentryHandler) qualify(key string) string {
	if len(h.groups) == 0 {
		return key
	}
	return strings.Join(slices.Concat(h.groups, []string{key}), ".")
}
