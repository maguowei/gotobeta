package sentry

import (
	"strings"
	"testing"

	"github.com/maguowei/gotobeta/internal/infra/config"
)

func TestInitRequiresDSNWhenEnabled(t *testing.T) {
	err := Init(&config.SentryConfig{Enabled: true})
	if err == nil {
		t.Fatal("Init() error = nil, want missing DSN error")
	}
	if !strings.Contains(err.Error(), "dsn") && !strings.Contains(err.Error(), "DSN") {
		t.Fatalf("Init() error = %v, want DSN message", err)
	}
}

func TestInitDisabledIsNoop(t *testing.T) {
	t.Parallel()

	if err := Init(nil); err != nil {
		t.Fatalf("Init(nil) error = %v", err)
	}
	if err := Init(&config.SentryConfig{Enabled: false}); err != nil {
		t.Fatalf("Init(disabled) error = %v", err)
	}
}

func TestFlushWithoutClientDoesNotPanic(t *testing.T) {
	t.Parallel()

	_ = Flush()
}
