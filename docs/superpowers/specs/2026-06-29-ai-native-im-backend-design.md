# AI-Native IM 后端 · 第一期技术设计

> 状态：已确认设计，待生成实施计划
> 日期：2026-06-29
> 范围：基础 IM 后端（Phase 1），为后续 AI 功能扩展预留演进缝
> 技术栈（固定）：Go + gin + Ent ORM + MySQL + Redis(可选) + gorilla/websocket + OTel/Sentry

## 1. 背景与目标

构建一款 AI-Native 的现代 IM 后端，对标 Slack + 微信的即时通信与工作场景对话。第一期**只做基础 IM 能力**，但所有设计需"可实施、可演进"：在数据模型与领域设计中预埋 AI 扩展缝，使后续 AI 功能（智能回复、会议纪要、知识沉淀、Agent）无需重构核心模型。

本设计遵循三套最佳实践 skill：`im-best-practices`（seq 模型、推拉结合、读写扩散、多端一致）、`database-design-best-practices`（MySQL 风格命名/类型/索引/逻辑外键）、`permission-design-best-practices`（RBAC + DataScope + ACL 组合）。

### 第一期范围（锁定）

**核心必含**：1:1/群聊/频道消息收发、每会话 seq 排序与增量同步、离线补拉、会话列表、已读水位、消息撤回。

**增量纳入**：多端同步（per-device 拉取位点）、陈述式状态（在线/输入中）、附件/媒体消息。

**明确不做**：全文检索（`Search` 端口预留，随 AI 语义搜索一起做）、E2EE、读写扩散混合、字段级权限/多维 DataScope scope_rules/ABAC 动态条件（RBAC+ACL 之上的扩展，预留）。

**AI 演进缝（只建模型/接口，不实现 AI）**：领域事件发布到 eventbus、Bot/Agent 作为一等发送者、结构化可扩展消息体（content blocks）、会话/消息级 metadata 扩展位。

## 2. 关键架构决策

### 决策 A：会话存储用「读扩散统一 Timeline」

每会话一条消息 timeline，成员各持 `read_seq` 拉取。1:1/群/频道统一一套模型，无写放大，多端同步天然（所有端共享 timeline 按 seq 拉取）。`unread = conversation.last_seq − member.read_seq` 推导，不双写计数器。

备选（未采纳）：写扩散（每用户收件箱，群聊写放大、多端需回灌）；混合扩散（第一期过度复杂）。

### 决策 B：`msgID` 与 `seq` 严格分工，seq 用 DB 事务行锁分配

| 标识 | 生成 | 负责 | 不负责 |
|---|---|---|---|
| `msgID`（全局） | localid Snowflake int64，时间有序 | 全局唯一、路由、跨会话引用 | 排序、未读、同步游标 |
| `seq`（会话内） | 每会话连续递增，从 1 起 | 会话内排序、未读、增量同步游标、空洞检测 | 全局唯一 |

seq 分配（Phase 1）：发消息事务内 `SELECT conversation ... FOR UPDATE` 锁行，`seq = ++last_seq`，同事务插消息——强一致、零额外依赖。通过 `SeqAllocator` 端口隔离，演进为 Redis `INCR` → 独立 seqsvr。

### 决策 C：WebSocket 鉴权用「短期 ticket」（token 不入 URL）

客户端先用 JWT 调 `POST /ws/ticket` 换一次性短 TTL ticket（存 Redis），再 `wss://…/ws?ticket=…`。复用 user 模块既有 JWT 校验。

### 决策 D：可靠路径与实时路径分离（IM 基线二）

- **可靠路径 = HTTP REST + seq 拉取**：发消息、增量拉取走 HTTP，seq 保证不丢不重不乱序。
- **实时路径 = WS 尽力而为**：只推「有新消息信号 (conv_id, seq)」+ 承载 typing/presence 等 ephemeral。推送丢失由拉取补偿。

### 决策 E：权限用 DB 动态 RBAC + DataScope + ACL 组合

第一期即落库为**动态 RBAC**（角色/权限/关联表可运行时配置），按 `permission-design` skill 的存储定义建模，适配 IM 多工作区场景。

