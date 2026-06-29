# IM 后端生产化加固 · 阶段 A 实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: 用 superpowers:subagent-driven-development 或 superpowers:executing-plans 逐任务实施。步骤用 `- [ ]` 复选框跟踪。

**Goal:** 在不改变单进程架构形态的前提下，为已落地的第一期 IM 后端补齐生产可用所需的纵深防御、可靠性、可观测性、安全与运维要素（阶段 A，A1–A8）。

**Architecture:** 沿用 `internal/modules/<name>/{domain,application,infra,adapter}` 分层与组合根装配。新增能力以端口隔离、组合根注入：workspace scope 中间件 + Ent 拦截器兜底 DataScope；权限版本化缓存与审计接线；可复用限流中间件；WS 健壮性改造；IM 指标与链路；健康探活与迁移分布式锁；TLS/CORS/JWT 吊销/输入校验。

**Tech Stack:** Go 1.26、gin、Ent ORM、MySQL、Redis(go-redis，可选)、gorilla/websocket、Prometheus、OTel/Sentry。

**Spec:** `docs/superpowers/specs/2026-06-29-im-production-evolution-design.md`

## Global Constraints

- 模块路径 `github.com/maguowei/gotobeta`；依赖方向 adapter → application → domain ← infra，由 `make test-architecture` 校验。
- domain 层零外部依赖；跨模块只经 `internal/pkg` 端口或领域事件协作；组合根在 `internal/app/**` 装配。
- 第三方 SDK 唯一归口：go-redis→`internal/infra/cache`、minio-go→`internal/infra/objstore`、jwt→`internal/pkg/auth`、prometheus→`internal/infra/metrics`。
- 日志必须 `log/slog` 的 `*Context` 方法（注入 traceId/requestId）；错误日志用 `logger.WithError(ctx, l, msg, err, attrs...)`；敏感字段经 `internal/pkg/sensitive` 脱敏。
- 配置只在组合根读取经构造函数注入；typed config 默认值安全；prod 环境配置校验严格。
- HTTP 变更先改 `api/openapi.yaml`；行为变更先写测试（TDD）；adapter `request`/`response` 不得 import domain。
- 每个里程碑完成跑 `make verify`（受限环境 `GOCACHE=/tmp/go-build-cache GOMODCACHE=/tmp/go-mod-cache`），覆盖率 ≥ 70%。
- 提交用中文 Conventional Commits。
- **阶段 A 单实例部署**：进程内 Hub/EventBus 多实例不互通，部署文档与就绪策略显式声明单实例 + 竖直扩容。

## 里程碑与验证

| 阶段 | 交付 | 验证命令 |
|---|---|---|
| A1 | claims workspace + scope 中间件 + Ent 拦截器兜底 | `make test && make test-architecture` + 越权集成测试 |
| A2 | 权限版本化缓存 + 变更审计 + 角色软删除过滤 | `make test` + RBAC 集成测试 |
| A3 | 通用限流 + 发消息频控 + WS 连接上限 | `make test` |
| A4 | WS Origin 白名单 + 缓冲断连 + presence 续期 + 优雅关闭 | `make test` + WS 集成测试 |
| A5 | IM 指标 + 消息→推送 span + Sentry 采样 | `make test` |
| A6 | readyz 依赖探活 + migrate 分布式锁 + 部署资产 | `make test` + `docker compose config` + `kubectl --dry-run` |
| A7 | TLS/CORS/JWT 吊销/输入校验 | `make test` |
| A8 | 索引 + 分区文档 + 收尾 | 全量 `make verify` |

---

## 阶段 A1：DataScope 纵深防御

> 目标：工作区隔离不再单点依赖应用层手动传参。建立 claims→ctx→repo 三层兜底，越权访问被多层拦截。

### Task A1.1: claims 承载 workspace 上下文 + requestctx 读取

**Files:**
- Modify: `internal/pkg/requestctx/*.go`（新增 `WorkspaceID(ctx) (int64, bool)` 与 `WithWorkspaceID`）
- Test: `internal/pkg/requestctx/workspace_test.go`

**Interfaces — Produces:**
- `func WithWorkspaceID(ctx context.Context, wsID int64) context.Context`
- `func WorkspaceID(ctx context.Context) (int64, bool)`

- [ ] **Step 1: 写失败测试** —— `WithWorkspaceID` 写入后 `WorkspaceID` 读回相同值；未设置返回 `(0,false)`。
- [ ] **Step 2: 运行测试确认失败**（函数未定义）。
- [ ] **Step 3: 实现** —— 私有 ctxKey 类型 + 两个函数，镜像现有 requestctx 既有键实现风格。
- [ ] **Step 4: `go test ./internal/pkg/requestctx/...` 通过。**
- [ ] **Step 5: commit** `feat(requestctx): 承载 workspace 上下文`。

