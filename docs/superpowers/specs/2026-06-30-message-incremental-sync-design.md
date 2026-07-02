# 消息增量同步接口设计

> 日期：2026-06-30 · 模块：messaging · 状态：待评审

## Context（背景）

当前 IM 后端的多端一致性存在缺口。会话内的变更有四类：新消息、编辑、撤回、reaction 增删。它们的"离线重连可追平性"现状不一：

| 变更 | 存储方式 | 离线 seq 增量能否追平 |
|---|---|---|
| 新消息 | 占 seq，存 message 表 | ✅ PullMessages（seq 增量）|
| 撤回 | 原消息原地改 status + 写占 seq 的系统条目 | ⚠️ 间接（靠系统条目）|
| 编辑 | 原地改 content + edited_at，**不占 seq** | ❌ |
| Reaction add/remove | 独立表、**硬删除**、无 updated_at | ❌ |

实时层是纯增量推送（Dispatcher 订阅事件 → WS 广播），**无离线补偿**。客户端断线重连后，只能靠 seq 增量拉新消息；编辑和 reaction 这类"原地变更"看不到，必须全量重载才一致。

随着编辑（已上线）、reaction（已上线）、未来 @提及等"消息侧变更"增多，这个缺口会持续放大。需要一个**统一的离线追平能力**：客户端带单一游标，一次拉回断线期间会话内发生的所有变更。

**目标**：离线重连追平（client catch-up）。客户端带 `lastChangeSeq`，服务端返回该游标之后该会话的所有变更，客户端逐条 apply 即追平。

## 关键设计决策（已与用户逐项确认）

1. **统一变更流**：所有变更（新消息/编辑/撤回/reaction）进一张 `message_change` 变更日志表，客户端单一游标追平一切。否决了"双流合并"（客户端需合并 message+changelog 两个流）。
2. **游标复用 conversation.last_seq 空间**：不引入独立的 change_seq 序列。每个变更在事务内调现有 `seqAllocator.Next`（行锁分配）拿一个 change_seq。
3. **零间隙保证来自行锁**：现有 `DBAllocator` 通过 `AddLastSeq(1)` 原子 UPDATE 拿 conversation 行写锁、持有到 commit，使**分配序 = 提交序**。否决了 Snowflake/updated_at 游标——它们生成序 ≠ 提交序，并发写会跳隙丢变更。
4. **胖日志**：changelog 行自带 apply 所需数据（与 WS 实时帧同构），客户端追平无需 N+1 回查。在线推送与离线追平共用一套"变更帧" apply 逻辑。
5. **created 不带正文**：新消息正文仍走 PullMessages（seq 增量）这条成熟路径；changelog 的 created 帧只是"第 N 个 seq 是消息 X"的指针，避免正文在两表冗余。
6. **change_seq = message.seq（同一空间）**：新消息只分配一次 seq，message 表与 changelog 行用同一值。changelog 是 last_seq 空间的无间隙投影。
7. **撤回零改动（R1）**：撤回仍写占 seq 的系统条目，该系统条目在 changelog 中表现为一条 created 帧（系统条目也是消息）。客户端据系统条目 payload 的 `recalledMsgId` 标记原消息撤回，与现有在线推送机制完全一致。否决了 R2（去掉系统条目改 recalled 帧）——那是破坏性变更，违反外科手术原则。
8. **reaction/编辑写路径升级**：为可同步，从"无事务单行写"改为"事务内写 + 占一个 seq + 写 changelog"。这是可靠离线同步的必要成本。

## 架构

**表职责分工**：
- `message` 表 = 消息**正文/状态的真相源**（当前值）
- `message_change` 表 = 会话内**变更的有序流**（谁在第几个 change_seq 发生了什么），胖日志，自带 apply 数据，只追加不更新

**写路径**（各用例在现有事务内追加一步 changes.Append，change_seq 复用同一次 seq 分配）：
```
SendMessage:    seq=alloc → message.Create(seq)   → changes.Append(seq, created, msgID, sender)        → 更新会话游标
EditMessage:    seq=alloc → message.Save(改content) → changes.Append(seq, edited, msgID, actor, {content,editedAt})
RecallMessage:  现状不变（写占 seq 的系统条目）→ 系统条目经 SendMessage 同款路径产生 created 帧
AddReaction:    seq=alloc → reaction.Add          → changes.Append(seq, reaction_add, msgID, actor, {userId,emoji})
RemoveReaction: seq=alloc → reaction.Remove       → changes.Append(seq, reaction_remove, msgID, actor, {userId,emoji})
```
> 注：reaction add/remove、edit 原来不开事务、不分配 seq，现在统一包进 `txRunner.RunInTx`，调一次 `seqAllocator.Next`，再写 changelog。撤回路径零改动。

