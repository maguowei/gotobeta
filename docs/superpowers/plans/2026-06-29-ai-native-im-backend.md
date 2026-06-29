# AI-Native IM 后端实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: 用 superpowers:subagent-driven-development 或 superpowers:executing-plans 逐任务实施。步骤用 `- [ ]` 复选框跟踪。

**Goal:** 在现有 Go DDD 脚手架上实现基础 IM 后端（多工作区、动态 RBAC、会话/消息、WS 实时、已读多端、presence/typing、S3 附件），并为 AI 扩展预留缝。

**Architecture:** 单 Go 进程，沿用 `internal/modules/<name>/{domain,application,infra,adapter}` 分层。新增 workspace / messaging / realtime / media 四模块；新增共享 `internal/pkg/event`（事件端口）、`internal/pkg/authz`（鉴权端口）、`internal/infra/eventbus`（进程内事件总线）、`internal/infra/objstore`（S3 对象存储）。读扩散统一 timeline + 每会话 seq + 推拉结合。

**Tech Stack:** Go 1.25、gin、Ent ORM、MySQL、Redis（go-redis，可选）、gorilla/websocket、minio-go（S3 兼容）、OTel/Sentry。

**Spec:** `docs/superpowers/specs/2026-06-29-ai-native-im-backend-design.md`

## 实施状态（2026-06-29 完成）

第一期 F/M1–M7 + 收尾 Z1 全部实现并提交，`make verify` 全门禁通过（generate 无漂移、lint 0 issue、总覆盖率 70.8% ≥ 70%、密钥/漏洞/模块/架构/构建/集成编译均绿）。

- **F 基础**：`internal/pkg/event` 事件端口、`internal/infra/eventbus` 进程内总线、config 扩展 IM/objstore 段。
- **M1 workspace**：多工作区 + DB 动态 RBAC/ACL，平台模板复制；handler + 路由 + datainit seed。
- **M2 messaging**：会话（单聊 dm_key 去重/群/频道）、按会话 seq 分配、发送/撤回/已读水位、读扩散时间线。
- **M3 realtime**：WS 网关（ticket 鉴权）、signal/typing/read/presence 帧、push-pull、多端对齐。
- **M4 media**：S3 兼容对象存储预签名上传 + 提交确认。
- **AI 缝**：领域事件外发、Bot/Agent 一等发送者、结构化 content block、metadata 列。
- **Z1 收尾**：`make verify` 全绿；`docs/events/README.md` 增 IM 进程内事件契约、README 增 IM 模块概览；集成测试见 `internal/integration/`（seq 并发/RBAC 解析/单聊去重，`make test-integration` 需 Docker，本地已通过）。

下方逐任务复选框为原始计划记录，保留以备追溯；实际实现以 git 提交历史与 `make verify` 结果为准。

## Global Constraints

- 模块路径 `github.com/maguowei/gotobeta`；分层依赖：adapter → application → domain ← infra；由 `make test-architecture` 校验。
- domain 层零外部依赖（不引入 gin/ent/viper/slog/net-http/database-sql）；按聚合分包，聚合包不互相 import。
- 跨模块禁止直接 import：只经 `internal/pkg` 端口或领域事件协作。
- 第三方 SDK 唯一归口：go-redis→`internal/infra/cache`、sarama→`internal/infra/eventbus`、minio-go→`internal/infra/objstore`、jwt→`internal/pkg/auth`。
- 实体 PK 用 `field.Int64("biz_id").Unique().Immutable()` + `TimeMixin`；ID 由 `localid.New()`（int64 Snowflake）生成。
- 应用层 CQRS 命名：`<动词><名词>Command/Query`、`<名词>Result`；query 包不得 import `internal/pkg/event`。
- HTTP request/response 不得 import domain，只从 command/query/result 映射。
- 配置只在组合根读取，构造函数注入；写操作幂等。
- 行为变更先写测试；HTTP 变更先改 `api/openapi.yaml`；完成前跑 `make verify`（受限环境 `GOCACHE=/tmp/go-build-cache GOMODCACHE=/tmp/go-mod-cache`）。
- 提交用中文 Conventional Commits。

## 里程碑与验证