### Task A1.2: workspace scope 中间件

**Files:**
- Create: `internal/pkg/httpx/middleware/workspace_scope.go`
- Test: `internal/pkg/httpx/middleware/workspace_scope_test.go`

**Interfaces — Consumes:** `auth.ClaimsFromContext`、`requestctx.WithWorkspaceID`。
**Produces:** `func WorkspaceScope(paramName string) gin.HandlerFunc`（解析 path `:{paramName}` 工作区 id → 写入 ctx；与 claims 中工作区成员关系的一致性校验保留给应用层成员校验，但中间件确保 ctx 注入的 workspace_id 只来自受信 path 段，绝不从请求体读取）。

- [ ] **Step 1: 写失败测试** —— 构造带 path 参数 `ws=42` 的 gin 请求，断言下游 handler ctx 内 `requestctx.WorkspaceID(ctx)==42`；path 参数非数字时 400。
- [ ] **Step 2: 运行确认失败。**
- [ ] **Step 3: 实现** —— `c.Param(paramName)` 解析为 int64，失败 `apperr.InvalidParam`；成功 `c.Request = c.Request.WithContext(requestctx.WithWorkspaceID(...))`。
- [ ] **Step 4: 测试通过。**
- [ ] **Step 5: commit** `feat(middleware): 工作区 scope 注入中间件`。

### Task A1.3: Ent 拦截器统一注入 workspace_id 过滤（repo 层兜底）

**Files:**
- Create: `internal/infra/entdb/workspace_interceptor.go`
- Test: `internal/infra/entdb/workspace_interceptor_test.go`

**Interfaces — Produces:**
- `func WorkspaceScopeInterceptor() ent.Interceptor` —— 对带 `workspace_id` 字段的实体类型，在 ctx 含 workspace_id 且查询未显式声明跳过时，注入 `WHERE workspace_id = ?`（含平台模板 `workspace_id IN (0, current)` 语义按表区分：RBAC 模板表用 `IN(0,cur)`，业务表用 `=cur`）。
- `func WithoutWorkspaceScope(ctx context.Context) context.Context` —— 显式逃逸（跨工作区维护/seed/平台超管路径用），逃逸必须显式、可审计。

**注意：** Ent 拦截器作用于 `Query`。本任务覆盖读路径兜底；写路径越权由应用层 + path scope 防御（写入的 workspace_id 来自 ctx，不来自请求体）。架构测试：拦截器在 `internal/infra/entdb`，不得 import modules。

- [ ] **Step 1: 写失败测试**（集成，build tag `integration`，testcontainers MySQL）：两个工作区各建数据，ctx 注入 ws1 后 `client.RbacRole.Query().All()` 只返回 ws1 + 平台模板(0)；`WithoutWorkspaceScope` 返回全部。
- [ ] **Step 2: 运行确认失败。**
- [ ] **Step 3: 实现** —— `ent.InterceptFunc` + `ent.TraverseFunc`，用 `q.WhereP(sql.FieldEQ/FieldIn("workspace_id", ...))`；按实体类型白名单（带 workspace_id 的表）应用；逃逸键检查。
- [ ] **Step 4: 在 `entdb.NewEntClient` 装配 `client.Intercept(WorkspaceScopeInterceptor())`。**
- [ ] **Step 5: `make test-integration`（需 Docker）/ 受限环境 `make test-integration-compile` 通过；commit** `feat(entdb): workspace_id 查询拦截器兜底 DataScope`。

### Task A1.4: 工作区路由挂载 scope 中间件 + 应用层入口断言 + 越权集成测试

**Files:**
- Modify: `internal/modules/workspace/adapter/http/router/*.go`、`internal/modules/messaging/adapter/http/router/*.go`、`internal/modules/media/.../router/*.go`（带 `{ws}` 的路由组挂 `WorkspaceScope("ws")`）
- Modify: 应用层工作区内用例入口断言 `requestctx.WorkspaceID(ctx) == cmd.WorkspaceID`（不一致 `apperr.Forbidden`）
- Test: `internal/integration/workspace_isolation_suite_test.go`

- [ ] **Step 1: 写失败集成测试** —— 用户 A 属 ws1，请求 ws2 的会话列表 → 403；即便绕过应用层校验，repo 拦截器也不返回 ws2 数据。
- [ ] **Step 2: 运行确认失败。**
- [ ] **Step 3: 路由挂中间件 + 应用层断言。**
- [ ] **Step 4: `make test && make test-architecture` 通过。**
- [ ] **Step 5: commit** `feat(workspace): DataScope 纵深防御接线与越权测试`。

