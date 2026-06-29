---
paths:
  - "internal/modules/**/domain/**"
---

# Domain Layer

- 领域层只表达业务规则，不依赖 Gin、Viper、Ent、数据库、HTTP DTO 或其他基础设施。
- 实体和值对象必须保持可单元测试；校验规则应放在构造函数或领域方法中。
- 聚合根（有不变量/状态机/业务行为的实体）使用私有字段 + getter + Factory 构造函数；状态修改只能通过业务方法，外部不得绕过。infra 层通过 `UnmarshalFromDB` 重建实体，跳过业务校验。
- 支撑性持久化模型（无业务规则、生命周期完全由基础设施驱动的技术记录，如 OAuth state、第三方身份资料快照）可保持公开字段；放在 domain 包内时必须在 `doc.go` 显式声明"非聚合根"，避免与聚合根混淆。判据：没有需要在领域内强制的不变量或状态机时，就不要升级成聚合根，否则只增样板而无收益。
- domain 层按聚合分包（`domain/todo/`、`domain/user/`），包边界 = 聚合边界；类型去冗余前缀（`todo.Todo`、`todo.Repository`、`todo.ErrNotFound`）。
- 仓储在聚合包内只定义接口（命名 `Repository`），具体实现放在 infra 层；哨兵错误命名 `Err<语义>`（如 `ErrNotFound`），定义在聚合包的 `errors.go` 中。
- 聚合根一聚合一文件，文件名等于聚合名；领域事件用过去时 + Event 后缀命名（如 `todo.CreatedEvent`）。
- 同一模块内聚合包之间禁止互相 import；跨聚合协调在 application 层完成。`make test-architecture` 会机械化检测违规。
- 领域服务优先按统一语言能力命名（`<聚合><能力>Policy` / `Calculator` / `Allocator`），不带 `Service` 后缀；确需后缀时包别名用 `<bc>domainsvc` 与应用服务的 `<bc>svc` 区分。
- 领域错误使用 `internal/pkg/apperr` 中的统一错误类型或哨兵错误，不直接拼 HTTP 状态码。
- 领域层禁止依赖 `log/slog` / `go.opentelemetry.io/otel` / `internal/pkg/logger` / `internal/pkg/trace`；错误用 `errors.New*Error` 构造立即返回，由应用层或边界层统一记日志。
- 新增业务模块时优先放在 `internal/modules/<name>/domain`。
- `make test-architecture` 会自动校验领域层不得 import infra、adapter、application、Gin、Ent（含生成代码 `internal/ent`）、Viper、database/sql、net/http 或 slog。