| 阶段 | 交付 | 验证命令 |
|---|---|---|
| F 基础 | event 端口 + 进程内 eventbus + config 扩展 | `make test` |
| M1 | workspace + 动态 RBAC + ACL + authz 端口 | `make generate && make test && make test-architecture` |
| M2 | conversation + member + DM 去重 + seq 端口 | 同上 |
| M3 | message + 发送/拉取/撤回 + 事件发布 + bot | 同上 + 并发 seq 集成测试 |
| M4 | WS ticket + 网关 Hub + 事件分发 signal | `make build` + WS 集成测试 |
| M5 | read_seq 上报 + 多端/成员推送 + 未读 | `make test` |
| M6 | presence/typing（Redis + WS） | `make test` |
| M7 | objstore(S3) + 附件 presign/commit | `make verify` |

每个里程碑结束：`make verify` 子集通过即 commit。整体完成跑完整 `make verify`。

---

## 阶段 F：基础设施

### Task F1: 领域事件端口 `internal/pkg/event`

**Files:** Create `internal/pkg/event/event.go`, `internal/pkg/event/event_test.go`

**Interfaces — Produces:**
- `type Event interface { Name() string; OccurredAt() time.Time }`
- `type Publisher interface { Publish(ctx context.Context, events ...Event) error }`
- `type Handler func(ctx context.Context, e Event) error`

- [ ] 写测试：定义一个 fake event 实现 `Event`，断言接口可被实现、`Name()/OccurredAt()` 可读。
- [ ] 实现 `event.go`：仅接口与一个 `BaseEvent{ name string; occurredAt time.Time }` 帮助结构（提供 `NewBaseEvent(name)`）。
- [ ] `make test ./internal/pkg/event/...` 通过；commit `feat(event): 新增领域事件端口`。

### Task F2: 进程内事件总线 `internal/infra/eventbus`

**Files:** Create `internal/infra/eventbus/inproc.go`, `inproc_test.go`

**Interfaces — Consumes:** `event.Event/Handler`. **Produces:** `eventbus.NewInProc(logger) *InProc`；`(*InProc) Subscribe(name string, h event.Handler)`；`(*InProc) Publish(ctx, ...event.Event) error`（实现 `event.Publisher`）。同步派发，handler 出错只记日志不阻断（推送是尽力而为）。

- [ ] 测试：Subscribe 后 Publish，handler 被调用；handler 返回 error 不影响 Publish 返回 nil；未订阅的事件 Publish 不 panic。
- [ ] 实现：`map[string][]Handler` + RWMutex；Publish 遍历调用，error 走 `logger.WarnContext`。
- [ ] 架构测试豁免：eventbus 在 `internal/infra`，不得 import modules（已满足）。
- [ ] `make test ./internal/infra/eventbus/...`；commit `feat(eventbus): 进程内事件总线实现`。

### Task F3: config 扩展 `internal/infra/config`

**Files:** Modify `internal/infra/config/*.go`、`configs/config.example.yaml`

新增 typed 配置段（默认值安全）：
- `im.recall_window`（默认 `2m`）、`im.presence_ttl`（默认 `30s`）、`im.ws_ticket_ttl`（默认 `30s`）、`im.message_page_size`（默认 100）。
- `objstore`：`endpoint/region/bucket/access_key/secret_key/use_ssl/public_base_url/presign_ttl`（默认空，dev 指向 MinIO）。

- [ ] 测试：加载 example 配置后字段有预期默认值。
- [ ] 实现 struct + viper 默认值 + 注入；example yaml 补充段。
- [ ] `make test ./internal/infra/config/...`；commit `feat(config): 新增 im 与 objstore 配置段`。

---

## 阶段 M1：工作区 + 动态 RBAC + ACL

> 模块目录 `internal/modules/workspace/`。聚合：workspace、membership、rbac、acl。实现 `internal/pkg/authz.PermissionChecker`。

### Task M1.1: authz 端口 `internal/pkg/authz`

**Files:** Create `internal/pkg/authz/authz.go`, `authz_test.go`

**Produces:**
- `type Subject struct { UserID int64 }`
- `type Request struct { WorkspaceID int64; Subject Subject; Action string; ResourceType string; ResourceID string }`
- `type Checker interface { Check(ctx context.Context, req Request) error }`（无权限返回 `apperr.Forbidden`）
- 错误：复用 `apperr`。

