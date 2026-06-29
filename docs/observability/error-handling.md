# 错误处理规范

## 分层职责

| 层 | 职责 |
|---|---|
| domain | 返回领域语义错误，不依赖 HTTP、Gin、slog 或基础设施包 |
| application | 编排用例，把基础设施错误包装成可理解的领域/应用错误 |
| infrastructure | 把数据库、Kafka、外部 API 等错误翻译成上层可处理的 error |
| adapter/http | 做协议边界校验，根据 `errors.Kind` 映射 HTTP 状态和响应体 |
| middleware | 兜底 recovery、审计日志、trace/request 上下文注入 |

## 错误类型

- 领域错误使用 `internal/pkg/apperr` 中的统一类型，携带 `Kind`。
- 支持的 `Kind`：`InvalidParam`、`NotFound`、`Conflict`、`Unauthorized`、`Forbidden`、`Internal`。
- 应用层不直接返回 HTTP 状态码；adapter 层根据 `errors.Kind` 映射到 HTTP 状态。
- 技术错误保留在 `Cause` 中供日志排查，不直接暴露给调用方。

## HTTP 映射

| Kind | HTTP | 业务码 | 说明 |
|---|---:|---:|---|
| `InvalidParam` | 400 | `40001` | 请求参数格式错误 |
| `Unauthorized` | 401 | `40101` | 未认证或认证失效 |
| `Forbidden` | 403 | `40301` | 已认证但无权限 |
| `NotFound` | 404 | `40401` | 资源不存在 |
| `Conflict` | 422 | `42201` | 状态冲突、唯一性冲突或业务规则不满足 |
| `Internal` | 500 | `50001` | 未预期系统错误 |

统一 API 响应为 `{code,message,data}`。`message` 面向调用方，不能包含 SQL、DSN、token、内部域名或堆栈。

## Panic 与 Sentry

- 所有 panic 由 `recovery` 中间件捕获，记录 ERROR 日志并上报 Sentry，返回 500。
- Sentry `request_path` 使用 Gin 路由模板，未匹配路由固定为 `unknown`，禁止使用原始 URL。
- 不要在业务代码中用 panic 表达可预期错误。

## 测试要求

- 新增 `Kind` 或错误码时，补 HTTP response 映射测试。
- handler 测试只断言协议响应；application 测试断言领域语义和错误包装。
- repository/infra 测试应覆盖基础设施错误翻译，不泄露底层错误给 HTTP 响应。

> 错误日志记录方式与分层约束见 [error-logging.md](./error-logging.md)。
