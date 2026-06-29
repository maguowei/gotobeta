# IM 后端生产化演进 · 技术设计与路线图

> 状态：已确认设计，待生成阶段 A 实施计划
> 日期：2026-06-29
> 范围：在已落地的第一期 IM 后端（F/M1–M7，`make verify` 70.8% 覆盖）之上，分阶段演进到生产可用并对齐业界最佳实践
> 前序文档：`docs/superpowers/specs/2026-06-29-ai-native-im-backend-design.md`（第一期设计）、`docs/superpowers/plans/2026-06-29-ai-native-im-backend.md`（第一期计划）
> 参照 skill：`im-best-practices`、`database-design-best-practices`、`permission-design-best-practices`

## 1. 背景与目标

第一期已实现基础 IM 后端：`workspace`(动态 RBAC/ACL/DataScope) / `messaging`(会话 + 每会话 seq + 消息 + 撤回 + 已读水位 + 读扩散 timeline) / `realtime`(WS 网关 + 进程内 Hub + presence/typing) / `media`(S3 预签名) 四模块，配套 `internal/pkg/{authz,event,imevent,imrt,rbac}` 与 `internal/infra/{eventbus,objstore,cache}` 端口，22 张 Ent schema。

本设计的目标是把它演进到**生产可用**且**符合业界最佳实践**，并明确"先加固再演进"的分阶段路线：

- **阶段 A — 生产加固**：不改变架构形态（单进程 + MySQL + 可选 Redis），补齐可靠性、安全、纵深防御、可观测性、运维要素，使系统在**单实例部署**下达到生产标准、短期可上线。
- **阶段 B — 水平扩展演进**：按第一期设计文档第 7 节，每个演进点以端口隔离平滑替换，解除单实例约束，支撑百万级连接与多实例。

本文档对阶段 A 给出逐里程碑详细设计，对阶段 B 给出端口隔离纲要。阶段 B 待阶段 A 落地后另起 spec→plan 周期。

### 1.1 关键约束

- **阶段 A 单实例部署**：当前 `realtime` 使用进程内 Hub（`infra/hub/hub.go`）+ 进程内 EventBus（`infra/eventbus/inproc.go`），多实例间消息不互通、presence 不共享。阶段 A 不解决跨实例路由，部署文档与就绪策略必须显式声明"单实例 + 竖直扩容"，多实例横向扩展是阶段 B 的目标。

## 2. 现状审计结论（经源码核实）

下表是对当前实现的核实结论，**纠正了若干常见误判**，作为阶段 A 里程碑的依据。

### 2.1 已正确实现、无需返工

| 项 | 证据 | 结论 |
|---|---|---|
| seq 分配并发正确性 | `internal/modules/messaging/infra/seqalloc/db_allocator.go:31-34` | 用 `UPDATE conversations SET last_seq = last_seq + 1`（原子自增取行写锁，持有至提交），在 `RunInTx` 内执行；同事务回读看到自身写入，并发事务阻塞在行锁上。**正确**。"两线程都读到同值"的竞态不成立；"Ent 不支持 FOR UPDATE"亦为误判（Ent 有 `.ForUpdate()`，此处用原子自增本不需要）。仅回读多一次往返，属可选微优化，不入阶段 A。 |
| 幂等发送 | `application/service/message_commands.go:42-47` + `messages` 表 `uk(conversation_id,client_msg_id)` | 先查 `FindByClientMsgID` 命中返回原结果，唯一索引兜底。**正确**。 |
| 事务边界 | `message_commands.go:54-74` + `internal/infra/entdb/tx.go` | seq 分配 + 消息插入 + 会话游标更新同一 `RunInTx`，任一失败整体回滚。**正确**。 |
| 撤回服务端时间校验 | `domain/message/message.go:143-154` + `message_commands.go:85-163` | `now.Sub(serverTime) > window` 用服务端时间；撤回插控制条目占新 seq。**正确**。 |
| 已读水位单调 + 未读推导 | `domain/conversation/member.go:96-113` | `MarkRead` 严格 `>` 单调；`Unread = lastSeq - readSeq` 推导不双写。**正确**。 |
| owner 受 ACL deny 约束 | `infra/authz/checker.go:40-47` | owner 视为 RBAC 允许但仍受 ACL 显式 deny 约束 —— 与第一期设计决策 E 一致，**非 bug**。 |
| 领域事件外发 | `message_commands.go` + `internal/pkg/imevent/` | `MessageCreated`/`ReadUpdated` 发布到 eventbus，realtime dispatcher 订阅扇出。**正确**。 |

### 2.2 经核实的真实缺口（阶段 A 范围）

