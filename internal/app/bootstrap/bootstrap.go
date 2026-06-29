package bootstrap

import (
	"context"
	stderrors "errors"
	"io"
	"log/slog"

	"github.com/prometheus/client_golang/prometheus"
	otelapi "go.opentelemetry.io/otel"
	tracex "go.opentelemetry.io/otel/trace"

	"github.com/maguowei/gotobeta/internal/infra/config"
	sentryx "github.com/maguowei/gotobeta/internal/infra/sentry"
	"github.com/maguowei/gotobeta/internal/pkg/logger"
	tracepkg "github.com/maguowei/gotobeta/internal/pkg/trace"
)

// Runtime 是所有进程入口共享的运行时句柄。
type Runtime struct {
	Cfg             *config.Config
	AppLogger       *slog.Logger
	AuditLogger     *slog.Logger
	TracerProvider  tracex.TracerProvider
	MetricsRegistry *prometheus.Registry

	closers       []func(context.Context) error
	logFileCloser io.Closer
}

// Options 控制 Init 行为。
type Options struct {
	ProcessName  string // server / worker / migrate / datainit
	EnableTracer bool   // false 时强制 noop（短命进程用）
}

// Init 是所有进程入口的唯一钥匙。
// 任一步失败已成功的 closer 会按 LIFO 释放。
func Init(ctx context.Context, opts Options) (*Runtime, error) {
	rt := &Runtime{}

	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}
	rt.Cfg = cfg

	if err := sentryx.Init(&cfg.Sentry); err != nil {
		rt.shutdownPartial(ctx)
		return nil, err
	}
	if cfg.Sentry.Enabled {
		rt.closers = append(rt.closers, func(context.Context) error {
			// Flush 返回 false 表示超时未刷完，把它合流到 Shutdown 错误中，
			// 让运维能在退出日志看到「Sentry 事件可能丢失」。
			if ok := sentryx.Flush(); !ok {
				return stderrors.New("sentry flush timed out, buffered events may be lost")
			}
			return nil
		})
	}

	// 每个进程持有独立的 metrics registry，避免全局 DefaultRegisterer 的隐式共享。
	rt.MetricsRegistry = prometheus.NewRegistry()

	traceCfg := tracepkg.Config{
		Endpoint:       cfg.Tracing.Endpoint,
		Insecure:       cfg.Tracing.Insecure,
		Sampler:        cfg.Tracing.Sampler,
		SampleRatio:    cfg.Tracing.SampleRatio,
		ServiceName:    cfg.Tracing.ServiceName,
		ServiceVersion: cfg.Tracing.ServiceVersion,
	}
	if !opts.EnableTracer {
		// 短命进程强制 noop，避免冷启动时拨 OTLP 拖慢 migrate
		traceCfg.Endpoint = ""
	}
	tp, tpShutdown, err := tracepkg.NewTracerProvider(ctx, traceCfg, cfg.Logger.AppEnv)
	if err != nil {
		rt.shutdownPartial(ctx)
		return nil, err
	}
	otelapi.SetTracerProvider(tp)
	tracepkg.SetGlobalPropagator()
	rt.TracerProvider = tp
	rt.closers = append(rt.closers, tpShutdown)

	appLogger, logCloser, err := logger.New(logger.Options{
		Level:       cfg.Logger.Level,
		Path:        cfg.Logger.Path,
		AppName:     cfg.Logger.AppName,
		AppEnv:      cfg.Logger.AppEnv,
		ProcessName: opts.ProcessName,
	})
	if err != nil {
		rt.shutdownPartial(ctx)
		return nil, err
	}
	rt.AppLogger = appLogger
	rt.logFileCloser = logCloser
	rt.closers = append(rt.closers, func(context.Context) error { return logCloser.Close() })

	auditLogger, auditCloser, err := logger.NewAuditLogger(logger.Options{
		Level:       cfg.Logger.Level,
		Path:        cfg.Logger.Path,
		AppName:     cfg.Logger.AppName,
		AppEnv:      cfg.Logger.AppEnv,
		ProcessName: opts.ProcessName,
	})
	if err != nil {
		rt.shutdownPartial(ctx)
		return nil, err
	}
	rt.AuditLogger = auditLogger
	rt.closers = append(rt.closers, func(context.Context) error { return auditCloser.Close() })

	return rt, nil
}

// Shutdown 按 LIFO 顺序释放所有 closer，并 join 全部错误。
func (r *Runtime) Shutdown(ctx context.Context) error {
	var errs []error
	for i := len(r.closers) - 1; i >= 0; i-- {
		if err := r.closers[i](ctx); err != nil {
			errs = append(errs, err)
		}
	}
	return stderrors.Join(errs...)
}

func (r *Runtime) shutdownPartial(ctx context.Context) {
	_ = r.Shutdown(ctx)
}