---

## 阶段 A2：权限缓存版本化 + 变更审计

> 目标：权限解析有缓存且变更即精准失效；授权变更可审计；角色软删除生效。

### Task A2.1: 角色软删除过滤

**Files:**
- Modify: `internal/modules/workspace/infra/persistence/rbac_repository.go`（`ResolveUserActions` 与 `ListUserRoleIDs` 联结时过滤 `role.status=1`）
- Test: `internal/modules/workspace/infra/persistence/rbac_repository_test.go` 或集成 `internal/integration/rbac_suite_test.go`

**Interfaces — Consumes:** 既有 `ResolveUserActions(ctx, ws, uid) (map[string]struct{}, error)`、`ListUserRoleIDs(ctx, ws, uid) ([]int64, error)`。

- [ ] **Step 1: 写失败集成测试** —— assign 一个角色后停用该角色（status=2），`ResolveUserActions` 不再返回其权限。
- [ ] **Step 2: 运行确认失败。**
- [ ] **Step 3: 实现** —— 解析 roleIDs 后增查 `RbacRole.Query().Where(BizIDIn(roleIDs...), StatusEQ(1))` 过滤有效角色，再查 role_permissions；`ListUserRoleIDs` 同样过滤。
- [ ] **Step 4: 测试通过。**
- [ ] **Step 5: commit** `fix(rbac): 解析权限时过滤已停用角色`。

### Task A2.2: 权限版本号端口 + 递增

**Files:**
- Modify: `internal/modules/workspace/domain/rbac/repository.go`（接口增 `BumpUserVersion(ctx, ws, uid int64) (int64, error)`、`GetUserVersion(ctx, ws, uid int64) (int64, error)`）
- Modify: `internal/modules/workspace/infra/persistence/rbac_repository.go`（UPSERT `rbac_permission_versions`）
- Modify: `internal/modules/workspace/application/service/workspace_commands.go`（`AssignRole`/`RevokeRole`/`BindRolePermission` 成功后 `BumpUserVersion`）
- Test: 集成测试断言版本递增

**Interfaces — Produces:** `BumpUserVersion(ctx, ws, uid int64) (newVersion int64, err error)`、`GetUserVersion(...) (int64, error)`（无记录返回版本 0）。

- [ ] **Step 1: 写失败集成测试** —— AssignRole 后 `GetUserVersion` 比之前 +1。
- [ ] **Step 2: 运行确认失败。**
- [ ] **Step 3: 实现 UPSERT** —— Ent `OnConflict` `AddVersion(1)` 或 select+update 事务；命令成功后调用。
- [ ] **Step 4: 测试通过。**
- [ ] **Step 5: commit** `feat(rbac): 授权变更递增权限版本号`。

### Task A2.3: 缓存包装 ResolveUserActions（版本键 + Redis 降级）

**Files:**
- Create: `internal/modules/workspace/infra/authz/cached_resolver.go`
- Test: `internal/modules/workspace/infra/authz/cached_resolver_test.go`
- Modify: `internal/modules/workspace/module.go`（kv 可用时包装 checker 的 resolver）

**Interfaces — Consumes:** `rbac.Repository`、`cache.RedisKV`（`Get/Set/Del`，若缺 Get 在 `internal/infra/cache` 补）。
**Produces:** `func NewCachedResolver(repo rbac.Repository, kv KV, ttl time.Duration) rbac.ActionResolver`；缓存键 `perm:user:{ws}:{uid}:v{version}`，命中返回，未命中查 DB 并回填；`kv==nil` 时直查 DB。

- [ ] **Step 1: 写失败单测**（fake repo + fake kv）：第一次查 DB 回填；第二次同版本命中缓存（DB 调用计数不增）；BumpVersion 后键变化→重新查 DB。
- [ ] **Step 2: 运行确认失败。**
- [ ] **Step 3: 实现** —— 读 version → 拼键 → kv.Get → 命中反序列化 actions；未命中 repo.ResolveUserActions + kv.Set(ttl)。
- [ ] **Step 4: 测试通过。**
- [ ] **Step 5: commit** `feat(authz): 权限动作集版本化缓存与 Redis 降级`。

### Task A2.4: 授权变更审计写入

**Files:**
- Modify: `internal/modules/workspace/domain/rbac/repository.go`（增 `RecordChange(ctx, log ChangeLog) error` 或新 audit 端口）
- Create: `internal/modules/workspace/infra/persistence/audit_repository.go`
- Modify: `application/service/workspace_commands.go` 与 ACL `Grant/Revoke`（变更后写 `rbac_permission_change_logs`，含 before/after/operator/request_id/reason）
- Test: 集成测试断言写入一条 change_log