| 严重度 | 缺口 | 证据 |
|---|---|---|
| 高 | **DataScope 仅靠应用层手动传 `workspace_id`，无纵深防御** | JWT claims 无 workspace_id（grep `internal/pkg/auth` 无结果）；无 Ent 拦截器；第一期设计决策 E 承诺的"repo 层 Ent 拦截器统一注入"未实现。任一处漏写成员校验即跨工作区越权。 |
| 高 | **权限版本化缓存与变更审计未接线** | `rbac_permission_versions` / `rbac_permission_change_logs` schema 已建但无任何递增/写入/读取代码。`ResolveUserActions` 每次三表查询，无缓存、无失效、无审计。 |
| 高 | **角色软删除未过滤** | `infra/persistence/rbac_repository.go` `ResolveUserActions` 过滤了 `permission.status=1`，但 user_role→role 链未过滤 `role.status=1`，被停用角色权限仍解析。 |
| 高 | **限流仅覆盖认证端点** | `internal/modules/user/adapter/http/middleware/ratelimit.go` 仅用于 OAuth/email；发消息、WS 握手、全局入口无限流；WS 无连接数上限。 |
| 中高 | **WS 健壮性缺口** | `adapter/ws/handler.go` `CheckOrigin` 恒 true；`adapter/ws/conn.go:36-43` 发送缓冲满 `default:` 静默丢弃无指标；presence 无 TTL 续期（`infra/presence`）；readPump/writePump 不监听 `ctx.Done()`；优雅关闭不通知 WS 客户端。 |
| 中 | **可观测性盲点** | `internal/infra/metrics/metrics.go` 无 IM 指标（WS 连接数、消息延迟、seq 耗时、推送成功率、eventbus 队列深度）；消息→推送无 span。 |
| 中 | **健康检查仅探活 DB** | `internal/app/server/server.go:100-102` 仅注册 database checker；缺 Redis/objstore；readiness 不反映依赖故障。 |
| 中 | **安全要素缺失** | 无 TLS/HTTPS 配置项；无 CORS 中间件；JWT 无吊销/logout 黑名单；输入仅 `binding:"required"`，无 contentType 枚举、clientMsgId 长度、content 结构、reply_to 存在性校验。 |
| 中 | **DB 迁移无分布式锁** | `cmd/migrate/main.go` 直接 `Schema.Create`，多副本并发迁移可能冲突；K8s 无 migrate Job/preStop hook。 |
| 低中 | **DB schema 演进预留不足** | `messages` 无分区预留与归档策略；缺 `sender_id`/`creator_id` 索引；逻辑外键无标注。 |

### 2.3 归阶段 B（端口隔离，不入阶段 A）

- 进程内 Hub/EventBus → 跨实例路由（Redis/Kafka）；独立 WS Gateway；seq → Redis INCR/seqsvr；离线推送 APNs/FCM；消息分区/宽列；全文/语义搜索；读写扩散混合；字段级权限/ABAC；AI 功能。

## 3. 阶段 A — 生产加固里程碑

每个里程碑独立可测、可提交，遵循"行为变更先写测试 / HTTP 变更先改 openapi.yaml / 完成跑 `make verify`"。建议执行顺序：A1→A2→A3→A4→A5→A6→A7→A8（安全正确性 → 可用性 → 可观测运维 → 安全收尾）。

### A1：DataScope 纵深防御

**目标**：工作区隔离不再单点依赖应用层手动传参，建立 repo 层兜底，越权访问在多层被拦截。

**现状**：`workspace_id` 由 HTTP path 参数经应用层校验成员关系后传入 repo；无 JWT claim、无 Ent 拦截器。

**改动面**：
- 鉴权上下文：扩展 `internal/pkg/auth` claims 或 requestctx，承载 `current_workspace_id`（从 JWT 或受信中间件解析，**严禁从请求体读取**）；提供 `requestctx.WorkspaceID(ctx)`。
- 中间件：新增 workspace scope 中间件，将 path 的 `{ws}` 与鉴权上下文校验一致后注入 ctx；不一致直接 403。
- repo 层兜底：在 `internal/infra/entdb` 增加按 ctx 注入 `workspace_id` 过滤的统一查询包装或 Ent 拦截器（覆盖带 workspace_id 的表）；超管/ACL 命中也不可跨工作区。
- 应用层：所有工作区内用例入口断言 `ctx.workspace_id == cmd.WorkspaceID`。

**验收**：
- 集成测试：用户 A（仅属 ws1）请求 ws2 资源 → 403，且即使应用层漏校验，repo 层过滤也不返回 ws2 数据。
- 架构测试不破坏分层（拦截器在 infra，不 import modules）。

### A2：权限缓存版本化 + 变更审计

**目标**：权限解析有缓存且变更即精准失效（不靠 TTL 漂移）；所有授权变更可审计；角色软删除生效。