- [ ] 测试：定义 fake Checker，断言可注入与调用。
- [ ] 实现端口（纯接口 + 值类型）。
- [ ] commit `feat(authz): 新增鉴权端口`。

### Task M1.2: Ent schema（9 张表）

**Files:** Create `internal/ent/schema/{workspace,workspace_member,rbac_permission,rbac_role,rbac_role_permission,rbac_user_role,rbac_acl_entry,rbac_permission_version,rbac_permission_change_log}.go`

按 spec 4.1/4.2/4.8 字段；`biz_id` int64 unique + TimeMixin（关联表用复合唯一索引，无需 biz_id 的可用自增 id + 唯一索引）。枚举用 `field.Int8` + 注释。JSON 列用 `field.JSON("settings", map[string]any{})` 或 `field.String().Optional()` 存原文。

- [ ] 逐个写 schema（字段 + `Indexes()` 唯一/普通索引）。
- [ ] `make generate`，确认 `internal/ent/` 生成新类型且 `make build` 通过。
- [ ] commit `feat(workspace): 新增工作区与 RBAC Ent schema`。

### Task M1.3: domain/workspace 聚合

**Files:** `internal/modules/workspace/domain/workspace/{workspace.go,status.go,repository.go,errors.go,doc.go}` + tests

- 实体字段：id、slug、name、ownerUserID、status、settings、时间戳；`New(id, slug, name, owner)` 校验 slug 非空且格式（小写字母数字连字符）；`UnmarshalFromDB(...)`。
- Repository：`Create/FindByID/FindBySlug/Save/ListByMemberUser`。

- [ ] 测试：slug 校验、状态流转。
- [ ] 实现聚合 + 仓储接口 + errors（`ErrNotFound/ErrSlugTaken`）。
- [ ] commit。

### Task M1.4: domain/membership、domain/rbac、domain/acl 聚合

**Files:** `internal/modules/workspace/domain/{membership,rbac,acl}/...`

- membership：`WorkspaceMember{workspaceID,userID,status,joinedAt}` + Repository（`Add/FindByWorkspaceUser/ListByUser/ListByWorkspace`）。
- rbac：`Role{id,workspaceID,code,name,roleType,status}`、`Permission{id,workspaceID,code,resourceType,actionKey}`、`UserRole{workspaceID,userID,roleID,effectiveEnd}`；Repository 接口：`ResolveUserActions(ctx, ws, userID) (map[string]struct{}, error)`（联结 user_roles→role_permissions→permissions 得动作集合）、角色/权限 CRUD、`AssignRole/RevokeRole`。
- acl：`AclEntry{...effect,expiresAt,reason,source}` + Repository：`FindDecisive(ctx, ws, subject, resourceType, resourceID, action) (effect, found)`、`Grant/Revoke`。

- [ ] 各聚合测试（动作集合解析逻辑、ACL effect 优先级在 domain 用纯函数表达 `Decide(rbacAllowed bool, acl *AclEntry) error`）。
- [ ] 实现；commit。

### Task M1.5: infra/persistence 仓储实现

**Files:** `internal/modules/workspace/infra/persistence/*.go`（workspace_repository、membership_repository、rbac_repository、acl_repository）

Ent 实现，`entdb.ClientFromCtx`。`ResolveUserActions` 用 Ent 查询联结（或多查询在 repo 内组装）。

- [ ] 集成测试（testcontainers MySQL，build tag `integration`）：建工作区→assign role→ResolveUserActions 返回正确动作集。
- [ ] 实现；commit。

### Task M1.6: PermissionChecker 实现（组合 RBAC+ACL+缓存）

**Files:** `internal/modules/workspace/infra/authz/checker.go` + test

实现 `authz.Checker`：
1. 平台超管/工作区 owner 短路（owner 全允许，除非 ACL 显式 deny）。
2. `ResolveUserActions` 得 RBAC 动作集；
3. 若带 ResourceID，查 ACL `FindDecisive`：deny→Forbidden；allow→放行；
4. RBAC 命中 action→放行，否则 Forbidden。
5. 可选 Redis 缓存动作集（version 失效），Redis 关闭时直查 DB。

- [ ] 测试（fake repos）：owner 放行、ACL deny 覆盖 RBAC allow、ACL allow 放行无 RBAC、普通成员越权 Forbidden。
- [ ] 实现；commit。