**Interfaces — Produces:** `RecordChange(ctx, ws int64, changeType, targetType int8, targetID, operatorID int64, requestID string, before, after map[string]any, reason string) error`。operatorID 从 claims，requestID 从 requestctx。

- [ ] **Step 1: 写失败集成测试** —— AssignRole 后查 `rbac_permission_change_logs` 有一条含 operator/target/after。
- [ ] **Step 2: 运行确认失败。**
- [ ] **Step 3: 实现写入 + 命令接线。**
- [ ] **Step 4: `make test && make test-architecture` 通过。**
- [ ] **Step 5: commit** `feat(rbac): 授权变更审计落库`。

---

## 阶段 A3：限流与过载保护

### Task A3.1: 提炼可复用 keyed 限流中间件到 internal/pkg

**Files:**
- Create: `internal/pkg/httpx/middleware/ratelimit.go`（从 user 模块的实现提炼：令牌桶 + 惰性清理 + 可选 keyFunc）
- Test: `internal/pkg/httpx/middleware/ratelimit_test.go`
- Modify: `internal/modules/user/.../middleware/ratelimit.go` 改为复用共享实现（或保留薄封装）

**Interfaces — Produces:**
- `type Limiter struct{...}`；`func NewLimiter(requestsPerMinute, burst int) *Limiter`
- `func (l *Limiter) Middleware(keyFunc func(*gin.Context) string) gin.HandlerFunc`（keyFunc nil 时按 ClientIP）

- [ ] **Step 1: 写失败测试** —— burst 内放行、超限 429；按 keyFunc 分桶（不同 key 独立）。
- [ ] **Step 2: 运行确认失败。**
- [ ] **Step 3: 实现**（镜像 user 模块逻辑，抽出 keyFunc）。
- [ ] **Step 4: user 模块改为复用，`make test` 通过（含 user ratelimit 既有测试不回归）。**
- [ ] **Step 5: commit** `refactor(middleware): 提炼可复用 keyed 限流中间件`。

### Task A3.2: 发消息按用户频控

**Files:**
- Modify: `internal/modules/messaging/adapter/http/router/*.go`（发消息路由挂按用户限流：keyFunc = claims.UserID）
- Modify: `internal/infra/config/config.go`（`im.message_rate_per_minute` 默认 120、`im.message_rate_burst` 默认 20）+ `configs/*.yaml`
- Test: `internal/modules/messaging/adapter/http/router/router_test.go`

- [ ] **Step 1: 写失败测试** —— 同一用户高频发消息触发 429；不同用户互不影响。
- [ ] **Step 2: 运行确认失败。**
- [ ] **Step 3: 配置 + 装配（按 UserID 限流）。**
- [ ] **Step 4: 测试通过。**
- [ ] **Step 5: commit** `feat(messaging): 发消息按用户频控`。

### Task A3.3: WS 连接数上限 + 握手限流

**Files:**
- Modify: `internal/modules/realtime/infra/hub/hub.go`（`Register` 返回 `bool`/error，达上限拒绝；新增 `ConnectionCount()`、`UserConnectionCount(uid)`；构造支持 maxTotal/maxPerUser）
- Modify: `internal/modules/realtime/adapter/ws/handler.go`（注册失败返回 503；握手按 IP 限流）
- Modify: `internal/modules/realtime/module.go` + config（`im.max_ws_connections` 默认 10000、`im.max_conn_per_user` 默认 10、`im.ws_handshake_rate_per_minute` 默认 60）
- Test: `internal/modules/realtime/infra/hub/hub_test.go`、`adapter/ws` 集成

**Interfaces — Produces:** `func New(maxTotal, maxPerUser int) *Hub`；`Register` 改为 `Register(userID int64, c imrt.Connection) bool`（imrt.Registry 接口同步更新）。

- [ ] **Step 1: 写失败测试** —— maxPerUser=1 时第二条连接 Register 返回 false；ConnectionCount 正确；达全局上限拒绝。
- [ ] **Step 2: 运行确认失败。**
- [ ] **Step 3: 实现 Hub 上限 + handler 503 + 握手限流。**
- [ ] **Step 4: `make test`（含 realtime 既有测试不回归）通过。**
- [ ] **Step 5: commit** `feat(realtime): WS 连接数上限与握手限流`。

### Task A3.4: HTTP 请求体大小上限 + MaxHeaderBytes

