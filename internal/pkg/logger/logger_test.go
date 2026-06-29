package logger

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	oteltrace "go.opentelemetry.io/otel/trace"

	"github.com/maguowei/gotobeta/internal/pkg/requestctx"
)

func TestLoggerWrapsPerCallAttrsAndInjectsRequestID(t *testing.T) {
	var buf bytes.Buffer
	log := NewWithWriter(Options{
		Level:   "debug",
		AppName: "codego",
		AppEnv:  "test",
	}, &buf).With(slog.String("component", "unit"))

	ctx := requestctx.WithRequestID(context.Background(), "req-456")
	log.InfoContext(ctx, "hello", slog.String("user", "alice"), slog.Int("count", 2))

	var entry map[string]any
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("unmarshal log entry: %v", err)
	}

	if entry["requestId"] != "req-456" {
		t.Fatalf("requestId = %v, want req-456", entry["requestId"])
	}
	if entry["component"] != "unit" {
		t.Fatalf("component = %v, want unit", entry["component"])
	}
	if _, ok := entry["user"]; ok {
		t.Fatalf("per-call attr leaked to top level: %#v", entry)
	}

	attrs, ok := entry["attrs"].(map[string]any)
	if !ok {
		t.Fatalf("attrs missing or wrong type: %#v", entry["attrs"])
	}
	if attrs["user"] != "alice" || attrs["count"] != float64(2) {
		t.Fatalf("attrs = %#v, want user/count", attrs)
	}
}

func TestNewCreatesAppAndAuditLogFiles(t *testing.T) {
	logDir := t.TempDir()

	appLogger, appCloser, err := New(Options{
		Level:       "debug",
		Path:        logDir,
		AppName:     "codego",
		AppEnv:      "test",
		ProcessName: "server",
		AddSource:   true,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	t.Cleanup(func() {
		if err := appCloser.Close(); err != nil {
			t.Errorf("close app log: %v", err)
		}
	})
	appLogger.InfoContext(context.Background(), "app log")

	auditLogger, auditCloser, err := NewAuditLogger(Options{
		Path:        logDir,
		AppName:     "codego",
		AppEnv:      "test",
		ProcessName: "server",
	})
	if err != nil {
		t.Fatalf("NewAuditLogger() error = %v", err)
	}
	t.Cleanup(func() {
		if err := auditCloser.Close(); err != nil {
			t.Errorf("close audit log: %v", err)
		}
	})
	auditLogger.InfoContext(context.Background(), "audit log")

	for _, name := range []string{"app.log", "audit.log"} {
		if _, err := os.Stat(filepath.Join(logDir, name)); err != nil {
			t.Fatalf("stat %s error = %v", name, err)
		}
	}
}

func TestNewRejectsFileAsLogDirectory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "logs")
	if err := os.WriteFile(path, []byte("not a dir"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if _, _, err := New(Options{Path: path}); err == nil {
		t.Fatalf("New() error = nil, want mkdir failure")
	}
	if _, _, err := NewAuditLogger(Options{Path: path}); err == nil {
		t.Fatalf("NewAuditLogger() error = nil, want mkdir failure")
	}
}

func TestParseLevelVariants(t *testing.T) {
	tests := map[string]slog.Level{
		"debug": slog.LevelDebug,
		"warn":  slog.LevelWarn,
		"error": slog.LevelError,
		"info":  slog.LevelInfo,
		"DEBUG": slog.LevelDebug,
		"":      slog.LevelInfo,
	}

	for raw, want := range tests {
		if got := parseLevel(raw); got != want {
			t.Fatalf("parseLevel(%q) = %v, want %v", raw, got, want)
		}
	}
}

func TestLoggerContextHelpersAndTraceAttrs(t *testing.T) {
	var buf bytes.Buffer
	log := NewWithWriter(Options{Level: "debug", AppName: "codego", AppEnv: "test"}, &buf)

	if got := FromContext(context.Background()); got != slog.Default() {
		t.Fatalf("FromContext(empty) should return slog.Default")
	}
	ctx := ToContext(context.Background(), log)
	if got := FromContext(ctx); got != log {
		t.Fatalf("FromContext(ctx) did not return injected logger")
	}

	traceID := oteltrace.TraceID{1, 2, 3}
	spanID := oteltrace.SpanID{4, 5, 6}
	spanCtx := oteltrace.NewSpanContext(oteltrace.SpanContextConfig{
		TraceID: traceID,
		SpanID:  spanID,
		Remote:  true,
	})
	ctx = oteltrace.ContextWithSpanContext(ctx, spanCtx)
	log.InfoContext(ctx, "hello")

	var entry map[string]any
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("unmarshal log entry: %v", err)
	}
	if entry["traceId"] != traceID.String() || entry["spanId"] != spanID.String() {
		t.Fatalf("trace attrs = (%v, %v), want (%s, %s)", entry["traceId"], entry["spanId"], traceID.String(), spanID.String())
	}
	if _, ok := entry["attrs"]; ok {
		t.Fatalf("attrs should be omitted when record has no per-call attrs: %#v", entry["attrs"])
	}
}

// stubHandler 让 slog.MultiHandler 行为可观测：第 1 个故意返回 error，第 2 个必须仍被调用。
type stubHandler struct {
	called bool
	err    error
}

func (h *stubHandler) Enabled(context.Context, slog.Level) bool { return true }
func (h *stubHandler) Handle(context.Context, slog.Record) error {
	h.called = true
	return h.err
}
func (h *stubHandler) WithAttrs([]slog.Attr) slog.Handler { return h }
func (h *stubHandler) WithGroup(string) slog.Handler      { return h }

func TestMultiHandlerDoesNotShortCircuit(t *testing.T) {
	failing := &stubHandler{err: errors.New("disk full")}
	fallback := &stubHandler{}

	mh := slog.NewMultiHandler(failing, fallback)
	err := mh.Handle(context.Background(), slog.NewRecord(time.Now(), slog.LevelInfo, "msg", 0))

	if !failing.called || !fallback.called {
		t.Fatalf("both handlers must be called; failing=%v fallback=%v", failing.called, fallback.called)
	}
	if err == nil || !errors.Is(err, failing.err) {
		t.Fatalf("multi handler should join failing handler error, got %v", err)
	}
}

func TestSourceHandlerAddsCallSite(t *testing.T) {
	var buf bytes.Buffer
	log := slog.New(newSourceHandler(slog.NewJSONHandler(&buf, nil)))

	log.InfoContext(context.Background(), "hello")

	var entry map[string]any
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("unmarshal log entry: %v", err)
	}

	source, ok := entry["source"].(map[string]any)
	if !ok {
		t.Fatalf("source missing or wrong type: %#v", entry["source"])
	}
	if source["file"] == "" {
		t.Fatalf("source.file is empty: %#v", source)
	}
	if source["func"] == "" {
		t.Fatalf("source.func is empty: %#v", source)
	}
	if source["line"] == nil {
		t.Fatalf("source.line is missing: %#v", source)
	}
}

