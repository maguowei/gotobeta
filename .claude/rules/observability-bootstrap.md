---
paths:
  - "cmd/**"
  - "internal/app/bootstrap/**"
  - "internal/pkg/logger/**"
  - "internal/pkg/trace/**"
  - "internal/infra/sentry/**"
  - "internal/app/server/**"
  - "internal/app/worker/**"
---

# Bootstrap 规范

## 入口约束

- 所有 `cmd/<x>/main.go` 必须调用 `bootstrap.Init` 并 `defer rt.Shutdown(context.Background())`。
- 禁止在 main 中直接 `logger.New` / `sentry.Init` / `trace.NewTracerProvider`：bootstrap 已经串联好顺序与回滚。
- 任何初始化失败必须先 `rt.Shutdown` 再退出，不得直接 `log.Fatal` / `os.Exit` 绕过日志刷盘。

## Options 约束

- `ProcessName` 必须显式声明，取值集合：`server` / `worker` / `migrate` / `datainit`。
- 进程名会写入每条日志的 `process` 顶层字段，便于按进程类型过滤。
- 长命进程（server / worker）：`EnableTracer: true`。
- 短命进程（migrate / datainit）：`EnableTracer: false`，强制 noop。

## 关闭语义

- `Runtime.Shutdown` 按 LIFO 顺序调用所有 closer，并 `errors.Join` 收集所有错误。
- 部分初始化失败时，`Init` 内部已对成功的 closer 做回滚，调用方收到 error 即可不必再调 Shutdown（但依然推荐 defer）。

## 测试

- 新增 cmd 必须在 `internal/codego/testdata` 的 golden 与 smoke 测试中覆盖编译。
- 修改 bootstrap 内部顺序（config → sentry → trace → logger → audit）必须同步更新 `bootstrap_test.go` 的 LIFO/Join 断言。