**Files:**
- Modify: `internal/app/server/server.go`（`http.Server.MaxHeaderBytes`；全局 `MaxBytesReader` 中间件或 gin `MaxMultipartMemory`）
- Modify: config（`server.max_request_body_bytes` 默认 1MB；上传走 media presign 不经此）
- Test: `internal/app/server` 或中间件测试

- [ ] **Step 1: 写失败测试** —— 超大 body POST 返回 413。
- [ ] **Step 2: 运行确认失败。**
- [ ] **Step 3: 实现中间件 + server 字段。**
- [ ] **Step 4: 测试通过；`make test`。**
- [ ] **Step 5: commit** `feat(server): 请求体与头部大小上限`。

---

## 阶段 A4：WS 生产健壮性

### Task A4.1: CheckOrigin 白名单

**Files:**
- Modify: `internal/modules/realtime/adapter/ws/handler.go`（`CheckOrigin` 按配置白名单；空白名单时按 same-origin 或拒绝跨域）
- Modify: `internal/modules/realtime/module.go` + config（`im.ws_allowed_origins []string`，与 A7 CORS 白名单共享配置）
- Test: `adapter/ws` 测试

- [ ] **Step 1: 写失败测试** —— Origin 在白名单握手成功；不在则拒绝。
- [ ] **Step 2: 运行确认失败。**
- [ ] **Step 3: 实现白名单校验。**
- [ ] **Step 4: 测试通过。**
- [ ] **Step 5: commit** `feat(realtime): WS Origin 白名单校验`。

### Task A4.2: 发送缓冲满 → 断连（不静默丢）+ 指标钩子

**Files:**
- Modify: `internal/modules/realtime/adapter/ws/conn.go`（`Send` 缓冲满时不再 `default:` 丢弃，改为标记溢出 → 触发 `close()`；暴露 `OnOverflow func()` 钩子供指标计数；缓冲大小配置化）
- Test: `adapter/ws/conn_test.go`

**Interfaces — Produces:** `newConn(userID, ws, bufSize int, onOverflow func())`。

- [ ] **Step 1: 写失败测试** —— 用阻塞的假 ws 写，灌满缓冲后 `Send` 触发 onOverflow 且 `closed` 被关闭。
- [ ] **Step 2: 运行确认失败。**
- [ ] **Step 3: 实现** —— 缓冲满分支调用 onOverflow + close（客户端重连后 HTTP 按 last_seq 补拉补偿）。
- [ ] **Step 4: 测试通过。**
- [ ] **Step 5: commit** `feat(realtime): 写缓冲溢出主动断连替代静默丢弃`。

### Task A4.3: presence TTL 周期续期

**Files:**
- Modify: `internal/modules/realtime/infra/presence/presence.go`（新增 `Refresh(ctx, userID) error`）
- Modify: `adapter/ws` PresenceReporter 接口增 `Refresh`，writePump 心跳周期调用续期
- Modify: `internal/modules/realtime/ephemeral.go`/`presence reporter` 实现 Refresh
- Test: `presence_test.go`

**Interfaces — Produces:** `func (s *Store) Refresh(ctx, userID int64) error`（Redis 模式刷新 TTL，内存模式 no-op）。

- [ ] **Step 1: 写失败测试** —— Refresh 调用 kv.Set 刷新 TTL（fake kv 断言）。
- [ ] **Step 2: 运行确认失败。**
- [ ] **Step 3: 实现 Refresh + writePump 周期调用（续期间隔 = pongWait/2）。**
- [ ] **Step 4: 测试通过。**
- [ ] **Step 5: commit** `feat(realtime): presence 周期续期防误判离线`。

### Task A4.4: 后台 goroutine 优雅退出 + Hub.GracefulShutdown

**Files:**
- Modify: `internal/modules/realtime/infra/hub/hub.go`（`GracefulShutdown(ctx)`：广播 close 帧 + 等待断开/超时）
- Modify: `adapter/ws/conn.go`/`handler.go`（readPump/writePump 监听 `ctx.Done()`）
- Modify: `internal/modules/realtime/module.go`（暴露 `Shutdown(ctx)`）+ `internal/app/server/server.go`（HTTP Shutdown 前调用 realtime Shutdown）
- Test: `hub_test.go` + WS 集成

**Interfaces — Produces:** `func (h *Hub) GracefulShutdown(ctx context.Context) error`、`func (m *Module) Shutdown(ctx context.Context) error`。

- [ ] **Step 1: 写失败测试** —— 注册若干假连接，GracefulShutdown 后全部收到 close 帧且 ConnectionCount 归零或超时返回。
- [ ] **Step 2: 运行确认失败。**
- [ ] **Step 3: 实现 + 组合根接线（server.go select ctx.Done 分支内先 realtimeMod.Shutdown 再 HTTP Shutdown）。**
- [ ] **Step 4: `make build` + WS 集成测试通过。**
- [ ] **Step 5: commit** `feat(realtime): 优雅关闭广播与后台 goroutine 退出`。

