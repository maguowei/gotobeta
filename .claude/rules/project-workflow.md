# 项目工作流

- 本文件是生成项目的默认 Claude Code 上下文规则；开始任务前先读 `CLAUDE.md`，再读取与本次改动路径匹配的 `.claude/rules/*.md`。
- 先判断改动影响面：API、domain、application、adapter/http、infra、配置、观测、部署、测试或文档，然后选择最小可证明的验证集。
- HTTP API 变更先更新 `api/openapi.yaml` 和请求/响应契约，再实现 DTO、Handler、Router 和测试。
- 行为变更先写或更新测试，保持单元测试优先；跨组件流程再补集成或 smoke 证明。
- 依赖通过构造函数显式注入；不要新增隐式全局状态、包级可变单例或绕过组合根的初始化。
- 配置只在组合根读取，通过 typed config 注入到依赖方；系统边界做输入校验，内部代码信任已校验的数据。
- 完成前默认运行 `make verify`。如果环境缺少工具，说明缺失项并运行可用的最强子集，不要把未验证结果说成已通过。
- 受限 agent 环境优先使用 `/tmp` 缓存：`GOCACHE=/tmp/go-build-cache`、`GOMODCACHE=/tmp/go-mod-cache`。
- 本地 Git hooks 使用 Lefthook；OpenAPI lint 使用 vacuum，密钥扫描使用 Gitleaks；这些工具默认通过 Go tool directive 固定版本。
- 提交信息使用中文 Conventional Commits，例如 `feat: 增加订单查询接口`。
