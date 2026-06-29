package requestctx

import (
	"context"
	"testing"
)

func TestRequestContextStoresRequestIDAndCaller(t *testing.T) {
	ctx := WithRequestID(context.Background(), "req-123")
	ctx = WithCaller(ctx, "alice", "user")

	if got := GetRequestID(ctx); got != "req-123" {
		t.Fatalf("GetRequestID() = %q, want %q", got, "req-123")
	}

	caller, callerType := GetCaller(ctx)
	if caller != "alice" || callerType != "user" {
		t.Fatalf("GetCaller() = (%q, %q), want (%q, %q)", caller, callerType, "alice", "user")
	}
}

func TestRequestContextDefaultsAnonymousCaller(t *testing.T) {
	caller, callerType := GetCaller(context.Background())
	if caller != "anonymous" || callerType != "anonymous" {
		t.Fatalf("GetCaller() = (%q, %q), want anonymous caller", caller, callerType)
	}
}