### Task A4.5: WS 超时参数配置化

**Files:**
- Modify: `adapter/ws/conn.go`、`handler.go`（writeWait/pongWait/pingPeriod/readLimit 由构造参数注入）
- Modify: `module.go` + config（`im.ws_pong_wait` 默认 60s、`im.ws_write_wait` 默认 10s、`im.ws_read_limit` 默认 4096）
- Test: 既有 WS 测试不回归

- [ ] **Step 1: 写测试** —— 自定义短 pongWait 时读超时更快（或断言参数透传）。
- [ ] **Step 2: 运行确认失败/调整。**
- [ ] **Step 3: 实现参数化。**
- [ ] **Step 4: `make test` 通过。**
- [ ] **Step 5: commit** `feat(realtime): WS 超时参数配置化`。

---

## 阶段 A5：可观测性补全

### Task A5.1: IM 指标收集器

**Files:**
- Modify: `internal/infra/metrics/metrics.go`（Collectors 增字段：`WSConnectionsActive prometheus.Gauge`、`MessageE2ELatency prometheus.Histogram`、`SeqAllocDuration prometheus.Histogram`、`PushTotal *prometheus.CounterVec{result}`、`EventBusQueueDepth prometheus.Gauge`；注册 + nil-safe Observe 方法）
- Test: `internal/infra/metrics/metrics_test.go`（注册后 collect 不 panic，指标存在）

**Interfaces — Produces:** `ObserveSeqAlloc(ctx, dur)`、`ObserveMessageLatency(ctx, dur)`、`IncPush(result string)`、`SetWSConnections(n float64)`、nil-safe。

- [ ] **Step 1: 写失败测试** —— NewCollectors 后断言新指标非 nil；Observe 方法 nil receiver 安全。
- [ ] **Step 2: 运行确认失败。**
- [ ] **Step 3: 实现新指标 + 注册 + Observe 方法（镜像现有 ObserveEventBus 风格）。**
- [ ] **Step 4: 测试通过。**
- [ ] **Step 5: commit** `feat(metrics): 新增 IM 关键指标`。

### Task A5.2: 指标埋点接线

**Files:**
- Modify: `internal/modules/messaging/.../service/message_commands.go`（seq 分配耗时、消息端到端延迟埋点；经注入的 metrics 端口，nil-safe）
- Modify: `internal/modules/realtime/dispatcher.go`（推送成功/失败计数）、hub Register/Unregister（连接数 gauge）
- Modify: 组合根注入 metrics collectors 到 messaging/realtime（构造函数新增可选参数或 functional option）
- Test: service/dispatcher 单测断言 Observe 被调用（fake metrics）

- [ ] **Step 1: 写失败测试** —— 发送消息后 fake metrics 记录一次 seq 分配观测；dispatcher 推送后 IncPush。
- [ ] **Step 2: 运行确认失败。**
- [ ] **Step 3: 实现埋点 + 注入。**
- [ ] **Step 4: `make test && make test-architecture` 通过（metrics 端口在 infra/pkg，注入不破坏分层）。**
- [ ] **Step 5: commit** `feat(observability): IM 指标埋点接线`。

### Task A5.3: 消息→推送链路 span + Sentry 采样

**Files:**
- Modify: `dispatcher.go`（`OnMessageCreated` 起 span `realtime.dispatch`，从事件恢复 trace context）
- Modify: `internal/infra/sentry/sentry.go` + config（`sentry.sample_rate` 默认 1.0，prod 可降）
- Test: sentry 配置默认值测试 + dispatcher span 测试（断言 span 创建）

- [ ] **Step 1: 写失败测试** —— sentry 配置加载 sample_rate 默认值；dispatcher 处理事件时创建 span（用 tracetest SpanRecorder）。
- [ ] **Step 2: 运行确认失败。**
- [ ] **Step 3: 实现 span + 采样配置。**
- [ ] **Step 4: `make test` 通过。**
- [ ] **Step 5: commit** `feat(observability): 推送链路 span 与 Sentry 采样`。

---

## 阶段 A6：健康检查与运维就绪

### Task A6.1: readyz 增 Redis / objstore 探活

**Files:**
- Modify: `internal/app/server/server.go`（Redis 启用时注册 `redis` checker（PING）；objstore 注册 `objstore` checker（BucketExists/Ping））
- Modify: `internal/infra/cache`（暴露 Ping helper）、`internal/infra/objstore`（暴露健康检查方法）
- Test: server 装配测试或各 infra 包测试