func TestSentryHandlerKeepsAttrsAndGroupsImmutable(t *testing.T) {
	base := &sentryHandler{}
	withAttrs := base.WithAttrs([]slog.Attr{slog.String("component", "worker")}).(*sentryHandler)
	withGroup := withAttrs.WithGroup("db").(*sentryHandler)

	if len(base.attrs) != 0 || len(base.groups) != 0 {
		t.Fatalf("base handler mutated: attrs=%v groups=%v", base.attrs, base.groups)
	}
	if got := withGroup.qualify("dsn"); got != "db.dsn" {
		t.Fatalf("qualified key = %q, want db.dsn", got)
	}

	dst := map[string]any{}
	withGroup.addAttr(dst, slog.Group("attrs", slog.String("tenant", "acme")))
	attrs, ok := dst["db.attrs"].(map[string]any)
	if !ok {
		t.Fatalf("db.attrs missing or wrong type: %#v", dst)
	}
	if attrs["db.tenant"] != "acme" {
		t.Fatalf("nested attrs = %#v, want tenant", attrs)
	}

	flat := map[string]any{}
	base.addAttr(flat, slog.Group("attrs", slog.String("tenant", "acme")))
	if flat["tenant"] != "acme" {
		t.Fatalf("flat attrs = %#v, want top-level tenant", flat)
	}
}

func TestHandlerWithGroupAndEnabledBranches(t *testing.T) {
	handler := &stubHandler{}

	if wrapped := newAttrsWrapHandler(handler).WithGroup("request"); wrapped == nil {
		t.Fatalf("attrsWrapHandler.WithGroup() returned nil")
	}
	if wrapped := newContextHandler(handler).WithGroup("request"); wrapped == nil {
		t.Fatalf("contextHandler.WithGroup() returned nil")
	}
	if wrapped := newSourceHandler(handler).WithAttrs([]slog.Attr{slog.String("component", "test")}); wrapped == nil {
		t.Fatalf("sourceHandler.WithAttrs() returned nil")
	}
	if wrapped := newSourceHandler(handler).WithGroup("request"); wrapped == nil {
		t.Fatalf("sourceHandler.WithGroup() returned nil")
	}

	mh := slog.NewMultiHandler(handler)
	if !mh.Enabled(context.Background(), slog.LevelInfo) {
		t.Fatalf("MultiHandler.Enabled() = false, want true")
	}
	if mh.WithAttrs([]slog.Attr{slog.String("k", "v")}) == nil {
		t.Fatalf("MultiHandler.WithAttrs() returned nil")
	}
	if mh.WithGroup("request") == nil {
		t.Fatalf("MultiHandler.WithGroup() returned nil")
	}

	if (&sentryHandler{level: slog.LevelError}).Enabled(context.Background(), slog.LevelInfo) {
		t.Fatalf("sentryHandler.Enabled(info) = true, want false")
	}
	if got := (&sentryHandler{}).WithAttrs(nil); got == nil {
		t.Fatalf("sentryHandler.WithAttrs(nil) returned nil")
	}
	if got := (&sentryHandler{}).WithGroup(""); got == nil {
		t.Fatalf("sentryHandler.WithGroup(empty) returned nil")
	}
}
