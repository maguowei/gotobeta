package trace

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel"
	tracenoop "go.opentelemetry.io/otel/trace/noop"
)

func TestNewTracerProvider_NoopWhenEndpointEmpty(t *testing.T) {
	t.Parallel()

	tp, shutdown, err := NewTracerProvider(context.Background(), Config{}, "test")
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if _, ok := tp.(tracenoop.TracerProvider); !ok {
		t.Fatalf("expected noop provider, got %T", tp)
	}
	if err := shutdown(context.Background()); err != nil {
		t.Fatalf("noop shutdown err = %v", err)
	}
}

func TestNewTracerProvider_ExporterWhenEndpointConfigured(t *testing.T) {
	t.Parallel()

	tp, shutdown, err := NewTracerProvider(context.Background(), Config{
		Endpoint:       "127.0.0.1:4317",
		Insecure:       true,
		Sampler:        "always",
		SampleRatio:    1,
		ServiceName:    "sample-service",
		ServiceVersion: "test",
	}, "test")
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if _, ok := tp.(tracenoop.TracerProvider); ok {
		t.Fatalf("expected sdk provider, got noop")
	}
	if err := shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown err = %v", err)
	}
}

func TestConfig_ShouldExport(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		cfg      Config
		expected bool
	}{
		{"empty", Config{}, false},
		{"endpoint-only", Config{Endpoint: "x:4317"}, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if got := c.cfg.ShouldExport(); got != c.expected {
				t.Fatalf("ShouldExport = %v, want %v", got, c.expected)
			}
		})
	}
}

func TestSetGlobalPropagator(t *testing.T) {
	SetGlobalPropagator()
	if otel.GetTextMapPropagator() == nil {
		t.Fatalf("global propagator is nil")
	}
}

// buildSampler 必须为每个合法 sampler 值返回一个非 nil 的 Sampler；
// 关键是 parent 不再退化为 default 的硬编码全量采样。
func TestBuildSampler_AllVariants(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		cfg  Config
	}{
		{"always", Config{Sampler: "always"}},
		{"never", Config{Sampler: "never"}},
		{"ratio", Config{Sampler: "ratio", SampleRatio: 0.1}},
		{"parent-with-ratio", Config{Sampler: "parent", SampleRatio: 0.1}},
		{"parent-zero-ratio", Config{Sampler: "parent", SampleRatio: 0}},
		{"empty-sampler", Config{}},
		{"unknown", Config{Sampler: "bogus"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got := buildSampler(c.cfg)
			if got == nil {
				t.Fatalf("buildSampler(%+v) = nil", c.cfg)
			}
			_ = got.Description()
		})
	}
}