- **RBAC（动态，落库）**：工作区级动作授权。`rbac_roles` / `rbac_permissions` / `rbac_role_permissions` / `rbac_user_roles` 四表；`tenant_id = workspace_id`，`workspace_id=0` 为平台模板角色/权限，建工作区时复制为租户级。默认角色 owner/admin/member/guest 为 seed 数据，权限目录（`workspace.manage`、`member.invite`、`member.remove`、`channel.create`、`channel.archive`、`bot.manage`、`message.send` 等）为 seed。
- **DataScope**：`workspace_id` 多租户硬边界，由 repo 层（Ent 拦截器）统一注入，`current_workspace_id` 只从鉴权上下文解析、严禁从请求参数读取；即使命中 ACL.ALLOW 或超管也不可跨工作区。通用 `scope_rules` 多维范围表第一期不需要（IM 数据范围 = 工作区隔离 + 会话成员关系），预留。
- **资源角色**：`conversation_members.role`（owner/admin/member）作为频道实例级角色，与工作区 RBAC 互补（如公告频道谁可发言、谁可管成员）。
- **ACL（动态，落库）**：`rbac_acl_entries` 处理实例级例外——临时授权某 guest 访问特定私有频道、显式 deny/冻结某用户，带 `effect`/`expires_at`/`reason`/`source`，可审计、可过期、可回收。
- **缓存 + 审计**：`rbac_permission_versions`（版本号 + 精准失效，不只靠 TTL）+ `rbac_permission_change_logs`（授权变更可审计）。
- 鉴权优先级（实例操作）：工作区隔离 → RBAC 动作 → ACL 显式拒绝 → ACL 显式允许 → DataScope/会话成员关系。后端永远是最终裁决方。
- 经 `internal/pkg/authz.PermissionChecker` 端口暴露，workspace 模块实现，注入 messaging/media。

## 3. 模块划分

新增模块于 `internal/modules/`，沿用现有装配范式 `New(client, logger, cfg) → Module` + `Mount(rg, mw...)`：

```
workspace/   工作区 + 成员 + RBAC/DataScope 鉴权
             → 实现 internal/pkg/authz.PermissionChecker 端口
messaging/   会话 + 消息聚合；发送/同步/已读/撤回用例（IM 核心）
realtime/    WS 网关 adapter + 进程内 Hub + 事件分发器(订阅 eventbus 扇出)
             presence/typing 也在此（Redis，可选，不落库）
media/       附件 + 对象存储端口（本地磁盘 dev / S3 prod 可插拔）

共享:
  internal/infra/eventbus   进程内 EventBus（端口+实现，演进 Redis/Kafka）
  internal/infra/objstore   对象存储端口
  internal/pkg/authz        PermissionChecker 端口（workspace 实现，注入 messaging）
```

依赖方向：adapter → application → domain ← infra。跨模块只经 `internal/pkg` 端口或领域事件协作，组合根注入，符合 `make test-architecture` 边界。messaging 通过 `authz.PermissionChecker` 端口依赖 workspace 的鉴权，不直接 import workspace 包。

### domain 聚合分包（包边界 = 聚合边界）

```
workspace/domain/workspace/      Workspace 聚合
workspace/domain/membership/     WorkspaceMember 聚合
workspace/domain/rbac/           Role/Permission/UserRole 聚合(动态 RBAC)
workspace/domain/acl/            AclEntry 聚合(实例级例外授权)
messaging/domain/conversation/   Conversation 聚合(含 ConversationMember)
messaging/domain/message/        Message 聚合
media/domain/attachment/         Attachment 聚合
```

workspace 模块同时承担工作区与 IAM 职责，实现 `authz.PermissionChecker`：内部组合 RBAC（动作）+ ACL（例外）+ DataScope（工作区隔离），对外只暴露 `Check(ctx, subject, action, resource) → bool` 与权限上下文加载。

## 4. 数据模型

MySQL 风格：`BIGINT` 应用层 Snowflake 主键、`TINYINT` 枚举带 COMMENT、逻辑外键 + 索引、无物理外键、审计字段（沿用 `time_mixin`）。新增 Ent schema 于 `internal/ent/schema/`，`make generate` 重新生成。

### 4.1 `workspaces` — 工作区（多租户根）
| 字段 | 类型 | 说明 |
|---|---|---|
| id | BIGINT PK | Snowflake |
| slug | VARCHAR(50) | uk_slug，唯一标识 |
| name | VARCHAR(100) | |
| owner_user_id | BIGINT | 逻辑外键→users.id |
| status | TINYINT | 1-正常 2-停用（默认1） |
| settings | JSON | 扩展配置 |
| created_at/updated_at/deleted_at | | 审计 |

