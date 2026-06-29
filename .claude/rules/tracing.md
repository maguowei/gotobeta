---
paths:
  - "configs/**"
  - "internal/pkg/trace/**"
  - "internal/app/bootstrap/**"
  - "internal/pkg/httpx/middleware/**"
  - "internal/infra/eventbus/**"
  - "internal/app/server/**"
  - "internal/app/worker/**"
---

# Trace 规范

## propagator

- 全局只在 `bootstrap.Init` 中通过 `trace.SetGlobalPropagator()` 设置一次（W3C TraceContext + Baggage 复合）。
- 业务代码取 propagator：`otel.GetTextMapPropagator()`，禁止重新构造。

## 跨进程传播

- HTTP server：`TraceContext` 中间件必须在中间件链顶端（仅次于 Recovery）。
- HTTP client：必须用 `otelhttp.NewTransport` 包装；禁止裸 `http.Client`。
- Kafka producer：消息 headers 必须注入 `traceparent`；CloudEvent struct 上的 `TraceParent` 字段在 `NewCloudEvent(ctx, ...)` 中自动填充。
- Kafka consumer：fetch 后必须 `ce.ExtractContext(ctx)` 还原后再交给 handler，并 `tracer.Start(ctx, "inbox.consume")` 起新 span。
- 短命进程（migrate / datainit）：`bootstrap.Options.EnableTracer=false`，强制 noop 避免 OTLP 拨号拖慢冷启动。

## 配置

- prod 环境 `tracing.endpoint` 非空时，`tracing.insecure` 必须为 false。
- `tracing.endpoint` 非空即视为启用（唯一真值源，没有冗余的 enabled 字段）。
- `sampler=ratio` 时 `sample_ratio` 必须在 (0, 1]。
- 不引入 `OTEL_*` 环境变量；统一走 `APP_TRACING_*` 前缀。

## 日志联动

- `contextHandler` 自动从 `SpanContextFromContext(ctx)` 注入 `traceId` / `spanId`。
- 链路调试：trace 后端按 traceId 反查 `logs.app` 索引即可拉到全部日志。
