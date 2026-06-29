---
paths:
  - "internal/modules/**/application/**"
---

# Application Layer

- 应用层负责用例编排、事务边界和跨领域协调，不直接处理 HTTP 请求细节。
- 用例出入参采用轻量 CQRS 命名：写用例入参 `<动词><名词>Command`（`application/command` 包）、读用例入参 `<动词><名词>Query`（`application/query` 包）、出参 `<名词>Result`（`application/result` 包）。
- 命令可改状态、返回最小必要数据；查询只读、不得触发写副作用或发布领域事件。
- 应用服务方法按读写拆分文件：`<name>_commands.go` 与 `<name>_queries.go`；模块自有技术性端口（PasswordHasher、EmailSender 等）定义在 `application/port` 包，仓储接口归 domain。
- 依赖通过构造函数显式注入，禁止使用隐式全局状态。
- 应用服务可以依赖领域接口和平台抽象，不反向依赖 handler、router 或 Ent schema。
- 写操作应尽量设计为幂等，外部调用失败要返回可诊断错误。
- 新增用例时优先先写应用层单元测试，再补实现。
- `make test-architecture` 会自动校验应用层不得 import infra、adapter、Gin、Ent（含生成代码 `internal/ent`）、Viper 或 net/http；`application/query` 包不得 import `internal/pkg/event`（查询不发事件），service 的 `*_queries.go` 同样不得发布事件（文件级靠 code review 兜底）。
