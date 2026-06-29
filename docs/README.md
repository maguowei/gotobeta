# gotobeta 文档索引

本目录记录生成项目的工程约束、运行手册和需要随代码一起维护的设计事实。代码行为、配置、部署或观测能力变化时，先更新对应文档，再运行验证命令。

## 项目信息

| 项 | 值 |
|---|---|
| 服务名 | `gotobeta` |
| Go module | `github.com/maguowei/gotobeta` |
| 默认质量门禁 | `make verify` |
| 冒烟验证 | `make smoke` |
| Agent 入口 | `AGENTS.md` -> `CLAUDE.md` 软链、`CLAUDE.md`、`.claude/rules/*.md` |

## 导航

| 文档 | 何时更新 |
|---|---|
| [events](events/README.md) | 新增领域事件、调整 outbox/inbox、Kafka topic、worker 消费或幂等策略 |
| [observability/logging](observability/logging.md) | 修改日志字段、脱敏、审计日志、Sentry sink 或 slog handler 链 |
| [observability/error-handling](observability/error-handling.md) | 调整 `internal/pkg/apperr`、HTTP 错误映射或 panic recovery |
| [observability/error-logging](observability/error-logging.md) | 调整错误日志归属层、告警级别或 Sentry 上报策略 |
| [observability/metrics](observability/metrics.md) | 新增指标、修改 label、调整 `/metrics` 暴露方式 |
| [observability/tracing](observability/tracing.md) | 调整 TraceContext、OTLP 配置、Kafka header 传播或手动 span |
| [releases](releases/README.md) | 每次发布前补充变更、验证、升级注意和回滚方案 |
| [tech](tech/README.md) | 新增关键设计、ADR、运行时生命周期或容量/性能约束 |
| [../configs](../configs/README.md) | 新增配置项、环境变量覆盖或默认值 |

## 文档维护规则

- API 行为变化必须同步 `api/openapi.yaml`、相关 HTTP 测试和必要的技术文档。
- 运行时资产变化必须同步 Makefile、Dockerfile、Compose/K8s 文档和 [releases](releases/README.md) 的验证记录。
- 可观测性变化必须同步日志、错误、指标或 tracing 文档，避免只改代码不改排障入口。
- 不在文档中写入真实 token、真实 DSN、真实生产 registry、真实 Kafka topic 或内部域名；只保留可替换示例。
- 缺少本地工具时，记录缺失项并运行可用的最强子集，不要把未验证结果写成已通过。

## 默认验证

```bash
env GOCACHE=/tmp/go-build-cache GOMODCACHE=/tmp/go-mod-cache make verify
```

可选补充：

```bash
make smoke
make lefthook-run
make lint-secrets
docker compose config
kubectl apply --dry-run=client -f deployments/kubernetes/
```
