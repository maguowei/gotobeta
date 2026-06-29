# 错误日志规范

## 1. 「记一次」原则

一个错误从产生到响应返回，整条链路上结构化错误日志只记一次。多记会污染 SLO 告警基线，少记会丢排障线索。

```
[infra] 抛出 → [service] 包装为 DomainError → [service 边界 WithError 记一次] → [handler] httpresponse.Error
```

## 2. 分层记录决策

| 层 | 写日志？ | 何时 |
|---|---|---|
| domain | 不写 | 领域不依赖 slog；返回 error 即可 |
| infra (repository / outbox) | 不写 | 抛 error 让上层决定；除非 log 内容是 infra 独有上下文（如 SQL 摘要） |
| application service | 写 | 边界处统一 `logger.WithError`，把跨多个 infra 调用的上下文聚合 |
| interface (handler) | 不写 | 转 HTTP 响应；service 已经记过 |
| panic 兜底中间件 | 写 | recover 后 ERROR 一次，确保不丢栈 |

## 3. DomainError.LogAttrs 展开

```go
e := errors.Internal("save failed", rootErr).WithCode("E_DB_001")
logger.WithError(ctx, l, "save todo failed", e, slog.Int64("todoId", id))
```

输出 JSON 字段：

| key | 来源 |
|---|---|
| `errKind` | `Kind.String()` 例如 `Internal` |
| `errMsg` | DomainError.Message |
| `errCode` | DomainError.Code（可选） |
| `errCause` | Cause.Error()（可选） |

## 4. Error vs Warn vs 不记

| 情境 | 级别 |
|---|---|
| Internal 错误（DB 写入失败、RPC 超时） | `Error` |
| 业务规则不通过（NotFound / Conflict） | 通常**不记**，HTTP 4xx 由 audit 中间件已经覆盖 |
| 重试触发、降级路径生效 | `Warn` |
| 单次失败但即将重试 | `Warn`（带 `attempt` 字段） |

## 5. 与 Sentry 的关系

- ERROR 级日志会被 `sentryHandler` 二次发送到 Sentry（仅当 `sentry.enabled=true`）
- panic 走 sentry middleware 的 `recover` 路径，不依赖日志路径
- 不要为了「让 Sentry 记一下」而硬升日志级别 — 用 `sentry.CaptureException` 显式上报更稳

> 错误处理与 errors 包用法详见 [error-handling.md](./error-handling.md)。
