package metrics

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

func TestClassifyHTTPStatusUsesBoundedLabels(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		err        error
		want       string
	}{
		{name: "2xx", statusCode: 204, want: "2xx"},
		{name: "3xx", statusCode: 302, want: "3xx"},
		{name: "4xx", statusCode: 404, want: "4xx"},
		{name: "5xx", statusCode: 503, want: "5xx"},
		{name: "timeout", err: context.DeadlineExceeded, want: "timeout"},
		{name: "zero status without error", want: "network_error"},
		{name: "network error", err: errors.New("dial tcp failed"), want: "network_error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ClassifyHTTPStatus(tt.statusCode, tt.err); got != tt.want {
				t.Fatalf("ClassifyHTTPStatus() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNewCollectorsRegistersMetrics(t *testing.T) {
	t.Parallel()

	reg := prometheus.NewRegistry()
	mc := NewCollectors(reg, "test_svc")

	if mc.HTTPRequestsTotal == nil {
		t.Fatal("HTTPRequestsTotal is nil")
	}
	if mc.HTTPRequestDuration == nil {
		t.Fatal("HTTPRequestDuration is nil")
	}
	if mc.ExternalCallTotal == nil {
		t.Fatal("ExternalCallTotal is nil")
	}
	if mc.ExternalCallDuration == nil {
		t.Fatal("ExternalCallDuration is nil")
	}
	if mc.EventBusEventsTotal == nil {
		t.Fatal("EventBusEventsTotal is nil")
	}
	if mc.EventBusDuration == nil {
		t.Fatal("EventBusDuration is nil")
	}

	mc.ObserveExternalCall(context.Background(), "example", "ping", time.Millisecond, 200, nil)
	mc.ObserveExternalCall(context.Background(), "example", "timeout", time.Millisecond, 0, context.DeadlineExceeded)
	mc.ObserveEventBus(context.Background(), "outbox", "todo.created", "success", time.Millisecond)
}

func TestCollectorsNilSafe(t *testing.T) {
	var mc *Collectors
	mc.ObserveExternalCall(context.Background(), "example", "ping", time.Millisecond, 200, nil)
	mc.ObserveEventBus(context.Background(), "inbox", "todo.created", "success", time.Millisecond)
}

func TestNewCollectorsIncludesRuntimeMetrics(t *testing.T) {
	t.Parallel()

	reg := prometheus.NewRegistry()
	_ = NewCollectors(reg, "test_rt")

	families, err := reg.Gather()
	if err != nil {
		t.Fatalf("gather metrics: %v", err)
	}

	want := map[string]bool{"go_goroutines": false, "process_cpu_seconds_total": false}
	for _, f := range families {
		if _, ok := want[f.GetName()]; ok {
			want[f.GetName()] = true
		}
	}
	for name, found := range want {
		if !found {
			t.Errorf("runtime metric %q not found", name)
		}
	}
}
