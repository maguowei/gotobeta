# 技术设计文档

存放本服务的技术设计文档，包括：

- 核心业务流程的时序图与状态机
- 关键数据模型设计
- 性能与容量规划
- 技术选型决策记录（ADR）

## 当前架构基线

- 代码按 DDD 垂直模块组织：`domain`、`application`、`infra`、`adapter/http`。
- 依赖方向是 `adapter -> application -> domain <- infra`。
- `internal/infra` 放全局基础设施实现，`internal/pkg` 放跨模块共享工具。
- 配置只在组合根读取，再通过构造函数注入到各层。
- 数据库 schema 不使用外键；跨聚合一致性由应用层、事务边界和唯一索引维护。

## 运行时约束

- HTTP 进程在组合根中监听 `SIGINT`/`SIGTERM`，收到信号后使用带超时的 `server.Shutdown` 优雅关闭。
- 后台 worker 进程同样由入口进程创建可取消 context，并把 context 传入 worker server。
- 所有入口进程都通过 `bootstrap.Init` 初始化 config、Sentry、tracing、logger 和 audit 依赖，退出前调用 `Runtime.Shutdown`。
- server、migrate、worker、datainit 等入口的失败路径不能直接绕过 deferred cleanup。

## 质量门禁

默认完整门禁：

```bash
env GOCACHE=/tmp/go-build-cache GOMODCACHE=/tmp/go-mod-cache make verify
make lint-actions
make lint-openapi
make lint-secrets
make vuln-check
```

变更影响运行时资产时补充：

```bash
make smoke
docker compose config
kubectl apply --dry-run=client -f deployments/kubernetes/
```

## ADR 模板

```
# ADR-NNN: 标题

## 背景
- 当前问题：
- 约束：

## 决策
- 选择：
- 不选择：

## 影响
- 正向影响：
- 代价：
- 迁移步骤：
- 回滚方案：

## 验证
- 测试：
- 运行时 proof：
```

## 维护规则

- 架构边界变化必须同步 `internal/architecture/dependency_test.go`。
- 配置、部署、事件或观测变化必须同步对应 docs 和 release note。
- 大型设计文档放在本目录下独立文件，文件名使用日期或 ADR 编号前缀。