索引：uk_slug、idx_owner、idx_deleted_at

### 4.2 `workspace_members` — 工作区成员关系（角色由 RBAC 表承载）
| 字段 | 类型 | 说明 |
|---|---|---|
| id | BIGINT PK | |
| workspace_id | BIGINT | 逻辑外键 |
| user_id | BIGINT | 逻辑外键 |
| status | TINYINT | 1-正常 2-禁用 |
| joined_at | TIMESTAMP | |
| created_at/updated_at | | 审计 |

索引：uk_ws_user(workspace_id,user_id)、idx_user(user_id)←"我加入的工作区"

> 成员角色不再用枚举列，统一由 `rbac_user_roles`（见 4.8）分配；本表只记录"用户属于工作区"的成员关系与状态。

### 4.3 `conversations` — 会话/频道（读扩散，每会话一行）
| 字段 | 类型 | 说明 |
|---|---|---|
| id | BIGINT PK | convID Snowflake，全局唯一/路由 |
| workspace_id | BIGINT | DataScope 边界 |
| type | TINYINT | 1-单聊DM 2-群聊 3-频道channel |
| visibility | TINYINT | 1-public 2-private（DM 默认2） |
| name | VARCHAR(100) NULL | DM 为空 |
| topic | VARCHAR(255) NULL | |
| creator_id | BIGINT | |
| dm_key | VARCHAR(64) NULL | 单聊去重：workspace_id:minUID#maxUID |
| last_seq | BIGINT | 当前最大 seq；seq 分配锁此行 |
| last_msg_id | BIGINT | 最后消息 msgID |
| last_msg_digest | VARCHAR(255) | 冗余摘要，加速会话列表渲染 |
| last_msg_at | TIMESTAMP | 服务端时间 |
| member_count | INT | |
| status | TINYINT | 1-正常 2-归档 3-解散 |
| metadata | JSON | ← AI 扩展位 |
| created_at/updated_at/deleted_at | | |

索引：uk_dm_key(dm_key)、idx_ws_type(workspace_id,type)、idx_last_msg_at

### 4.4 `conversation_members` — 会话成员（读水位 + 设置）
| 字段 | 类型 | 说明 |
|---|---|---|
| id | BIGINT PK | |
| conversation_id | BIGINT | 逻辑外键 |
| member_type | TINYINT | 1-user 2-bot ← AI 缝 |
| member_id | BIGINT | user_id 或 bot_id |
| role | TINYINT | 1-owner 2-admin 3-member |
| read_seq | BIGINT | 已读水位（账号级，多端对齐，默认0） |
| last_read_at | TIMESTAMP | |
| is_muted | TINYINT | 0/1 |
| is_pinned | TINYINT | 0/1 |
| status | TINYINT | 1-正常 2-已退出 |
| joined_at | TIMESTAMP | |
| created_at/updated_at | | |

索引：uk_conv_member(conversation_id,member_type,member_id)、idx_member(member_type,member_id)←"我的会话列表"

### 4.5 `messages` — 消息 = 会话 Timeline 类型化条目
| 字段 | 类型 | 说明 |
|---|---|---|
| id | BIGINT PK | msgID Snowflake，全局唯一/路由/跨会话引用 |
| conversation_id | BIGINT | 逻辑外键 |
| seq | BIGINT | 每会话连续递增，排序/未读/同步游标/空洞检测 |
| sender_type | TINYINT | 1-user 2-bot 3-system ← AI 缝 |
| sender_id | BIGINT | system 条目可为0 |
| client_msg_id | VARCHAR(64) | 幂等键 |
| content_type | TINYINT | 1-text 2-image 3-file 4-voice 10-recall 11-system 20-card… |
| content | JSON | ← content blocks 结构化消息体（AI 缝） |
| reply_to_msg_id | BIGINT NULL | 引用回复 |
| status | TINYINT | 1-正常 2-已撤回 3-已删除 |
| edited_at | TIMESTAMP NULL | |
| server_time | TIMESTAMP | 服务端权威时间（撤回窗口/排序） |
| metadata | JSON | ← AI 打标/情绪/摘要扩展位 |
| created_at | TIMESTAMP | |

