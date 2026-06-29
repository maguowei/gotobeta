# IM 后端生产运维 Runbook（阶段 A）

阶段 A 形态：**单实例进程 + 单 MySQL + Redis**，进程内 Hub/EventBus。
本文档汇总上线约束、关键配置、监控信号与常见故障处置。

## 1. 部署约束（单实例）

- 进程内 Hub 与 EventBus 不跨实例广播：**WS 网关与消息发送必须同实例**。
  多副本时 A 实例发的消息无法推给连在 B 实例的客户端。阶段 A 通过单实例规避；
  Kubernetes `Deployment` 保持 `replicas: 1`，`Service` 设 `sessionAffinity: ClientIP`。
- 水平扩展是阶段 B 工作（外置 Hub/广播总线，如 Redis Pub/Sub 或 NATS），阶段 A 不支持。
- 优雅关闭：收到 SIGTERM 后先 `realtime.Shutdown` 广播 WS close 帧并等待连接排空，
  再关闭 HTTP server；Pod `preStop` 预留 sleep 让 LB 摘流。

## 2. 关键配置

| 配置 | 作用 | 备注 |
|---|---|---|
| `im.message_rate_per_minute` / `_burst` | 发消息限流（按用户） | 过载保护 |
| `im.max_ws_connections` / `max_conn_per_user` | WS 连接上限 | 超限握手返回 WS 1013 |
| `im.ws_handshake_rate_per_minute` | 握手限流 | 防连接风暴 |
| `im.ws_pong_wait` / `ws_write_wait` / `ws_read_limit` | WS 心跳与帧限制 | 影响假死连接回收 |
| `im.ws_allowed_origins` | WS Origin 白名单 | 与 `server.cors_allowed_origins` 配合 |
| `server.behind_tls_proxy` / `server.tls.*` | 传输层加密 | prod 二选一，见 config README |
| `redis.enabled` | presence/ticket/JWT 黑名单 | 关闭时这些能力降级为单机/不可用 |

## 3. 监控信号

核心指标（见 `docs/observability/metrics.md`）：

- `ws_connections_active`：连接数趋势；逼近 `max_ws_connections` 需扩容（阶段 B）或排查泄漏。
- `message_e2e_latency_seconds`：P99 上升说明投递链路退化。
- `seq_alloc_duration_seconds`：升高指向热点会话写锁争用。
- `push_total{result="error"}`：推送失败率，持续非零排查下行链路。
- HTTP 4xx/429：限流是否过紧；`/readyz` 失败说明依赖（DB/Redis/对象存储）不可用。

链路排查：按 traceId 反查 `logs.app`（trace 与日志已联动）。

## 4. 常见故障处置

- **消息发不出（429）**：确认 `im.message_rate_*` 是否过紧；查 `push_total`/限流中间件日志。
- **WS 频繁断连**：核对 `ws_pong_wait` 与客户端心跳间隔；`ws_connections_active` 是否触顶返回 1013。
- **logout 后旧 token 仍可用**：确认 `redis.enabled=true`（JWT 黑名单依赖 Redis）；黑名单不可用时吊销 fail-open。
- **迁移并发冲突**：`migrate` 使用 MySQL 命名锁（`GET_LOCK`），多副本同时迁移仅一个执行 DDL，其余等待。
- **依赖故障摘流**：`/readyz` 聚合 DB/Redis/对象存储探针，任一失败即返回非 200，由探针驱动摘流。

## 5. 上线前检查清单

- [ ] `make verify` 全绿
- [ ] prod 配置满足 `server.tls.enabled` 或 `server.behind_tls_proxy`
- [ ] `replicas: 1` 且 `sessionAffinity: ClientIP`
- [ ] Redis 启用（presence / ticket / JWT 吊销）
- [ ] 迁移 Job 在应用启动前完成
- [ ] Prometheus 采集 `/metrics`，告警覆盖 e2e 延迟与 readyz