### Task M1.7: application + adapter + 平台模板 seed + 装配 + openapi

**Files:** `internal/modules/workspace/application/{command,query,result,service}/...`、`adapter/http/{handler,request,response,router}/...`、`internal/modules/workspace/module.go`、`internal/app/datainit/...`（seed 平台角色/权限）、`api/openapi.yaml`、`internal/app/server/server.go`（Mount）

用例：CreateWorkspace（建工作区 + 建租户级角色（复制平台模板）+ 把 owner 加入并 assign owner 角色，全在一个 tx）、ListMyWorkspaces、InviteMember、AssignRole、ListRoles。

- [ ] openapi 先行：`/workspaces`、`/workspaces/{ws}/members`、`/workspaces/{ws}/roles`、`/workspaces/{ws}/members/{uid}/roles`。
- [ ] 服务 + handler + 路由（复用 `user.AuthMiddleware()`）+ datainit seed。
- [ ] server.go 装配 `workspace.New(...)` 并 Mount；把 checker 暴露给后续模块（组合根持有）。
- [ ] `make generate && make test && make test-architecture && make build`；commit `feat(workspace): 工作区与动态 RBAC 用例与接口`。

---

## 阶段 M2：会话与成员

> 模块 `internal/modules/messaging/`，聚合先做 conversation（含 ConversationMember）。seq 端口也在此。

### Task M2.1: seq 端口 + Ent schema（conversation、conversation_member）

**Files:** `internal/modules/messaging/application/port/seq.go`（`SeqAllocator interface { Next(ctx, convID int64) (int64, error) }`）、`internal/ent/schema/{conversation,conversation_member}.go`

按 spec 4.3/4.4；`conversations.last_seq` int64、`dm_key` 唯一可空、`uk_conv_member`。

- [ ] schema + `make generate` + build。
- [ ] commit `feat(messaging): 会话 Ent schema 与 seq 端口`。

### Task M2.2: domain/conversation 聚合 + 仓储

**Files:** `internal/modules/messaging/domain/conversation/{conversation.go,member.go,type.go,repository.go,errors.go}` + tests

- `Conversation{id,workspaceID,type,visibility,name,topic,creatorID,dmKey,lastSeq,...}`；工厂 `NewDM/NewGroup/NewChannel`；`DMKey(ws, a, b)` 确定性（min#max）。
- `ConversationMember{convID,memberType,memberID,role,readSeq,...}`；`Unread(lastSeq) = lastSeq-readSeq`。
- Repository：`Create/FindByID/FindByDMKey/Save/AddMember/FindMember/ListMembers/ListByMember/BumpLastSeqTx`。

- [ ] 测试：DMKey 对称性、unread 计算、成员角色。
- [ ] 实现 + 仓储实现（infra/persistence）。
- [ ] commit。

### Task M2.3: seq DB 行锁实现 + application + adapter + openapi

**Files:** `internal/modules/messaging/infra/seqalloc/db_allocator.go`（`SELECT ... FOR UPDATE` 锁 conversation 行 `++last_seq`，必须在 tx 内调用，用 `entdb.ClientFromCtx`）、application（CreateConversation 含 DM 去重、ListConversations、AddMember/RemoveMember）、adapter、openapi、module.go

- [ ] 集成测试：并发对同一会话分配 seq 无重复（用 errgroup 并发 N 次，断言得到 1..N 连续集合）。
- [ ] openapi + handler + 路由 + 装配（注入 M1 checker：建会话/加成员前 `checker.Check`）。
- [ ] `make generate && make test && make test-architecture && make build`；commit `feat(messaging): 会话用例与 seq 分配`。

---

## 阶段 M3：消息核心

### Task M3.1: Ent schema（message、bot）+ 领域事件类型

**Files:** `internal/ent/schema/{message,bot}.go`、`internal/modules/messaging/domain/message/events.go`（`MessageCreated` 实现 `event.Event`）

message 按 spec 4.5：`uk_conv_seq`、`uk_conv_client`、content JSON、server_time。

- [ ] schema + generate + build；事件类型测试。
- [ ] commit。

### Task M3.2: domain/message 聚合 + 仓储

**Files:** `internal/modules/messaging/domain/message/{message.go,content.go,sender.go,status.go,repository.go,errors.go}` + tests