索引：uk_conv_seq(conversation_id,seq)←排序/增量拉取/空洞检测、uk_conv_client(conversation_id,client_msg_id)←幂等、idx_conv_created(conversation_id,created_at)←冷热/分区预留

> 撤回/编辑通知/成员变更系统提示都作为带 seq 的控制条目（`sender_type=system` 或 `content_type=recall`）进同一 timeline，多端按 seq 拉取即对齐（IM 基线五）。

### 4.6 `bots` — Bot/Agent（AI 一等公民，第一期只建模）
| 字段 | 类型 | 说明 |
|---|---|---|
| id | BIGINT PK | |
| workspace_id | BIGINT | |
| name | VARCHAR(100) | |
| type | TINYINT | 1-系统bot 2-用户自建 3-Agent（预留） |
| owner_user_id | BIGINT NULL | |
| config | JSON | ← 未来 AI 配置（模型/提示词/权限范围） |
| status | TINYINT | 1-正常 2-停用 |
| created_at/updated_at | | |

索引：idx_ws(workspace_id)

### 4.7 `attachments` — 附件元数据（消息体只存引用 key）
| 字段 | 类型 | 说明 |
|---|---|---|
| id | BIGINT PK | |
| workspace_id | BIGINT | |
| uploader_id | BIGINT | |
| object_key | VARCHAR(255) | 对象存储 key |
| file_name | VARCHAR(255) | |
| content_type | VARCHAR(100) | MIME |
| size_bytes | BIGINT | |
| status | TINYINT | 1-待提交 2-已提交 |
| metadata | JSON | 宽高/时长等 |
| created_at | TIMESTAMP | |

索引：uk_object_key(object_key)、idx_uploader(uploader_id)

### 4.8 权限表（动态 RBAC + ACL，`tenant_id = workspace_id`）

所有权限表带 `workspace_id`（=租户）；`workspace_id=0` 为平台模板，建工作区时复制为租户级；运行时查询强制 `workspace_id IN (0, current)` 过滤。

**`rbac_permissions` — 权限定义（动作目录）**
```
id BIGINT PK | workspace_id BIGINT (0=平台通用)
code VARCHAR(128)  权限编码 如 message.send / channel.create / member.invite
name VARCHAR(100)  | resource_type VARCHAR(64) | action_key VARCHAR(64)
status TINYINT 1-正常 2-禁用 | created_at/updated_at
uk_ws_code(workspace_id,code), idx_resource_action(workspace_id,resource_type,action_key)
```

**`rbac_roles` — 角色**
```
id BIGINT PK | workspace_id BIGINT (0=平台模板)
code VARCHAR(64) 如 owner/admin/member/guest | name VARCHAR(100)
role_type TINYINT 1-系统 2-工作区 | status TINYINT 1-正常 2-禁用 | 审计
uk_ws_code(workspace_id,code), idx_ws_status(workspace_id,status)
```

**`rbac_role_permissions` — 角色↔权限**
```
workspace_id BIGINT | role_id BIGINT | permission_id BIGINT | created_at
PK(role_id,permission_id), idx_permission(permission_id), idx_ws_role(workspace_id,role_id)
```

**`rbac_user_roles` — 用户↔角色（工作区内授权）**
```
workspace_id BIGINT | user_id BIGINT | role_id BIGINT
source_type TINYINT 1-手工 2-默认 4-临时 | effective_end_at TIMESTAMP NULL 过期 | created_at
PK(workspace_id,user_id,role_id), idx_role(role_id), idx_ws_user_expire(workspace_id,user_id,effective_end_at)
```

**`rbac_acl_entries` — 实例级例外授权（私有频道特批/冻结）**
```
id BIGINT PK | workspace_id BIGINT
subject_type TINYINT 1-用户 2-角色 | subject_id BIGINT
resource_type VARCHAR(64) 如 conversation | resource_id VARCHAR(128)
action_code VARCHAR(128) | effect TINYINT 1-允许 2-拒绝
reason VARCHAR(255) | source_type TINYINT | expires_at TIMESTAMP NULL | created_at/created_by
idx_subject(workspace_id,subject_type,subject_id,action_code,effect,expires_at),
idx_resource(workspace_id,resource_type,resource_id,action_code,effect,expires_at),
idx_expires(workspace_id,expires_at) ← 定时清理过期授权
```
> ACL 必须有 `effect/expires_at/reason/source`，否则退化为永久黑箱。

