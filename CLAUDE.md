# gotobeta

## 项目概述

基于 Go DDD 分层架构的后端服务，模块路径 `github.com/maguowei/gotobeta`。

## 构建与测试

```bash
make test          # 运行所有测试
make build         # 构建到 bin/server
make lint          # 运行 golangci-lint
make lint-actions  # 校验 GitHub Actions workflow
make lint-openapi  # 校验 api/openapi.yaml
make lint-secrets  # 使用 gitleaks 扫描硬编码密钥
make mod-verify    # 校验模块缓存完整性
make vuln-check    # 扫描 Go 已知漏洞
make smoke         # 运行生成项目冒烟验证
make verify        # generate/tidy/lint/secrets/vuln/coverage/architecture/build 质量门禁
make test-architecture  # 校验 DDD 分层依赖边界
make lefthook-run  # 运行全部 Lefthook pre-commit hooks
make generate      # 重新生成 Ent 代码
make test-integration-compile  # 只编译 integration build tag，不启动 Docker
make test-integration  # 运行集成测试（需要 Docker）
```

## Agent Workflow

1. 开始前先读取本文件；`.claude/rules/project-workflow.md` 是无 `paths` 的默认规则。
2. `.claude/rules/*.md` 中带 `paths` frontmatter 的规则只在处理匹配文件时适用，按本次改动路径加载。
3. `.claude/hooks/post-tool-use-quality.sh` 提供编辑后的轻量格式化、OpenAPI lint 和 Gitleaks 扫描。
4. `runtime-verification.md` 覆盖 Makefile、Dockerfile、Compose、K8s 和 worker 入口变更。
5. 行为变更先写或更新测试；HTTP API 变更先更新 `api/openapi.yaml`。
6. 完成前默认运行 `make verify`。如果环境缺少工具，说明缺失项并运行可用的最强子集。

受限 agent 环境优先显式指定 Go 缓存目录：

```bash
env GOCACHE=/tmp/go-build-cache GOMODCACHE=/tmp/go-mod-cache make verify
```

提交前本地门禁由 Lefthook 管理：

```bash
go tool lefthook validate
go tool lefthook run pre-commit --all-files
go tool gitleaks dir --redact --no-banner .
go tool actionlint .github/workflows/*.yml
go tool govulncheck ./...
```

涉及运行时资产时补充对应 proof：

```bash
docker compose config
kubectl apply --dry-run=client -f deployments/kubernetes/
```

## 架构分层

```
internal/
  modules/<name>/
    domain/          # 实体、值对象、聚合仓储接口（<聚合>Repository）、领域事件
    application/     # 用例编排与事务边界；command/query/result 三包 + 可选 port 包
    infra/           # 仓储实现、外部适配（按技术命名子包）
    adapter/http/    # Handler、Router、Request、Response
  infra/             # 全局基础设施（config、localid、metrics、sentry）
  pkg/               # 跨模块共享工具（apperr、logger、httpx、idgen 端口）
  app/server/        # HTTP 服务组合根
  ent/               # Ent ORM schema 与生成代码
```

## 开发规范

- 依赖方向：adapter → application → domain ← infra（模块级基础设施）；`adapter/` 是 driving/入站适配器（协议→用例），`infra/` 是 driven/出站适配器（端口→技术）
- domain 层零外部依赖（不引入 gin、ent、viper、slog 等）
- domain 层按聚合分包（domain/todo/、domain/user/），包边界 = 聚合边界；类型去冗余前缀（todo.Todo、todo.Repository、todo.ErrNotFound）
- 仓储接口定义在各聚合包内（一聚合一接口，命名 Repository），实现在 infra 层
- 应用层出入参用 CQRS 命名：<动词><名词>Command / <动词><名词>Query / <名词>Result；查询只读、不发布领域事件
- application/adapter 层 import 别名由 .golangci.yml 的 importas 统一（todocmd、todoreq 等）；domain 聚合包名自解释，无需强制别名
- 应用层通过接口依赖基础设施，不直接 import infra 包
- 跨模块禁止直接 import：模块之间只通过 internal/pkg 共享契约或领域事件协作
- 第三方 SDK 经唯一归口封装使用（sarama→infra/eventbus、go-redis→infra/cache、otel SDK→pkg/trace、sentry-go→infra/sentry+pkg/logger、jwt→pkg/auth），业务代码不直接 import SDK
- cmd/*/main 保持极薄：只做信号处理、bootstrap 与 internal/app 组合根调用
- HTTP request/response 契约只能从 command/query/result 映射，不得 import domain
- 写操作尽量幂等
- 配置只在组合根读取，通过构造函数注入各层；共享内核（internal/pkg，bootstrap 除外）不得 import internal/infra
- 数据库 schema 不使用外键，跨聚合一致性由应用层和唯一索引约束。
- 以上分层与归口边界由 `make test-architecture` 自动校验。

## 提交规范

使用中文 Conventional Commits：`feat: 描述`、`fix: 描述`、`docs: 描述` 等。
