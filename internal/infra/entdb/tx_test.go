package entdb

import (
	"context"
	"testing"
)

func TestNewEntTxRunnerStoresClient(t *testing.T) {
	runner := NewEntTxRunner(nil)
	if runner == nil {
		t.Fatalf("NewEntTxRunner() = nil")
	}
	if runner.client != nil {
		t.Fatalf("runner.client = %v, want nil", runner.client)
	}
}

func TestClientFromCtxReturnsFallback(t *testing.T) {
	if got := ClientFromCtx(context.Background(), nil); got != nil {
		t.Fatalf("ClientFromCtx() = %v, want fallback nil", got)
	}
}
