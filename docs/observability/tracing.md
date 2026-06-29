# Trace 规范

## 1. 架构总览

```
HTTP client ─[traceparent]─▶ HTTP server (TraceContext middleware)
                                  │
                                  ▼
                              Service (logger.WithError 自动带 traceId)
                                  │ outbox (CloudEvent.TraceParent 注入)
                                  ▼
                              Outbox publisher ─[Kafka headers]─▶ Inbox consumer
                                                                       │
                                                                       ▼
                                                                  ce.ExtractContext
                                                                  + new "inbox.consume" span
```

## 2. TracerProvider 三种状态

| 状态 | 触发条件 | 用途 |
|---|---|---|
| noop | `tracing.endpoint=""`（默认）或入口显式禁用 tracer | 零开销，本地开发和短命进程免依赖 |
| OTLP/gRPC | `tracing.endpoint` 非空 | 接 jaeger / tempo / collector |
| 自定义 | 修改 `internal/pkg/trace/provider.go` | 临时换 stdout exporter 调试 |

短命进程（migrate / datainit）通过 `bootstrap.Options.EnableTracer=false` 强制 noop。

`tracing.endpoint` 非空即视为启用。生产环境如果使用明文 OTLP，需要确认内网拓扑；默认不建议把 `tracing.insecure=true` 带到公网链路。

## 3. 手动开 span 最小示例

```go
import (
    "go.opentelemetry.io/otel"
    traceapi "go.opentelemetry.io/otel/trace"
)

func (s *Service) DoJob(ctx context.Context, id string) error {
    ctx, span := otel.Tracer("my-service").Start(ctx, "do-job",
        traceapi.WithAttributes(attribute.String("job.id", id)),
    )
    defer span.End()

    if err := s.run(ctx, id); err != nil {
        span.RecordError(err)
        span.SetStatus(codes.Error, err.Error())
        return err
    }
    return nil
}
```

## 4. 跨进程链路

### HTTP → HTTP

server 的 `TraceContext` 中间件用 `otelhttp.NewHandler` 包裹，自动从 `traceparent` 头继承上游 span。client 必须用 `otelhttp.NewTransport` 包装：

```go
client := &http.Client{Transport: otelhttp.NewTransport(http.DefaultTransport)}
```

### HTTP → Outbox → Worker

`NewCloudEvent(ctx, source, e)` 自动从 ctx 注入 `traceparent` / `tracestate` 字段。Outbox publisher 写 Kafka 时把这两个值平铺到 message headers；inbox consumer `ce.ExtractContext(ctx)` 还原后开 `inbox.consume` span。

### Worker → 外部 HTTP

worker 内部继续用 `otelhttp.NewTransport`；trace 链会自动拼接。

## 5. 采样策略

| Sampler | 适用 |
|---|---|
| `always` | 联调阶段；后端只跑少量请求 |
| `never` | 出问题先关采样，但仍透传 traceparent |
| `parent` (默认) | 跟随上游决策；自身根 span 全采 |
| `ratio` | 大流量按 `sample_ratio` 抽样（0–1） |

## 6. 本地开发

`docker-compose.yml` 片段（参考）：

```yaml
otel-collector:
  image: otel/opentelemetry-collector-contrib:latest
  ports: ["4317:4317"]
jaeger:
  image: jaegertracing/all-in-one:latest
  ports: ["16686:16686"]  # UI
```

启动后设置：

```bash
APP_TRACING_ENDPOINT=localhost:4317 APP_TRACING_INSECURE=true \
  go run ./cmd/server
```

## 7. 日志联动

`contextHandler` 会从 `SpanContextFromContext(ctx)` 自动注入 `traceId` / `spanId` 到每条日志。所以：

- 拿日志的 traceId 即可在 jaeger UI 反查链路
- 拿 jaeger 的 traceId 即可在日志后端 `traceId:xxx` 拉到全部 log

## 8. 变更检查

- HTTP 中间件顺序必须保持 TraceContext 在业务 handler 之前，且在 recovery 之后尽早执行。
- 新增外部 HTTP client 时使用 `otelhttp.NewTransport` 包装 transport。
- 新增 Kafka/worker 链路时传递 `traceparent` / `tracestate`，消费侧先恢复上下文再开处理 span。
- 修改 tracing 配置时同步 `configs/config.*.yaml`、`internal/infra/config` 测试和本文件。