- [ ] **Step 1: 写失败测试** —— 健康注册表含 redis/objstore checker；依赖不可用时返回错误字符串。
- [ ] **Step 2: 运行确认失败。**
- [ ] **Step 3: 实现 checker（Redis 关闭时不注册，按可选依赖语义）。**
- [ ] **Step 4: `make test` 通过。**
- [ ] **Step 5: commit** `feat(health): readyz 探活 Redis 与对象存储`。

### Task A6.2: migrate 分布式锁

**Files:**
- Modify: `internal/app/server/migrate.go`（`runMigrate` 用 MySQL `GET_LOCK('gotobeta_migrate', timeout)` 包裹 `Schema.Create`，结束 `RELEASE_LOCK`；拿不到锁则跳过并日志）
- Test: `internal/app/server/migrate_test.go`（mock/sqlmock 或集成断言串行）

- [ ] **Step 1: 写失败测试** —— 并发两次 runMigrate 仅一个执行 DDL（集成）或断言 GET_LOCK 调用。
- [ ] **Step 2: 运行确认失败。**
- [ ] **Step 3: 实现 advisory lock。**
- [ ] **Step 4: `make test`（受限环境跑可用子集）。**
- [ ] **Step 5: commit** `feat(migrate): 迁移分布式锁防并发冲突`。

### Task A6.3: 部署资产完善

**Files:**
- Modify: `deployments/kubernetes/`（新增 migrate `Job` 或 initContainer；Pod `preStop` hook sleep；WS 用 `service.spec.sessionAffinity: ClientIP`）
- Modify: `docker-compose.yml`（补 MinIO 服务 + healthcheck + 卷）
- Modify: `Dockerfile`（如需，确认 migrate 二进制已构建——已存在）
- Test: `docker compose config` + `kubectl apply --dry-run=client -f deployments/kubernetes/`（缺工具则记录）

- [ ] **Step 1: 修改 compose 增 MinIO，`docker compose config` 通过。**
- [ ] **Step 2: 修改 k8s（migrate Job + preStop + sessionAffinity），`kubectl --dry-run` 通过（缺则检查 YAML）。**
- [ ] **Step 3: commit** `chore(deploy): MinIO 依赖、迁移 Job、preStop 与 WS 会话亲和`。

---

## 阶段 A7：安全与输入校验

### Task A7.1: 输入范围校验

**Files:**
- Modify: `internal/modules/messaging/adapter/http/request/message_request.go`（`content_type` 枚举范围、`client_msg_id` 长度 ≤64、`content` 非空 map、`reply_to_msg_id` ≥0）
- Modify: 应用层/领域层对 `reply_to_msg_id` 存在性校验（同会话存在）
- Test: `message_request_test.go`

- [ ] **Step 1: 写失败测试** —— 非法 contentType / 超长 clientMsgId / 空 content 被拒（400）。
- [ ] **Step 2: 运行确认失败。**
- [ ] **Step 3: 实现 binding + 自定义校验。**
- [ ] **Step 4: 测试通过。**
- [ ] **Step 5: commit** `feat(messaging): 发消息输入范围校验`。

### Task A7.2: CORS 中间件

**Files:**
- Create: `internal/pkg/httpx/middleware/cors.go`
- Modify: `internal/app/server/server.go`（装配 CORS，白名单来自 config，与 WS origins 共享）
- Modify: config（`server.cors_allowed_origins []string`）
- Test: `cors_test.go`

- [ ] **Step 1: 写失败测试** —— 白名单 Origin 返回 ACAO 头；非白名单不返回；预检 OPTIONS 204。
- [ ] **Step 2: 运行确认失败。**
- [ ] **Step 3: 实现 CORS（手写最小实现，不引入新依赖）。**
- [ ] **Step 4: 测试通过。**
- [ ] **Step 5: commit** `feat(server): CORS 白名单中间件`。

### Task A7.3: JWT 吊销（logout 黑名单）

**Files:**
- Modify: `internal/pkg/auth/jwt.go`（确保签发含 jti=RegisteredClaims.ID）
- Create: `internal/modules/user/infra/security/token_blacklist.go`（Redis set，键 `jwt:revoked:{jti}`，TTL=token 剩余有效期）
- Modify: `internal/pkg/httpx/middleware/auth.go`（校验后查黑名单，命中 401）—— 经注入的 `Revoker` 端口，nil 时跳过
- Modify: user 模块新增 `POST /auth/logout`（openapi 先行）
- Test: `token_blacklist_test.go` + auth 中间件测试

**Interfaces — Produces:** `type RevocationChecker interface { IsRevoked(ctx, jti string) (bool, error) }`（放 `internal/pkg/auth` 或 middleware 注入）。