- content blocks：`Content struct { Type int; Payload json.RawMessage }`（domain 不依赖外部，用标准库 encoding/json）。
- `New(msgID, convID, seq, sender, clientMsgID, content, serverTime)`；`Recall(now, window)` 校验 `now-serverTime<window` 否则 `ErrRecallWindowExpired`。
- Repository：`Insert/FindByID/FindByClientMsgID/ListAfterSeq(convID, afterSeq, limit)/MarkRecalled`。

- [ ] 测试：撤回窗口、幂等查询、内容校验。
- [ ] 实现 + 仓储实现。
- [ ] commit。

### Task M3.3: SendMessage 用例（tx + seq + 幂等 + 事件）

**Files:** `internal/modules/messaging/application/service/message_commands.go` + tests

流程（spec 5.1）：authz Check `message.send`（resource=conversation:cid）→ 查 `FindByClientMsgID` 命中返回原结果 → `txRunner.RunInTx`{ `seqAllocator.Next` → `New` → `Insert` → `BumpLastSeq` } → `publisher.Publish(MessageCreated)` → 返回 result。

- [ ] 测试（fake repos/seq/publisher/checker）：正常发送分配 seq、重复 clientMsgID 幂等返回、越权 Forbidden、事件被发布。
- [ ] 实现；commit。

### Task M3.4: PullMessages / RecallMessage 用例 + adapter + openapi + 装配

**Files:** application query（ListAfterSeq）、recall command、adapter、openapi、module.go、server.go 装配（注入 eventbus publisher）

- [ ] openapi：`POST .../messages`、`GET .../messages?after_seq=&limit=`、`POST /messages/{mid}/recall`。
- [ ] handler + 路由 + 装配。
- [ ] 集成测试：发 3 条→拉取 after_seq=0 得 3 条且 seq 连续；撤回后状态变更并产生控制条目。
- [ ] `make generate && make test && make test-architecture && make build`；commit `feat(messaging): 消息发送/拉取/撤回与事件发布`。

---

## 阶段 M4：实时 WS 网关

> 模块 `internal/modules/realtime/`。gorilla/websocket 直接在该模块 adapter 使用（非 SDK 归口约束对象）。

### Task M4.1: WS ticket 签发（Redis）

**Files:** `internal/modules/realtime/infra/ticket/redis_ticket.go`（用 `internal/infra/cache` 的 redis client；Redis 关闭时退化为进程内带 TTL 的 map）、application、adapter `POST /ws/ticket`

- [ ] 测试：签发→校验一次性消费→二次校验失败/过期失败。
- [ ] openapi + 实现 + 路由（鉴权后）；commit。

### Task M4.2: 进程内 Hub（连接注册表）

**Files:** `internal/modules/realtime/infra/hub/hub.go` + test

`Hub`：`Register(userID int64, conn *Conn)`、`Unregister`、`Push(userID int64, frame []byte)`、`Broadcast(userIDs []int64, frame)`。线程安全 `map[int64]map[*Conn]struct{}`。

- [ ] 测试：注册多连接、Push 到该用户全部连接、Unregister 清理。
- [ ] 实现；commit。

### Task M4.3: WS 升级 handler + 心跳 + 帧编解码

**Files:** `internal/modules/realtime/adapter/ws/{handler.go,frame.go,conn.go}`

- `GET /ws?ticket=`：校验 ticket→取 userID→Upgrade→注册到 Hub→读循环（auth/ping/typing/read 上行帧）+ 写循环 + ping/pong 心跳超时。
- 帧 JSON 编解码（spec 6）。

- [ ] 测试：用 `httptest.Server` + gorilla dialer，ticket 鉴权成功握手、ping→pong、非法 ticket 拒绝。
- [ ] 实现；commit。

### Task M4.4: 事件分发器（订阅 eventbus → 推 signal）

**Files:** `internal/modules/realtime/application/dispatcher.go`、module.go、server.go 装配

- 订阅 `MessageCreated`：取会话成员（经 messaging 的查询端口或一个跨模块 `ConversationMemberLookup` 端口放 `internal/pkg`）→ 对在线成员 `Hub.Push(signal{cid,seq})`。
- 跨模块协作：在 `internal/pkg/imevent` 定义共享事件名与 payload，或 realtime 通过注入的查询端口读成员（端口定义在 `internal/pkg`）。