**读路径**（新接口）：
```
GET .../conversations/:cid/changes?afterChangeSeq=N&limit=M
→ SELECT * FROM message_change
  WHERE conversation_id=? AND change_seq > N
  ORDER BY change_seq ASC LIMIT M
→ {changes:[...], nextCursor, hasMore}
```

**客户端追平流程**：断线重连 → 带本地 lastChangeSeq 调 /changes → 收到变更帧批 → 逐条 apply（与 WS 帧同构）→ 推进游标 → hasMore 则翻页续拉。在线时走 WS 帧实时 apply，离线追平走 /changes 重放，两者帧结构一致、客户端一套 apply 逻辑。

## 数据模型

新增 Ent schema `internal/ent/schema/message_change.go`：

```
Fields:
  biz_id          int64  Unique Immutable   // = changeId（Snowflake，行身份）
  conversation_id int64                      // 逻辑外键 → conversations.biz_id
  change_seq      int64                      // 复用 conversation.last_seq 空间，会话内严格递增
  change_type     int8                       // 1=created 2=edited 3=reaction_add 4=reaction_remove
  message_id      int64                      // 逻辑外键 → messages.biz_id（变更目标）
  actor_id        int64                      // 触发者 user_id（系统条目为 0）
  payload         JSON Optional              // 胖日志：apply 所需数据
  TimeMixin 仅取 created_at 语义（日志只追加）—— 沿用 TimeMixin（含 updated_at 字段但不使用）

Indexes:
  (conversation_id, change_seq) Unique       // 游标查询主路径 + 零间隙兜底
```

> `biz_id`（Snowflake）= 行的全局唯一身份（项目惯例，主键去自增）；`change_seq` = 会话内游标。职责分离：前者身份，后者顺序。

**payload 按 change_type**（胖日志，与 WS 帧同构）：
| change_type | payload |
|---|---|
| 1 created | `{}`（正文在 message 表，走 PullMessages）|
| 2 edited | `{content:{...}, editedAt}` |
| 3 reaction_add | `{userId, emoji}` |
| 4 reaction_remove | `{userId, emoji}` |

> 撤回不占独立 change_type：撤回系统条目（content_type=recall 的 message）经 created（type=1）帧进流，客户端据其 content 的 recalledMsgId 标记撤回。

## API 契约

```
GET /api/v1/workspaces/{ws}/conversations/{cid}/changes
    ?afterChangeSeq=<int64>   // 客户端游标，0=从头
    &limit=<int>              // 默认 50，上限 200（复用 message pageSize/maxPageSize）
```
鉴权/作用域：`WorkspaceScope` 中间件 + `requireActiveMember`（会话活跃成员），复刻 PullMessages。

**响应**：
```json
{
  "code": 0, "message": "success",
  "data": {
    "changes": [
      {"changeSeq": 101, "changeType": 1, "messageId": "8001", "actorId": "9", "payload": {}},
      {"changeSeq": 102, "changeType": 2, "messageId": "8001", "actorId": "9",
       "payload": {"content": {"text": "改后"}, "editedAt": "2026-06-30T10:05:00Z"}},
      {"changeSeq": 103, "changeType": 3, "messageId": "8001", "actorId": "9",
       "payload": {"userId": "9", "emoji": "👍"}}
    ],
    "nextCursor": 103,
    "hasMore": false
  }
}
```
- `nextCursor` = 本批最后一条 changeSeq；`hasMore` = 本批是否取满 limit。客户端下次带 nextCursor 续拉。

## 改动清单（新增 N / 修改 M）

### 数据模型与生成
- **N** `internal/ent/schema/message_change.go`（照抄 reaction.go 模式）
- 生成：`make generate` → `internal/ent/messagechange*.go`；迁移经 `cmd/migrate`

### Domain（新聚合包，独立于 message/reaction 聚合）
- **N** `internal/modules/messaging/domain/messagechange/messagechange.go`：`Change` 聚合（私有字段 + getter + `New(...)` 工厂校验 change_type 合法 + `UnmarshalFromDB`）。**不得 import message/reaction 聚合**。
- **N** `internal/modules/messaging/domain/messagechange/repository.go`：`Repository` 接口——`Append(ctx, *Change) error`、`ListAfter(ctx, convID, afterSeq int64, limit int) ([]*Change, error)`。
- **N** `internal/modules/messaging/domain/messagechange/errors.go`：`ErrInvalidChangeType`。

