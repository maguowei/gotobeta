# 日志规范

## 1. 设计取舍

- 标准库 `log/slog`：自带 attr 模型、零依赖、性能可控；不引入 zerolog / zap。
- handler 链分层：`attrsWrap → (sourceHandler 可选) → contextHandler → multi(json+text+sentry)`，每层只做一件事，便于按需替换。
- Sentry 仅作为 ERROR 级别采样上报的 sink；业务日志的真源仍是 app.log + stdout。
- 「记一次」原则：同一错误链路最多记录一次结构化日志（panic 例外，由 recover 中间件记一次最外层）。

## 2. 业务代码用法

### 取 logger

```go
// 优先：从 ctx 取请求级 logger（HTTP 中间件已自动注入）
l := logger.FromContext(ctx)

// 次选：构造函数注入 *slog.Logger，常见于 service / repository
type TodoService struct {
    logger *slog.Logger
}
```

### 字段命名

- 顶层字段（process / appName / appEnv / traceId / spanId / requestId）由 logger / contextHandler 自动注入。
- 业务字段一律用 attr 形式：`slog.String / slog.Int64 / slog.Time / slog.Group`。
- 命名走 camelCase（与 sloglint key-naming-case=camel 对齐）。

### 级别决策

| 级别 | 使用场景 |
|---|---|
| `DebugContext` | 本地排障；prod 环境默认关闭 |
| `InfoContext` | 流程节点（创建成功、状态切换） |
| `WarnContext` | 可恢复异常（重试触发、降级生效） |
| `ErrorContext` | 需告警的失败（Internal 错误、消息持久化失败） |

```go
l.InfoContext(ctx, "todo created",
    slog.Int64("todoId", todo.ID),
    slog.String("title", todo.Title),
)
```

## 3. WithError 正确用法

```go
// 正例：service 边界统一记一次，handler 仅做 HTTP 转换
if err := s.txRunner.RunInTx(ctx, fn); err != nil {
    logger.WithError(ctx, s.logger, "create todo failed", err,
        slog.Int64("todoId", todoID),
    )
    return nil, err
}
```

```go
// 反例：service 与 handler 都记一次（重复噪声）
// service:
s.logger.ErrorContext(ctx, "create todo failed", slog.Any("error", err))
return nil, err
// handler:
appLogger.ErrorContext(c.Request.Context(), "create todo failed", slog.Any("error", err))
httpresponse.Error(c, err)
```

DomainError 自动展开 `errKind` / `errCode` / `errCause`；普通 error 落入 `error` 字段。

## 4. 敏感字段处理

| 类型 | 工具 |
|---|---|
| 手机号 | `sensitive.RedactPhone` |
| 邮箱 | `sensitive.RedactEmail` |
| 身份证 | `sensitive.RedactIDCard` |
| Token / Secret | `sensitive.RedactToken` |
| `Bearer xxx` 头 | `sensitive.RedactBearer` |

Audit 中间件批量脱敏由 `audit.mask_sensitive_fields` 配置项控制。

## 5. 性能与采样

- `AddSource` 默认关闭：开启后 `runtime.CallersFrames` 每条日志多 ~500ns；prod 仅在排障时短暂打开。
- 双写 file + stdout：单条日志 ~2× 写开销，但便于 sidecar 采集 + 容器日志查看；不引入 lumberjack 滚动，由 fluentbit / vector 等外部组件接管。

## 6. 故障排查

| 现象 | 排查 |
|---|---|
| 日志没 traceId | 检查中间件链是否包含 `TraceContext`；上游是否带了 `traceparent` 头 |
| audit.log 空 | 检查 `audit.enabled` 配置；中间件链是否注册了 `Audit` |
| Sentry 无消息 | 检查 `sentry.enabled=true` 且 DSN 非空；level 至少 ERROR |
| 字段被压到 `attrs` | 这是 `attrsWrap` 设计：以 `With` 写入的留顶层，每次调用传入的进 attrs |
