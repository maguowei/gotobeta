package trace

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	tracex "go.opentelemetry.io/otel/trace"
	tracenoop "go.opentelemetry.io/otel/trace/noop"
)

// Shutdown 是 TracerProvider 的关闭函数。
type Shutdown func(context.Context) error

// NewTracerProvider 根据 Config 创建 TracerProvider。
// Endpoint 为空时返回 noop provider，shutdown 返回 nil。
func NewTracerProvider(ctx context.Context, cfg Config, appEnv string) (tracex.TracerProvider, Shutdown, error) {
	if !cfg.ShouldExport() {
		return tracenoop.NewTracerProvider(), func(context.Context) error { return nil }, nil
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(cfg.ServiceName),
			semconv.ServiceVersion(cfg.ServiceVersion),
			semconv.DeploymentEnvironment(appEnv),
		),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("build trace resource: %w", err)
	}

	opts := []otlptracegrpc.Option{otlptracegrpc.WithEndpoint(cfg.Endpoint)}
	if cfg.Insecure {
		opts = append(opts, otlptracegrpc.WithInsecure())
	}

	exporter, err := otlptrace.New(ctx, otlptracegrpc.NewClient(opts...))
	if err != nil {
		return nil, nil, fmt.Errorf("create otlp exporter: %w", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(buildSampler(cfg)),
	)

	shutdown := func(ctx context.Context) error {
		shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		return tp.Shutdown(shutdownCtx)
	}
	return tp, shutdown, nil
}

func buildSampler(cfg Config) sdktrace.Sampler {
	switch cfg.Sampler {
	case "always":
		return sdktrace.AlwaysSample()
	case "never":
		return sdktrace.NeverSample()
	case "ratio":
		return sdktrace.ParentBased(sdktrace.TraceIDRatioBased(cfg.SampleRatio))
	case "parent", "":
		// 跟随父 span 的采样决定，无父 span 时按 SampleRatio 决定。
		// SampleRatio 非法（<=0 或 >1）时退化为不采样新 trace，避免默认全量上报。
		ratio := cfg.SampleRatio
		if ratio <= 0 || ratio > 1 {
			ratio = 0
		}
		return sdktrace.ParentBased(sdktrace.TraceIDRatioBased(ratio))
	default:
		// Validate() 已限制为 always/never/parent/ratio；走到这里说明配置异常，按最保守策略不采样。
		return sdktrace.ParentBased(sdktrace.NeverSample())
	}
}