### Application
- **N** `internal/modules/messaging/application/query/change_queries.go`：`ListChangesQuery{WorkspaceID, OperatorUserID, ConversationID, AfterChangeSeq, Limit}`。
- **N** `internal/modules/messaging/application/result/change_result.go`：`ChangeResult{ChangeSeq, ChangeType, MessageID, ActorID, Payload}`。
- **M** `application/service/message_queries.go`：`MessageService.ListChanges`（requireActiveMember → ListAfter → 映射 + hasMore/nextCursor）。
- **M** `application/service/message_commands.go`：SendMessage / EditMessage / AddReaction / RemoveReaction 各在事务内追加 `changes.Append`；reaction/edit 用例升级为事务 + 分配 seq（见写路径）。
- **M** `MessageService` 构造函数 + struct：注入 `messagechange.Repository`。

### Adapter/HTTP
- **M** `adapter/http/handler/message_handler.go`：`ListChanges` handler + `MessageUseCase` 接口加该方法。
- **N** `adapter/http/response/change_response.go`：`ChangesResponse{Changes, NextCursor, HasMore}` + `ToChangesResponse`。
- **M** `adapter/http/router/router.go`：注册 `GET .../changes`。

### Infra/Persistence
- **N** `internal/modules/messaging/infra/persistence/message_change_repository.go`：实现 `Append`（ent create）、`ListAfter`（where conversation_id + change_seq > N, order asc, limit）、`toDomain`。
- **M** `internal/modules/messaging/module.go`：装配 `MessageChangeRepository` 并注入 `MessageService`。

### 契约
- **M** `api/openapi.yaml`：加 `/changes` 端点（operationId `listChanges`）+ `ChangesResponse`/`ChangeItem` schema，`make lint-openapi` 须 100/100。

## 错误处理与一致性

- **原子性**：`Append` 与业务写在同一事务，要么全成功要么全回滚——不会出现"消息存了 changelog 漏了"。
- **零间隙**：change_seq 来自行锁分配（`AddLastSeq`），分配序 = 提交序；`(conversation_id, change_seq)` 唯一索引兜底并发。客户端可凭 changeSeq 连续性检测丢失。
- **读接口错误**：非成员 → 403；非法 afterChangeSeq（负数）→ 400；limit 越界 → 钳到 maxPageSize（不报错，复刻 PullMessages）。
- **作用域隔离**：change 查询严格按 conversation_id 过滤。

## 测试

| 层 | 用例 |
|---|---|
| domain messagechange | New/UnmarshalFromDB 字段透传、change_type 合法性（非法值拒绝）|
| service ListChanges | 非成员拒绝、afterSeq 过滤、limit 钳制、hasMore 计算、空结果 |
| service 写路径 | 发消息/编辑/加 reaction/删 reaction 各在事务内写对应 change（fake change repo 断言）；编辑+reaction 后 changeSeq 连续递增、无间隙 |
| infra persistence | Append + ListAfter 映射单测 |
| 集成测试（核心）| 发消息→编辑→加 reaction→撤回，`ListChanges(afterSeq=0)` 一次拉回全部变更，断言顺序/类型/payload/changeSeq 连续；再 `ListChanges(afterSeq=中段)` 验证增量 |
| 集成测试（R1）| 撤回经系统条目 created 帧出现在流中 |

## 验证闭环

- 契约先行：`api/openapi.yaml` 加端点，`make lint-openapi` 100/100
- Schema：`make generate` 重新生成 ent（受限环境关沙箱 + `GOPROXY=https://goproxy.cn,direct`；go.mod churn 用 `git checkout` 还原）；迁移 `cmd/migrate`
- 分层：`make test-architecture`（messagechange 聚合不 import message/reaction 聚合；domain 不 import 基础设施）
- 全量：`make verify`（覆盖率 ≥70%）
- 提交：中文 Conventional Commits，如 `feat(messaging): 会话变更增量同步接口`

## 后续（本次不做）

- changelog 压缩：同一 target 的多次编辑只保留最新（YAGNI，先不做）
- WS 实时帧与 changelog 帧结构正式统一为同一 DTO（目前结构同构但各自定义）
- @提及变更接入 changelog（@提及功能落地时顺带加 change_type）
- changelog 表分区/归档策略（与 message 表分区策略对齐，量大时再做）