- [ ] 测试：发布 MessageCreated→在线成员收到 signal 帧。
- [ ] 实现 + 装配（eventbus.Subscribe 在组合根接线）。
- [ ] `make build` + WS 集成测试；commit `feat(realtime): WS 网关与事件分发`。

---

## 阶段 M5：已读与多端

### Task M5.1: ReportRead 用例 + 推送

**Files:** messaging application（`ReportReadCommand`：`read_seq=max(old,new)` 单调更新 conversation_member）、realtime 推送 read 帧给本人其他端 + 会话其他成员

- [ ] 测试：read_seq 单调（旧值不回退）；未读 = last_seq-read_seq。
- [ ] openapi `POST /conversations/{cid}/read` + handler + 经事件/端口触发 realtime 推送。
- [ ] commit `feat(messaging): 已读水位上报与多端对齐`。

### Task M5.2: 会话列表未读聚合

**Files:** messaging query（ListConversations 返回 last_seq/read_seq/unread/last_msg_digest，按 last_msg_at 排序）

- [ ] 测试：列表含正确 unread。
- [ ] 实现；commit。

---

## 阶段 M6：presence / typing

### Task M6.1: presence（Redis TTL + 上线/下线广播）

**Files:** `internal/modules/realtime/infra/presence/redis_presence.go`（`internal/infra/cache`）、Hub 注册/注销时写 presence + 广播 `presence` 帧给相关会话成员

- [ ] 测试：上线写 key、TTL 续期、下线删除、广播帧。
- [ ] 实现；commit。

### Task M6.2: typing 广播

**Files:** WS 上行 `typing` 帧 → 广播给会话其他成员（不落库，可选 Redis 短 TTL 去抖）

- [ ] 测试：typing 帧广播到会话其他在线成员，不回送自己。
- [ ] 实现；commit `feat(realtime): presence 与 typing`。

---

## 阶段 M7：附件（S3 兼容）

### Task M7.1: objstore 端口 + minio-go 实现

**Files:** `internal/infra/objstore/{objstore.go,minio.go}` + test；`go get github.com/minio/minio-go/v7`

- 端口：`Presigner interface { PresignPut(ctx, key string, ttl time.Duration) (url string, err error); PublicURL(key string) string }`。
- minio 实现（SDK 归口于此包）；架构测试 sdkGateways 增加 minio-go→objstore 一条（修改 `internal/architecture/dependency_test.go`）。

- [ ] 测试（minio testcontainer 或跳过的集成测试）：presign 返回可用 URL。
- [ ] 实现 + 架构测试归口条目；commit `feat(objstore): S3 兼容对象存储归口`。

### Task M7.2: attachment 模块 + presign/commit + 消息引用

**Files:** `internal/modules/media/{domain/attachment,application,infra/persistence,adapter,module.go}`、Ent schema `attachment`、openapi、server.go 装配

- `POST /attachments:presign`→建 pending attachment + 返回 presigned PUT；`POST /attachments/{id}:commit` 或发消息时校验并置 committed；消息 content block type=image/file 引用 attachment_id/object_key。

- [ ] openapi + 集成测试：presign→commit→发带附件消息引用成功。
- [ ] 实现 + 装配；commit `feat(media): 附件预签名上传与消息引用`。

---

## 收尾

### Task Z1: 全量验证 + 文档

- [x] `make verify`（受限环境带 GOCACHE/GOMODCACHE）全绿；缺工具则说明并跑最强子集。
- [x] 更新 `docs/` 事件/可观测性说明；README 增 IM 模块概览。
- [x] commit `docs: IM 模块说明`；分支汇总。

## Self-Review 记录

- 覆盖性：spec 各节（架构决策 A–E、数据模型 4.1–4.10、流程 5.1–5.7、WS 6、演进 7、测试 8、API 9、里程碑 10）均映射到 F/M1–M7/Z 任务。
- 占位扫描：本计划用"任务+关键接口+验证"粒度；实现期每任务先写测试（TDD），代码以现有 todo/user 模块为范式直接套用。
- 类型一致：authz.Checker、event.Publisher、SeqAllocator、Hub、Presigner 接口签名在引入任务中定义，后续任务按签名消费。
