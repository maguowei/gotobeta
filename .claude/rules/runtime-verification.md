---
paths:
  - "Makefile"
  - "Dockerfile"
  - "docker-compose.yml"
  - "deployments/kubernetes/**"
  - "cmd/server/**"
  - "cmd/migrate/**"
  - "cmd/worker/**"
  - ".claude/settings.json"
  - "AGENTS.md"
  - "CLAUDE.md"
---

# 运行时验证

- 生成项目默认以 `make verify` 作为完整质量门禁；它负责 `generate`、`tidy`、`lint`、`lint-secrets`、`test`、`test-architecture` 和 `build`。
- `make smoke` 是生成项目冒烟路径，至少要覆盖代码生成、依赖整理、测试、分层校验和二进制构建。
- 修改 Makefile、Dockerfile、Compose、K8s 或 worker 入口时，除单元测试外还要证明运行时资产可解析或可构建。
- Docker Compose 变更至少运行 `docker compose config`；缺少 Docker 时记录真实缺失并继续跑 `make verify` 或可用子集。
- Kubernetes 变更至少运行 `kubectl apply --dry-run=client -f deployments/kubernetes/`；缺少 kubectl 时记录缺失并检查 YAML/模板内容。
- 异步能力启用时，镜像和部署资产必须包含 `server`、`migrate`、`worker` 多二进制；Compose/K8s worker 应运行 `/app/worker`，不要退回 `go run ./cmd/worker`。
- 受限缓存目录下优先使用 `env GOCACHE=/tmp/go-build-cache GOMODCACHE=/tmp/go-mod-cache make verify`。
- 本地 Git hooks 由 Lefthook 管理，提交前使用 `make lefthook-validate` 与 `make lefthook-run`。
- 密钥扫描使用 `make lint-secrets`，OpenAPI 契约检查使用 `make lint-openapi`；这些目标默认通过 `go tool` 运行已固定版本的工具。
