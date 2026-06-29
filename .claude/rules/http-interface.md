---
paths:
  - "internal/app/server/**"
  - "internal/modules/**/adapter/http/**"
  - "internal/modules/system/adapter/http/**"
  - "internal/pkg/httpx/**"
---

# HTTP Interface

- adapter/http 层只负责协议适配：绑定参数、调用应用服务、转换响应。
- 请求模型放 `adapter/http/request` 包（`<动词><名词>Request`，经 `ToCommand()`/`ToQuery()` 映射到应用层），响应模型放 `adapter/http/response` 包（`<名词>Response`）。
- handler 端对应用服务的依赖接口命名 `<能力>UseCase`，与应用服务实现 `<能力>Service` 区分。
- 请求模型必须在边界完成校验，应用层默认信任已校验输入。
- 统一使用 `internal/pkg/httpx/response` 输出 JSON，避免 handler 自己拼响应结构。
- 路由注册集中在模块自己的 router 中，再由 bootstrap 装配。
- HTTP 组合根必须使用 `signal.NotifyContext` 和带超时的 `server.Shutdown` 处理 SIGINT/SIGTERM。
- 不要在 handler 中直接访问数据库、读取配置文件或创建基础设施客户端。
- `make test-architecture` 会自动校验模块 adapter 层不得 import infra、Ent 或 database/sql；`request`/`response` 包不得 import domain（契约只能从 command/query/result 映射），handler 包不得 import `domain/entity`；组合根装配在 `internal/app/**`，不参与该断言。
