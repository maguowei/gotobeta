# Observability Stack

Local OpenTelemetry Collector, Prometheus, Jaeger, and Grafana stack for gotobeta.

## 启动

```bash
docker compose -f deployments/observability/docker-compose.observability.yml up
```

运行服务并启用 OTLP tracing：

```bash
APP_TRACING_ENDPOINT=localhost:4317 \
APP_TRACING_INSECURE=true \
go run ./cmd/server
```

## 组件

| 组件 | 地址 | 用途 |
|------|------|------|
| Prometheus | http://localhost:9090 | 指标存储与查询 |
| Jaeger | http://localhost:16686 | 分布式追踪可视化 |
| Grafana | http://localhost:3000 | 统一仪表板 |
| OTEL Collector | localhost:4317 (gRPC) / :4318 (HTTP) | 遥测数据收集与路由 |

## 数据流

```
Service → (OTLP gRPC) → OTEL Collector
                          ├─ traces  → Jaeger
                          ├─ metrics → Prometheus ← Grafana
                          └─ logs    → debug (stdout)

Prometheus  ← scrape ← Service :8080/metrics
            ← scrape ← OTEL Collector :8889
```

Grafana 预配置了 Prometheus 和 Jaeger 数据源，以及服务概览仪表板（基于 RED 方法论）。Prometheus 启用了 `exemplar-storage`，支持从高延迟指标直接跳转到对应 trace（需 Exemplar label `traceID`）。

## 预置仪表板

启动后访问 Grafana (http://localhost:3000)，可直接使用预置的「gotobeta - 服务概览」仪表板，包含：

- **HTTP 请求速率 (QPS)**：按 method/path 分组的请求吞吐量
- **HTTP 错误率**：4xx/5xx 错误占比
- **HTTP 请求耗时百分位**：P50/P95/P99 延迟分布
- **外部调用耗时**：对外部服务的 P95 调用延迟
- **事件总线处理速率**：inbox/outbox 事件处理吞吐量
- **事件总线处理耗时**：事件处理 P95 延迟
- **Go Runtime**：goroutines、heap 内存、GC 耗时

## 登录

Grafana 默认本地登录：`admin` / `${GRAFANA_ADMIN_PASSWORD:-codego-local-admin}`。共享环境请提前设置 `GRAFANA_ADMIN_PASSWORD` 环境变量。

## Metrics → Traces 关联

本项目的 Prometheus 指标中间件和 Observe 辅助函数已内置 Exemplar 支持：当请求带有有效的 OpenTelemetry SpanContext 时，histogram 观测会自动写入 `traceID` Exemplar。在 Grafana 的 Explore 面板中，点击 Prometheus histogram 数据点的 Exemplar 标记即可跳转到 Jaeger 查看完整 trace。

## 日志收集

当前日志通过 `slog` 输出 JSON 到 stdout，每条日志自动嵌入 `traceId` / `spanId`。K8s 部署场景下可通过以下方式接入集中式日志系统：

- **Loki + Promtail**：以 DaemonSet 或 sidecar 方式采集容器 stdout 日志，按 `traceId` label 与 Jaeger 关联。
- **ELK (Elasticsearch + Logstash + Kibana)**：Fluentd/Filebeat sidecar 采集日志，按 `traceId` 字段在 Kibana 中关联 trace。

OTLP logs exporter 暂不引入——Go OTel logs API 虽已稳定，但日志收集工具链仍以 sidecar 为主流。

## 为何使用 Prometheus client 而非 OTel Metrics

Tracing 使用 OTel SDK，Metrics 使用 Prometheus client_golang。当前不迁移到 OTel Metrics，理由：

1. Prometheus client_golang 成熟、稳定，Grafana/Alertmanager 生态原生支持
2. OTEL Collector 已通过 Prometheus exporter 桥接，不存在数据孤岛
3. `prometheus/bridge/otelprometheus` 可按需启用，无需重写 instrumentation
4. OTel Go Metrics SDK 虽已稳定，但告警和可视化工具链仍以 Prometheus 为主

未来迁移路径：当团队需要 OTLP-native metrics push 或跨语言统一 SDK 时，可通过 bridge 渐进迁移。

## 注意

此 stack 仅用于本地开发和 smoke 验证。生产部署应使用受管凭据、持久存储、TLS for OTLP 和集群原生服务发现。
