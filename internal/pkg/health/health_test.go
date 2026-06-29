package health

import (
	"context"
	"errors"
	"testing"
)

func TestRunAllReturnsOKWhenNothingRegistered(t *testing.T) {
	t.Parallel()
	r := NewRegistry()
	result := r.RunAll(context.Background())
	if result.Status != "ok" {
		t.Fatalf("status = %q, want ok", result.Status)
	}
	if len(result.Checks) != 0 {
		t.Fatalf("checks = %v, want empty", result.Checks)
	}
}

func TestRunAllReturnsOKWhenAllPass(t *testing.T) {
	t.Parallel()
	r := NewRegistry()
	r.Register("db", CheckerFunc(func(context.Context) error { return nil }))
	r.Register("cache", CheckerFunc(func(context.Context) error { return nil }))

	result := r.RunAll(context.Background())
	if result.Status != "ok" {
		t.Fatalf("status = %q, want ok", result.Status)
	}
	if result.Checks["db"] != "ok" || result.Checks["cache"] != "ok" {
		t.Fatalf("checks = %v, want all ok", result.Checks)
	}
}

func TestRunAllReturnsDegradedWhenAnyFails(t *testing.T) {
	t.Parallel()
	r := NewRegistry()
	r.Register("db", CheckerFunc(func(context.Context) error { return nil }))
	r.Register("cache", CheckerFunc(func(context.Context) error { return errors.New("connection refused") }))

	result := r.RunAll(context.Background())
	if result.Status != "degraded" {
		t.Fatalf("status = %q, want degraded", result.Status)
	}
	if result.Checks["db"] != "ok" {
		t.Fatalf("db check = %q, want ok", result.Checks["db"])
	}
	if result.Checks["cache"] != "connection refused" {
		t.Fatalf("cache check = %q, want connection refused", result.Checks["cache"])
	}
}
