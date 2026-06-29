# 指标规范

## 基本规则

- 使用 Prometheus，指标命名遵循 `<namespace>_<subsystem>_<name>_<unit>` 格式。
- 指标在 `internal/infra/metrics/metrics.go` 中集中注册，避免在 handler 层直接操作 counter/histogram。
- `Init` 会先 unregister 再注册，避免测试重复注册 panic。
- `/metrics` 路径仅在 `cfg.Metrics.Enabled=true` 时注册，路径来自 `metrics.path`。

## Label 约束

| 指标域 | 允许 label | 禁止 |
|---|---|---|
| HTTP | `method`、`path`、`status` | 原始 URL、query、用户 ID、trace ID |
| 外部调用 | `target`、`status` | 具体错误文本、动态 host、请求参数 |
| 事件总线 | `component`、`event_type`、`status` | event ID、aggregate ID、payload 字段 |

HTTP 请求指标由 `internal/pkg/httpx/middleware/metrics.go` 自动采集。`path` 使用 Gin 路由模板，未匹配路由固定为 `unknown`，禁止使用原始 URL 作为 label。

外部调用统一通过 `ObserveExternalCall` 记录，`status` label 限定为 `2xx`、`4xx`、`5xx`、`timeout`、`network_error`。

事件总线通过 `ObserveEventBus` 记录 inbox/outbox 处理次数和耗时，`status` 使用 `published`、`processed`、`ignored`、`retry`、`dead`、`error` 等有限值。

## 暴露方式

- HTTP server 通过 `server.host:server.port` 暴露 `metrics.path`。
- 启用事件 worker 时，worker 进程会在自己的 `server.host:server.port` 暴露同一个 `metrics.path`。
- Docker Compose 内部通过 `server:8080`、`worker:8080` 采集；Kubernetes 容器声明 `containerPort: 8080` 便于采集系统发现。

## 新增指标清单

1. 先确认已有 HTTP、external call、eventbus 指标是否能表达需求。
2. 新增指标必须在 `internal/infra/metrics` 集中注册，并写清楚 namespace、subsystem、name、unit。
3. label 集合必须是有限枚举，不能包含动态 ID、错误文本或高基数字段。
4. 补充单元测试或 handler/worker 级别验证，确保重复初始化不会 panic。
5. 同步本文件和 release note。
