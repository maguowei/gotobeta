# gotobeta

由 `codego` 生成的 Go 业务脚手架，基于 DDD 分层架构。

## 目录概览

```text
cmd/
  server/   # HTTP 服务入口
  migrate/  # 数据库迁移入口
internal/
  infra/     # 配置、ID 生成、指标、Sentry 等基础设施
  app/       # 进程组合根（HTTP server、worker server）
  modules/   # 业务模块（按限界上下文垂直切片）
  pkg/       # 跨模块共享工具（错误、日志、HTTP 中间件）
  ent/       # Ent ORM schema 与生成代码
```

## 快速开始

```bash
make run          # 启动 HTTP 服务
make test         # 运行所有测试
make lint         # Go 与 OpenAPI 检查
make lint-secrets # Gitleaks 密钥扫描
make build        # 构建到 bin/
make smoke        # 生成代码、整理依赖、测试、架构校验和构建
make verify       # 完整质量门禁：生成、整理、lint、密钥、模块、漏洞、覆盖率、架构和构建
make test-architecture  # 校验 DDD 分层依赖边界
```

## AI-Native IM 模块

在脚手架之上实现的第一期 IM 后端（Slack + 微信风格），按限界上下文垂直切片，并为后续 AI 能力预留扩展缝：

| 模块 | 职责 | 关键能力 |
|------|------|----------|
| `modules/workspace` | 工作区（租户）与动态 RBAC | 多工作区组织模型、DB 动态角色/权限 + ACL、平台模板复制 |
| `modules/messaging` | 会话与消息 | 单聊（dm_key 对称去重）/群聊/频道、按会话 seq 时间线、发送/撤回/已读水位、读扩散统一时间线 |
| `modules/realtime` | 实时下行 | WS 网关（ticket 一次性鉴权）、在线状态/typing、push-pull 可靠下行、多端对齐 |
| `modules/media` | 附件 | S3 兼容对象存储预签名上传 + 提交确认 |

预留的 4 个 AI 扩展缝：领域事件经事件总线外发（见 `docs/events/`）、Bot/Agent 作为一等发送者（`senderType`）、消息 content 采用结构化 block、表上预留 metadata 列。

单节点 MVP，关键边界（事件总线、对象存储、缓存）通过端口接口预留演进路径；集成测试（seq 并发分配、RBAC 解析、单聊去重）见 `internal/integration/`（`make test-integration`，需 Docker）。完整设计见 `docs/superpowers/specs/`。

## Demo API

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/healthz` | 存活探针 |
| GET | `/readyz` | 就绪探针（检查 DB 等依赖） |
| GET | `/api/v1/todos` | 列出待办 |
| GET | `/api/v1/todos/:id` | 查询待办 |
| POST | `/api/v1/todos` | 创建待办 |
| POST | `/api/v1/todos/:id/complete` | 完成待办 |
| DELETE | `/api/v1/todos/:id` | 删除待办 |

## 常用命令

```bash
make run            # 启动服务
make migrate        # 执行数据库迁移
make test           # 运行测试
make lint           # Go 与 OpenAPI 检查
make lint-actions   # GitHub Actions workflow 检查
make lint-openapi   # OpenAPI 契约检查
make lint-secrets   # Gitleaks 密钥扫描
make mod-verify     # Go 模块缓存校验
make vuln-check     # Go 已知漏洞扫描
make smoke          # 冒烟验证
make verify         # 完整质量门禁
make test-architecture  # 架构边界测试
make build          # 构建
make generate       # 重新生成 Ent 代码
make docker-build   # 构建 Docker 镜像
make test-integration-compile  # 编译集成测试
make test-integration          # 运行集成测试（需要 Docker）
```

## 配置

配置默认读取 `configs/config.local.yaml`。切换环境时设置 `APP_ENV=dev|test|prod`，或用 `APP_CONFIG_DIR` 指向其他配置目录。

敏感配置（DSN、Token）通过环境变量注入，统一使用 `APP_` 前缀，例如 `APP_DATABASE_DSN`。