**改动面**：
- 版本号：`AssignRole`/`RevokeRole`/`BindRolePermission`/ACL `Grant`/`Revoke` 成功后递增对应 subject 的 `rbac_permission_versions.version`（UPSERT）。
- 缓存：`ResolveUserActions` 经 `internal/infra/cache` 读 `perm:user:{ws}:{uid}:v{version}`，命中返回，未命中查 DB 后回填；Redis 关闭时直查 DB（保持现有行为）。
- 审计：上述变更操作写 `rbac_permission_change_logs`（`before_json`/`after_json`/`operator_id`/`request_id`/`reason`），`request_id` 从 requestctx 取。
- 角色状态过滤：`ResolveUserActions` 与 `ListUserRoleIDs` 增加 `role.status=1` 过滤。

**验收**：
- 单测：变更后 version 递增、缓存键随之变化；停用角色后其权限不再解析。
- 集成测试：变更产生一条 change_log，含完整 before/after/operator。
- Redis 关闭路径回退 DB 正确。

### A3：限流与过载保护

**目标**：入口、发消息、WS 连接均有上限，恶意客户端无法耗尽资源。

**改动面**：
- 通用限流：将现有 `ratelimit` 中间件提升为可复用组件（或在 `internal/pkg`/`internal/infra` 提供限流端口），支持按 IP / 按用户维度。
- 发消息频控：发送用例前置按用户（可选按会话）令牌桶限流，超限返回 429。
- WS：握手限流（按 IP）；`Hub` 增加全局连接数上限与单用户多端上限，超限拒绝升级（503/403）。
- HTTP server：显式 `MaxHeaderBytes`、请求体大小上限中间件。
- 配置：`im.max_ws_connections` / `im.max_conn_per_user` / `im.ws_handshake_rate` / `im.message_rate_per_user` 等，带安全默认值。

**验收**：单测/集成：超限返回 429/503；连接数达上限拒绝；配置默认值加载正确。

### A4：WS 生产健壮性

**目标**：WS 网关在弱网、慢消费者、滚动重启下行为可预期，不静默丢消息。

**改动面**：
- `CheckOrigin`：改为按配置白名单校验（与 CORS 白名单共用配置）。
- 发送缓冲：`conn.go` 缓冲满时不再静默 `default:` 丢弃，改为记录指标（counter）+ 主动断连（由客户端重连后 HTTP 按 last_seq 补拉补偿），缓冲大小配置化。
- presence 续期：`writePump` 心跳周期内调用 `presence.Refresh` 续期 TTL；`PresenceReporter` 增 `Refresh`。
- 优雅退出：readPump/writePump 监听 `ctx.Done()`；`Hub.GracefulShutdown` 广播 close 帧并等待断开/超时；组合根在 HTTP Shutdown 前调用；eventbus 订阅可取消。
- 超时配置化：`writeWait`/`pongWait`/`pingPeriod`/`readLimit` 迁移到 config。

**验收**：
- WS 集成测试：缓冲压满 → 断连且指标递增，重连后 HTTP 补拉一致；ticket 鉴权与心跳不回归。
- 优雅关闭：触发 shutdown，客户端收到 close 帧，进程在超时内退出且无 goroutine 泄漏。

### A5：可观测性补全

**目标**：IM 关键链路有指标与链路追踪，故障可定位。

**改动面**：
- 指标（`internal/infra/metrics`）：WS 活跃连接 gauge、消息端到端延迟 histogram、seq 分配耗时 histogram、推送 success/failure counter、eventbus 队列深度/分发延迟。
- 链路：`MessageCreated` 发布与 dispatcher 扇出包 span，串联"发消息 → 推送 signal"；WS frame 处理关键路径埋点。
- Sentry：增加 `sample_rate` 配置，避免大流量过载。

**验收**：指标在 `/metrics` 暴露且命名符合现有 namespace；trace 中可见消息发布到推送 span 链。

### A6：健康检查与运维就绪

**目标**：依赖故障能在 readiness 反映；迁移在多副本下安全；部署资产完整。

**改动面**：
- 健康检查：`readyz` 注册 Redis、objstore checker（Redis/objstore 关闭时按"可选依赖"语义处理，启用才探活）。
- 迁移并发安全：`cmd/migrate` 用 MySQL advisory lock（`GET_LOCK`）包裹 `Schema.Create`，确保单写者。
- 部署：K8s 增 migrate 作为 init Job/preStart；Pod 增 `preStop` hook 给优雅关闭留时间；WS 走 `sessionAffinity=ClientIP`（单实例下也利于一致性）；`docker-compose` 补 MinIO 依赖。

**验收**：依赖宕机时 `readyz` 返回 degraded；并发跑两个 migrate 仅一个执行 DDL；compose up 后冒烟通过。

### A7：安全与输入校验

