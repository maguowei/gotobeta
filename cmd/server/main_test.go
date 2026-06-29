package main

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/maguowei/gotobeta/internal/app/bootstrap"
)

func TestRunServerUsesBootstrapAndHTTPRunner(t *testing.T) {
	originalBootstrapInit := bootstrapInit
	originalRunHTTP := runHTTP
	t.Cleanup(func() {
		bootstrapInit = originalBootstrapInit
		runHTTP = originalRunHTTP
	})

	bootstrapCalled := false
	httpCalled := false
	bootstrapInit = func(context.Context, bootstrap.Options) (*bootstrap.Runtime, error) {
		bootstrapCalled = true
		return &bootstrap.Runtime{AppLogger: slog.New(slog.DiscardHandler)}, nil
	}
	runHTTP = func(context.Context, *bootstrap.Runtime) error {
		httpCalled = true
		return nil
	}

	if err := run(); err != nil {
		t.Fatalf("run() error = %v", err)
	}
	if !bootstrapCalled || !httpCalled {
		t.Fatalf("bootstrapCalled=%v httpCalled=%v, want both true", bootstrapCalled, httpCalled)
	}
}

func TestRunServerReturnsBootstrapError(t *testing.T) {
	originalBootstrapInit := bootstrapInit
	t.Cleanup(func() {
		bootstrapInit = originalBootstrapInit
	})

	wantErr := errors.New("config invalid")
	bootstrapInit = func(context.Context, bootstrap.Options) (*bootstrap.Runtime, error) {
		return nil, wantErr
	}

	if err := run(); !errors.Is(err, wantErr) {
		t.Fatalf("run() error = %v, want %v", err, wantErr)
	}
}

func TestRunServerReturnsHTTPError(t *testing.T) {
	originalBootstrapInit := bootstrapInit
	originalRunHTTP := runHTTP
	t.Cleanup(func() {
		bootstrapInit = originalBootstrapInit
		runHTTP = originalRunHTTP
	})

	wantErr := errors.New("listen failed")
	bootstrapInit = func(context.Context, bootstrap.Options) (*bootstrap.Runtime, error) {
		return &bootstrap.Runtime{AppLogger: slog.New(slog.DiscardHandler)}, nil
	}
	runHTTP = func(context.Context, *bootstrap.Runtime) error {
		return wantErr
	}

	if err := run(); !errors.Is(err, wantErr) {
		t.Fatalf("run() error = %v, want %v", err, wantErr)
	}
}
