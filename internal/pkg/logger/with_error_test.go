package logger

import (
	"bytes"
	"context"
	"encoding/json"
	stderrors "errors"
	"log/slog"
	"testing"

	"github.com/maguowei/gotobeta/internal/pkg/apperr"
)

func TestWithError_DomainErrorExpandsLogAttrs(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	l := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	WithError(context.Background(), l, "boom", apperr.Internal("x", stderrors.New("root")).WithCode("E1"),
		slog.String("user", "alice"),
	)

	var record map[string]any
	if err := json.Unmarshal(buf.Bytes(), &record); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if record["errKind"] != "Internal" {
		t.Errorf("errKind = %v", record["errKind"])
	}
	if record["errCode"] != "E1" {
		t.Errorf("errCode = %v", record["errCode"])
	}
	if record["errCause"] != "root" {
		t.Errorf("errCause = %v", record["errCause"])
	}
	if record["user"] != "alice" {
		t.Errorf("user = %v", record["user"])
	}
	if record["msg"] != "boom" {
		t.Errorf("msg = %v", record["msg"])
	}
	if record["level"] != "ERROR" {
		t.Errorf("level = %v", record["level"])
	}
}

func TestWithError_PlainErrorFallsBack(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	l := slog.New(slog.NewJSONHandler(&buf, nil))
	WithError(context.Background(), l, "io failed", stderrors.New("disk full"))

	var record map[string]any
	_ = json.Unmarshal(buf.Bytes(), &record)
	if record["error"] != "disk full" {
		t.Errorf("error = %v", record["error"])
	}
}

func TestWithError_NilErrorStillLogs(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	l := slog.New(slog.NewJSONHandler(&buf, nil))
	WithError(context.Background(), l, "noop msg", nil, slog.String("k", "v"))

	var record map[string]any
	_ = json.Unmarshal(buf.Bytes(), &record)
	if record["msg"] != "noop msg" || record["k"] != "v" {
		t.Errorf("record = %#v", record)
	}
}