**`rbac_permission_versions` — 缓存版本（精准失效）**
```
workspace_id BIGINT | subject_type TINYINT 1-用户 2-角色 | subject_id BIGINT
version BIGINT | updated_at
PK(workspace_id,subject_type,subject_id) | 缓存键 perm:user:{ws}:{uid}:v{version}
```

**`rbac_permission_change_logs` — 授权变更审计**
```
id BIGINT PK | workspace_id BIGINT | change_type TINYINT | target_type TINYINT | target_id BIGINT
operator_id BIGINT | request_id VARCHAR(64) | before_json JSON | after_json JSON | reason | created_at
idx_target(workspace_id,target_type,target_id,created_at), idx_operator(operator_id,created_at)
```

> 鉴权链路：工作区隔离(硬边界) → RBAC 动作 → ACL 显式拒绝 → ACL 显式允许 → DataScope/会话成员。`current_workspace_id` 只从鉴权上下文取，repo 层统一注入 `workspace_id` 过滤。平台模板角色/权限通过 `cmd/datainit` seed。

### 4.9 放 Redis / 不落库
- **presence**：`presence:{workspace}:{user}` TTL 心跳续期。
- **typing**：`typing:{conv}` 短 TTL，广播不持久。
- **WS ticket**：`ws:ticket:{token}` 一次性短 TTL。
- **seq 加速**（演进期）：`conv:{id}:seq`，第一期不用（走 DB 行锁）。
- **权限缓存**：`perm:user:{ws}:{uid}:v{version}`，版本号变更即失效，不只靠 TTL。

## 5. 关键流程

### 5.1 发送消息（可靠路径）
```
Client POST /workspaces/{ws}/conversations/{cid}/messages
       {client_msg_id, content_type, content, reply_to?}
Server: 1. authz: 是否会话成员 + 是否可发言(role/频道设置)
        2. 幂等: 查 uk_conv_client(cid, client_msg_id) 命中→返回原结果
        3. TX{ SELECT conversation FOR UPDATE; seq=++last_seq;
               msgID=snowflake; INSERT message;
               UPDATE conversation last_seq/last_msg_*/last_msg_at }
        4. publish MessageCreated 事件 → eventbus
        5. 返回 {msg_id, seq, server_time}
Dispatcher(订阅事件): 向所有在线成员连接推 signal{cid, seq}
Receivers: 按本地 last_seq 拉 GET …/messages?after_seq=
```
ACK（第5步）仅表示"服务端已持久化"，不等于对方已收到。

### 5.2 增量同步 / 离线补拉
`GET /conversations/{cid}/messages?after_seq=X&limit=N` → (X, max] 区间消息。客户端校验 seq 连续，发现空洞立即补拉。推送只触发信号，拉取保证最终一致。

### 5.3 会话列表同步
`GET /conversations?changed_after=<ts/cursor>` → 各会话 `last_seq/read_seq/unread/last_msg_digest`，按 `last_msg_at` 排序。

### 5.4 已读上报
`POST /conversations/{cid}/read {read_seq}` → `read_seq = max(old, new)` 单调更新 → 推「已读更新」给本人其他端 +（按设置）给其他成员发已读回执。

### 5.5 撤回
`POST /messages/{mid}/recall` → 校验 `now − server_time < 撤回窗口`（服务端时间）+ 权限 → TX 内插一条 `content_type=recall` 控制条目（占新 seq）并标记目标 `status=已撤回` → 推 signal。

### 5.6 presence/typing
WS ephemeral 帧 → 写 Redis(TTL) → 广播会话成员，不落库、不占 seq。

### 5.7 附件
`POST /attachments:presign {file_name,content_type,size}` → `{upload_url, object_key, attachment_id}` → 客户端直传对象存储 → 发消息时 content 引用 `attachment_id` → 服务端校验并置 `status=已提交`。

## 6. WebSocket 协议帧（第一期 JSON，Protobuf 预留）
```
上行: {t:"auth", ticket}  {t:"ping"}  {t:"typing", cid}
      {t:"read", cid, read_seq}
下行: {t:"auth_ok"|"auth_err"}  {t:"pong"}
      {t:"signal", cid, seq}         ← 有新消息(仅信号)
      {t:"presence", uid, online}    ← 在线状态变更
      {t:"typing", cid, uid}
      {t:"read", cid, uid, read_seq} ← 已读水位更新
```
自适应心跳 + 断线重连：重连后客户端用 HTTP 按各会话 `last_seq` 补拉，不依赖 WS 补消息。