- [ ] **Step 1: openapi 增 `/auth/logout`；写失败测试** —— logout 后旧 token 被中间件拒（401）。
- [ ] **Step 2: 运行确认失败。**
- [ ] **Step 3: 实现黑名单 + 中间件查询 + logout handler；Redis 关闭时降级（文档说明吊销不可用）。**
- [ ] **Step 4: `make test && make lint-openapi` 通过。**
- [ ] **Step 5: commit** `feat(auth): JWT 吊销与 logout 端点`。

### Task A7.4: TLS/HTTPS 配置项 + prod 校验

**Files:**
- Modify: config（`server.tls.cert_file`/`key_file`/`enabled`；`server.behind_tls_proxy bool`）+ Validate（prod 下 `tls.enabled || behind_tls_proxy` 二选一，否则校验失败）
- Modify: `internal/app/server/server.go`（tls.enabled 时 `ListenAndServeTLS`）
- Test: config validate 测试 + server tls 分支测试

- [ ] **Step 1: 写失败测试** —— prod 且无 TLS 无代理声明 → Validate 报错；tls.enabled 走 TLS 分支。
- [ ] **Step 2: 运行确认失败。**
- [ ] **Step 3: 实现配置 + Validate + listener 分支。**
- [ ] **Step 4: `make test` 通过。**
- [ ] **Step 5: commit** `feat(server): TLS/HTTPS 配置与生产校验`。

---

## 阶段 A8：DB schema 演进预留 + 收尾

### Task A8.1: 补缺失索引

**Files:**
- Modify: `internal/ent/schema/message.go`（评估 `sender_id` 索引必要性，按"我发的消息"查询需要则加 `idx(conversation_id, sender_id)` 或 `idx(sender_id)`）
- Modify: `internal/ent/schema/conversation.go`（`creator_id` 索引，按"我创建的会话"需要则加）
- Run: `make generate`
- Test: 防漂移（generate 无 diff）+ build

- [ ] **Step 1: 评估查询场景，仅在有真实查询时加索引（避免无用索引）。**
- [ ] **Step 2: 加索引 + `make generate`。**
- [ ] **Step 3: `make build` + `git diff --stat internal/ent` 确认生成同步。**
- [ ] **Step 4: commit** `feat(ent): 补充消息/会话查询索引`。

### Task A8.2: 分区/归档策略文档化 + 逻辑外键标注

**Files:**
- Modify: `internal/ent/schema/message.go`/`conversation.go` 等（字段 Comment 标注逻辑外键关系）
- Create/Modify: `docs/` 增"消息存储分区与归档策略"说明（阶段 A 不建分区，给出 `conversation_id+created_at` 分区迁移路径与冷热分层方向，对接阶段 B 宽列演进）
- Test: `git diff --check`（文档）+ generate 无漂移

- [ ] **Step 1: 标注逻辑外键 Comment + `make generate`。**
- [ ] **Step 2: 写分区/归档文档。**
- [ ] **Step 3: commit** `docs(storage): 消息分区与归档策略及逻辑外键标注`。

### Task A8.3: 全量验证 + 文档收尾

**Files:**
- Modify: `README.md`（IM 模块生产加固概览、单实例部署约束）
- Modify: `docs/`（可观测性指标清单、运维 runbook 要点）
- Run: 全量 `make verify`

- [ ] **Step 1: 更新 README + 运维文档。**
- [ ] **Step 2: `make verify`（受限环境带 GOCACHE/GOMODCACHE）全绿；缺工具说明并跑最强子集。**
- [ ] **Step 3: commit** `docs: 阶段 A 生产加固收尾与运维说明`。

---

## Self-Review 记录

- **Spec 覆盖**：spec §3 的 A1–A8 全部映射到本计划同名里程碑；§5 测试策略逐项落到各任务的 TDD 步骤；§1.1 单实例约束写入 Global Constraints 与 A8.3 文档。
- **占位扫描**：以"任务 + 接口签名 + TDD 步骤 + 验证命令"粒度；实现期每任务先写测试，代码以现有 todo/user/messaging/realtime 模块为范式。
- **类型一致**：`requestctx.WorkspaceID`、`WorkspaceScopeInterceptor`、`Limiter.Middleware(keyFunc)`、`Hub.Register(...) bool`/`GracefulShutdown`、`presence.Refresh`、metrics `Observe*`、`RevocationChecker` 在引入任务中定义，后续任务按签名消费。
- **风险点**：A1.3 Ent 拦截器与 A3.3 改 `imrt.Registry` 接口签名影响多处调用方，实现时同步更新所有实现与 mock。