**目标**：传输与跨域安全、令牌可吊销、系统边界严格校验。

**改动面**：
- TLS/HTTPS：config 增 `server.tls.cert/key`，提供 HTTPS listener（或显式声明由反代终止 TLS 并在文档固化）；生产校验强制其一。
- CORS：新增 CORS 中间件，白名单配置驱动（与 WS Origin 白名单共用）。
- JWT 吊销：logout 端点 + Redis token 黑名单（按 jti / 过期时间），鉴权中间件校验黑名单；Redis 关闭时降级说明。
- 输入校验：发消息等边界校验 `content_type` 枚举范围、`client_msg_id` 长度（≤64）、`content` 结构、`reply_to_msg_id` 存在性。

**验收**：单测：非法 contentType/超长 clientMsgId/越界 content 被拒；logout 后旧 token 401；生产配置缺 TLS 且无反代声明时启动校验失败。

### A8：DB schema 演进预留 + 收尾

**目标**：为阶段 B 存储演进留好缝，补齐缺失索引，全量验证与文档。

**改动面**：
- 索引：`messages.sender_id`、`conversations.creator_id` 按查询需要补索引（评估必要性，避免无用索引）。
- 分区预留：`messages` 按 `conversation_id + created_at` 的分区与冷热/归档策略**文档化**（阶段 A 不强制建分区，给出迁移路径）；逻辑外键在 schema 注释标注。
- 收尾：全量 `make verify`；更新 `docs/`（运维说明、可观测性指标清单、单实例部署约束）、README IM 模块概览。

**验收**：`make verify` 全绿；schema 变更有防漂移；文档与实现一致。

## 4. 阶段 B — 水平扩展演进纲要

所有演进点以 interface 隔离、组合根注入，替换实现不触碰用例与领域层。待阶段 A 落地后另起 spec→plan。

| 维度 | 阶段 A 现状 | 阶段 B 演进 |
|---|---|---|
| 消息扇出 / 连接 | 进程内 EventBus + 进程内 Hub（单实例） | Redis Streams/pub-sub 跨实例路由 → Kafka；连接元数据入 Redis，Conn 仍本地持有写 |
| WS 网关 | 与业务同进程 | 独立无状态 Gateway 服务，仅长连接 + 协议 + 鉴权 + 本机下行 |
| seq 分配 | DB 行锁（`SeqAllocator` 端口） | Redis INCR → 独立 seqsvr |
| 事件总线 | 进程内 EventBus（`event.Publisher` 端口） | Redis Streams → Kafka |
| 离线触达 | WS 在线信号 + HTTP 补拉 | `Pusher` 端口 + APNs/FCM + 设备 token 表 + 离线唤醒 |
| 消息存储 | MySQL + `uk(conv,seq)` + 分区预留 | 按 conv+时间分区 → 宽列(ScyllaDB) |
| 群聊扩散 | 读扩散统一 timeline | 大群读扩散 / 中小群写扩散混合（按规模压测拐点） |
| 全文 / 语义搜索 | 无（`Search` 端口空实现预留） | ES / 向量库（随 AI） |
| 权限 | 动态 RBAC + ACL + 版本化缓存 | 字段级权限 / scope_rules 多维范围 / ABAC 动态条件 |
| 协议 | JSON 帧 | Protobuf；QUIC 弱网优化 |
| AI | 领域事件 + Bot 一等公民已建模 | 订阅 `MessageCreated` 做总结 / 智能回复 / Agent（以 bot 身份发消息） |

## 5. 测试策略

- 沿用第一期分层测试：domain 单测、application fake 依赖单测、testcontainers 集成测试、WS 集成测试、`make test-architecture` 守护边界。
- 阶段 A 每里程碑新增针对性测试：
  - A1：跨工作区越权（应用层 + repo 层双层）。
  - A2：版本递增/缓存失效/角色软删除过滤/审计写入。
  - A3：限流 429/连接数上限。
  - A4：缓冲压满断连 + 重连补拉、优雅关闭无泄漏。
  - A5：指标暴露与命名、span 链。
  - A6：依赖故障 readiness、并发 migrate 互斥。
  - A7：输入校验、logout 吊销、生产 TLS 校验。
- 完成跑 `make verify`（受限环境带 `GOCACHE=/tmp/go-build-cache GOMODCACHE=/tmp/go-mod-cache`），覆盖率维持 ≥ 70%。

## 6. 交付物与后续

- 本 spec：分阶段策略 + 阶段 A 逐里程碑详细设计 + 阶段 B 纲要。
- 随后 `writing-plans`：仅为**阶段 A（A1–A8）**生成逐任务可执行计划（TDD + openapi-first + make verify）。
- 阶段 B：阶段 A 落地后另起 spec→plan 周期。
- 提交：中文 Conventional Commits。
