package eventbus_test

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"

	"github.com/maguowei/gotobeta/internal/infra/eventbus"
	"github.com/maguowei/gotobeta/internal/infra/metrics"
	"github.com/maguowei/gotobeta/internal/pkg/event"
)

type sampleEvent struct {
	event.BaseEvent
}

func newSample(name string) sampleEvent {
	return sampleEvent{BaseEvent: event.NewBaseEvent(name, time.Unix(0, 0))}
}

func newBus() *eventbus.InProc {
	return eventbus.NewInProc(slog.New(slog.DiscardHandler), nil)
}

func TestPublishInvokesSubscribers(t *testing.T) {
	bus := newBus()
	called := 0
	bus.Subscribe("a.created", func(_ context.Context, _ event.Event) error {
		called++
		return nil
	})

	if err := bus.Publish(context.Background(), newSample("a.created")); err != nil {
		t.Fatalf("Publish error: %v", err)
	}
	if called != 1 {
		t.Fatalf("handler called %d times, want 1", called)
	}
}

func TestPublishSwallowsHandlerError(t *testing.T) {
	bus := newBus()
	second := false
	bus.Subscribe("a.created", func(_ context.Context, _ event.Event) error {
		return errors.New("boom")
	})
	bus.Subscribe("a.created", func(_ context.Context, _ event.Event) error {
		second = true
		return nil
	})

	if err := bus.Publish(context.Background(), newSample("a.created")); err != nil {
		t.Fatalf("Publish should swallow handler error, got %v", err)
	}
	if !second {
		t.Fatal("second handler should still run after first errored")
	}
}

func TestPublishUnsubscribedNoPanic(t *testing.T) {
	bus := newBus()
	if err := bus.Publish(context.Background(), newSample("nobody.listening")); err != nil {
		t.Fatalf("Publish error: %v", err)
	}
}

func TestPublishReportsMetrics(t *testing.T) {
	mc := metrics.NewCollectors(prometheus.NewRegistry(), "test")
	bus := eventbus.NewInProc(slog.New(slog.DiscardHandler), mc)
	bus.Subscribe("a.created", func(_ context.Context, _ event.Event) error {
		return nil
	})
	bus.Subscribe("a.created", func(_ context.Context, _ event.Event) error {
		return errors.New("boom")
	})

	if err := bus.Publish(context.Background(), newSample("a.created")); err != nil {
		t.Fatalf("Publish error: %v", err)
	}

	processed := testutil.ToFloat64(mc.EventBusEventsTotal.WithLabelValues("eventbus_inproc", "a.created", "processed"))
	if processed != 1 {
		t.Fatalf("processed metric = %v, want 1", processed)
	}
	failed := testutil.ToFloat64(mc.EventBusEventsTotal.WithLabelValues("eventbus_inproc", "a.created", "error"))
	if failed != 1 {
		t.Fatalf("error metric = %v, want 1", failed)
	}
}
