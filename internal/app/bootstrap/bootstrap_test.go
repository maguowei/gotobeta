package bootstrap

import (
	"context"
	stderrors "errors"
	"testing"
)

func TestInitReturnsConfigError(t *testing.T) {
	t.Setenv("APP_CONFIG_DIR", t.TempDir())
	t.Setenv("APP_LOGGER_LEVEL", "trace")

	_, err := Init(context.Background(), Options{ProcessName: "test"})
	if err == nil {
		t.Fatalf("Init() error = nil, want config validation error")
	}
}

// errClose 实现 closer 模拟。
type errClose struct {
	name string
	err  error
	out  *[]string
}

func (c errClose) close(_ context.Context) error {
	*c.out = append(*c.out, c.name)
	return c.err
}

func TestRuntime_ShutdownLIFO(t *testing.T) {
	t.Parallel()
	var out []string
	rt := &Runtime{
		closers: []func(context.Context) error{
			errClose{name: "first", out: &out}.close,
			errClose{name: "second", out: &out}.close,
			errClose{name: "third", out: &out}.close,
		},
	}
	if err := rt.Shutdown(context.Background()); err != nil {
		t.Fatalf("err = %v", err)
	}
	if len(out) != 3 || out[0] != "third" || out[1] != "second" || out[2] != "first" {
		t.Fatalf("LIFO violated: %v", out)
	}
}

func TestRuntime_ShutdownJoinsErrors(t *testing.T) {
	t.Parallel()
	var out []string
	e1 := stderrors.New("e1")
	e2 := stderrors.New("e2")
	rt := &Runtime{
		closers: []func(context.Context) error{
			errClose{name: "a", err: e1, out: &out}.close,
			errClose{name: "b", err: e2, out: &out}.close,
		},
	}
	err := rt.Shutdown(context.Background())
	if !stderrors.Is(err, e1) || !stderrors.Is(err, e2) {
		t.Fatalf("expected joined errors, got %v", err)
	}
}

func TestRuntime_ShutdownEmptyNoError(t *testing.T) {
	t.Parallel()
	rt := &Runtime{}
	if err := rt.Shutdown(context.Background()); err != nil {
		t.Fatalf("empty shutdown err = %v", err)
	}
}

func TestInitAndShutdown(t *testing.T) {
	t.Setenv("APP_CONFIG_DIR", t.TempDir())
	t.Setenv("APP_LOGGER_PATH", t.TempDir())
	t.Setenv("APP_SENTRY_ENABLED", "false")
	t.Setenv("APP_DATABASE_DSN", "root:password@tcp(127.0.0.1:3306)/gotobeta?parseTime=true&charset=utf8mb4")
	// hmac_secret 无默认值（fail-closed），与 DSN 同理需显式注入；用 >=32 字节满足所有环境校验。
	t.Setenv("APP_AUTH_JWT_HMAC_SECRET", "test-hmac-secret-at-least-32-bytes-long")

	rt, err := Init(context.Background(), Options{ProcessName: "test", EnableTracer: false})
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	if rt.Cfg == nil {
		t.Fatalf("Cfg is nil")
	}
	if rt.AppLogger == nil {
		t.Fatalf("AppLogger is nil")
	}
	if rt.AuditLogger == nil {
		t.Fatalf("AuditLogger is nil")
	}
	if rt.TracerProvider == nil {
		t.Fatalf("TracerProvider is nil")
	}
	if err := rt.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}
}
