package main

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/maguowei/gotobeta/internal/app/bootstrap"
)

func TestMigrateSmoke(t *testing.T) {
	t.Log("migrate 命令编译通过")
}

func TestRunMigrateUsesBootstrapAndMigrator(t *testing.T) {
	originalBootstrapInit := bootstrapInit
	originalRunMigrate := runMigrate
	t.Cleanup(func() {
		bootstrapInit = originalBootstrapInit
		runMigrate = originalRunMigrate
	})

	bootstrapCalled := false
	migrateCalled := false
	bootstrapInit = func(context.Context, bootstrap.Options) (*bootstrap.Runtime, error) {
		bootstrapCalled = true
		return &bootstrap.Runtime{AppLogger: slog.New(slog.DiscardHandler)}, nil
	}
	runMigrate = func(context.Context, *bootstrap.Runtime) error {
		migrateCalled = true
		return nil
	}

	if err := run(); err != nil {
		t.Fatalf("run() error = %v", err)
	}
	if !bootstrapCalled || !migrateCalled {
		t.Fatalf("bootstrapCalled=%v migrateCalled=%v, want both true", bootstrapCalled, migrateCalled)
	}
}

func TestRunMigrateReturnsBootstrapError(t *testing.T) {
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

func TestRunMigrateReturnsMigrationError(t *testing.T) {
	originalBootstrapInit := bootstrapInit
	originalRunMigrate := runMigrate
	t.Cleanup(func() {
		bootstrapInit = originalBootstrapInit
		runMigrate = originalRunMigrate
	})

	wantErr := errors.New("migration failed")
	bootstrapInit = func(context.Context, bootstrap.Options) (*bootstrap.Runtime, error) {
		return &bootstrap.Runtime{AppLogger: slog.New(slog.DiscardHandler)}, nil
	}
	runMigrate = func(context.Context, *bootstrap.Runtime) error {
		return wantErr
	}

	if err := run(); !errors.Is(err, wantErr) {
		t.Fatalf("run() error = %v, want %v", err, wantErr)
	}
}