## 7. 演进路径

| 维度 | Phase 1 | 演进 |
|---|---|---|
| seq 分配 | `SeqAllocator`=DB 行锁 | Redis INCR → seqsvr |
| 消息扇出 | `MessageFanout`=进程内 channel | Redis pub/sub → Kafka |
| 连接管理 | 进程内 Hub | 独立 Gateway 服务 |
| 事件总线 | 进程内 EventBus | Redis Streams → Kafka |
| 消息存储 | MySQL `uk(conv,seq)` | 按 conv+时间分区 → 宽列(ScyllaDB) |
| 全文/语义搜索 | `Search` 端口(空实现) | ES / 向量库(随 AI) |
| 权限 | DB 动态 RBAC + ACL + 版本化缓存 | 字段级权限/scope_rules 多维范围/ABAC 动态条件 |
| AI | 订阅 `MessageCreated` 事件 | 总结/智能回复/Agent(以 bot 身份发消息) |

所有演进点都以端口（interface）隔离，组合根注入，替换实现不触碰用例与领域层。

## 8. 测试策略

- **domain 单测**：seq 单调、`unread=last_seq−read_seq`、撤回窗口（服务端时间）、dm_key 去重、content blocks 校验。
- **application 单测**：fake repo/seqalloc/eventbus，验证幂等（重复 client_msg_id 返回同结果）、事务边界、authz 调用。
- **集成测试**（testcontainers MySQL）：seq 分配并发正确性（并发发消息无重复/空洞 seq）、幂等唯一索引、会话列表查询。
- **WS 集成测试**：ticket 鉴权、signal 推送、断线重连后 HTTP 补拉一致。
- **架构测试**：`make test-architecture` 守护分层与跨模块端口边界。
- 完成前 `make verify`。

## 9. API 契约

新增端点先写入 `api/openapi.yaml`，再实现 DTO/Handler/Router（OpenAPI-first）：

```
POST   /workspaces                              创建工作区
GET    /workspaces                              我的工作区列表
POST   /workspaces/{ws}/members                 加入/邀请成员
POST   /workspaces/{ws}/conversations           创建会话/频道(DM 自动 dm_key 去重)
GET    /workspaces/{ws}/conversations           我的会话列表(?changed_after)
POST   /workspaces/{ws}/conversations/{cid}/messages   发送消息
GET    /workspaces/{ws}/conversations/{cid}/messages    增量拉取(?after_seq&limit)
POST   /conversations/{cid}/read                已读上报
POST   /messages/{mid}/recall                   撤回
POST   /attachments:presign                     预签名上传
POST   /ws/ticket                               换 WS 鉴权 ticket
GET    /ws                                       WebSocket 升级(?ticket)
```

## 10. 实施里程碑（供 writing-plans 拆解）

1. **M1 工作区与权限**：workspace 模块、动态 RBAC（roles/permissions/role_permissions/user_roles）+ ACL + 版本化缓存、`PermissionChecker` 端口、DataScope 工作区隔离（repo 层注入）、平台模板角色/权限 seed（`cmd/datainit`）、Ent schema + 迁移。
2. **M2 会话与成员**：conversation 聚合、DM 去重、会话列表查询、成员管理。
3. **M3 消息核心**：message 聚合、`SeqAllocator`(DB 行锁)、幂等、发送/增量拉取、撤回；in-process EventBus + `MessageCreated`。
4. **M4 实时网关**：WS ticket 鉴权、进程内 Hub、事件分发器(signal 扇出)、心跳重连。
5. **M5 已读与多端**：read_seq 上报与对齐、已读回执、多端 signal 路由。
6. **M6 presence/typing**：Redis 在线状态与输入中广播。
7. **M7 附件**：objstore 端口(本地/S3)、预签名、attachment 提交。
8. 横切：OpenAPI 契约、单测/集成测试、`make verify`、可观测性(metrics/trace)。

每个里程碑独立可测、可演示，遵循"行为变更先写测试 / API 变更先改 openapi.yaml"。
