---
paths:
  - "**/*_test.go"
  - "internal/integration/**"
  - "internal/testutil/**"
---

# 测试分层规范

## 分层边界

| 层级 | 范围 | 运行方式 |
|------|------|----------|
| 单元测试 | 单个函数/方法，无外部依赖 | `go test ./...` |
| 集成测试 | 需要真实数据库/容器 | `go test -tags integration ./internal/integration/...` |
| E2E 测试 | 完整服务链路 | 独立环境手动触发 |

## Build Tag 用法

- 集成测试文件顶部必须声明 `//go:build integration`
- 默认 `make test` 不运行集成测试，避免 CI 依赖 Docker

## Testcontainers 最佳实践

- 使用 `testutil.StartMySQL(ctx, t)` 启动容器，`t.Cleanup` 自动清理
- 每个 suite 独立容器，避免测试间状态污染
- 容器启动失败直接 `t.Fatal`，不要 skip
