---
paths:
  - "cmd/**"
  - "internal/pkg/logger/**"
  - "internal/pkg/sensitive/**"
  - "internal/app/bootstrap/**"
  - "internal/pkg/httpx/middleware/**"
  - "internal/modules/**"
---

# 日志规范

## 库与初始化

- 必须 `log/slog`；禁止 `fmt.Println` / `log.Printf` 输出业务日志。
- 所有进程入口必须走 `internal/app/bootstrap.Init`；禁止用 `slog.Default()` 承载业务日志。
- 业务代码取 logger：优先 `logger.FromContext(ctx)`，其次构造函数注入。

## context 强制

- 所有日志写入必须使用 `*Context` 系列方法：`InfoContext` / `WarnContext` / `ErrorContext` / `DebugContext`。
- 禁止不带 ctx 的 `Info` / `Warn` / `Error` / `Debug`（无法注入 traceId / requestId）。
- 仅允许在 bootstrap / 单元测试中显式传 `context.Background()` 或测试专用 context。

## 字段与级别

- 级别：`DEBUG`（本地）/ `INFO`（流程节点）/ `WARN`（可恢复异常）/ `ERROR`（需告警）。
- 业务字段一律 attr 形式传入，禁止字符串拼接。
- 错误日志必须用 `logger.WithError(ctx, l, msg, err, attrs...)`。
- 同一错误链路上日志记录不超过一次（panic 除外）。
- 进程标识 `process` 字段由 bootstrap 自动注入（`server` / `worker` / `migrate` / `datainit`）。

## 敏感字段

- phone / email / id_card / token / password 必须经 `sensitive.Redact*` 脱敏。
- audit 中间件敏感字段通过 `audit.mask_sensitive_fields` 配置项统一管理。

## 工具拦截

- golangci-lint 已启用 sloglint `context: "all"`、`static-msg: true`、`attr-only: true`。
- CR 看到 `logger.Info(...)` 不带 ctx 一律打回。
