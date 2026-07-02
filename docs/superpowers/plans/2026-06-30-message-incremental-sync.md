# 消息增量同步接口 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 新增会话变更增量同步接口，客户端带单一游标一次性追平断线期间的所有变更（新消息/编辑/撤回/reaction 增删）。

**Architecture:** 引入 `message_change` 变更日志表作为会话内变更的有序流。change_seq 复用 conversation.last_seq 空间（行锁分配，零间隙）。每个写用例在同一事务内追加一条胖日志（自带 apply 数据，与 WS 帧同构）。新接口 `GET .../changes?afterChangeSeq=N` 按 change_seq 升序返回变更帧。

**Tech Stack:** Go DDD 分层、Ent ORM（MySQL 无外键）、gin、CQRS 命名、进程内 EventBus。

## Global Constraints

- Go 版本以 `go.mod`/CI 为准，不擅自升级
- 分层依赖：adapter → application → domain ← infra；domain 零外部依赖（不引 gin/ent/viper/slog）
- domain 按聚合分包，聚合包间禁止互 import（跨聚合协调在 application 层）
- 仓储接口定义在聚合包内（命名 `Repository`），实现在 infra 层
- CQRS：写入参 `<动词><名词>Command`、读入参 `<动词><名词>Query`、出参 `<名词>Result`；查询只读不发事件
- 生成代码不手改；改 schema 后 `make generate`
- HTTP request/response 契约只从 command/query/result 映射，不 import domain
- change_type 枚举全程固定：`1=created 2=edited 3=reaction_add 4=reaction_remove`（撤回不占独立类型，经系统条目 created 帧同步）
- 受限环境 Go 缓存：`GOCACHE=$TMPDIR/go-build`（不覆盖 GOMODCACHE，用默认 `~/.Go/pkg/mod`）；generate/verify 需关沙箱
- 提交用中文 Conventional Commits
- 事务边界：`seqAllocator.Next(ctx, convID)` 必须在 `txRunner.RunInTx` 内调用；`idGenerator.NextID(ctx)` 生成 biz_id

---

## File Structure

**新增：**
- `internal/ent/schema/message_change.go` — Ent schema
- `internal/modules/messaging/domain/messagechange/messagechange.go` — Change 聚合
- `internal/modules/messaging/domain/messagechange/repository.go` — Repository 接口
- `internal/modules/messaging/domain/messagechange/errors.go` — 哨兵错误
- `internal/modules/messaging/domain/messagechange/messagechange_test.go` — domain 单测
- `internal/modules/messaging/infra/persistence/message_change_repository.go` — 仓储实现
- `internal/modules/messaging/infra/persistence/message_change_repository_test.go` — 映射单测
- `internal/modules/messaging/application/query/change_queries.go` — ListChangesQuery
- `internal/modules/messaging/application/result/change_result.go` — ChangeResult
- `internal/modules/messaging/application/service/change_queries_test.go` — ListChanges 单测
- `internal/modules/messaging/adapter/http/response/change_response.go` — ChangesResponse
- `internal/integration/messaging_change_suite_test.go` — 集成测试

**修改：**
- `internal/modules/messaging/application/service/message_service.go` — 注入 changes repo
- `internal/modules/messaging/application/service/message_queries.go` — ListChanges 方法
- `internal/modules/messaging/application/service/message_commands.go` — Edit 写路径改造 + changelog
- `internal/modules/messaging/application/service/reaction_commands.go` — reaction 写路径改造 + changelog
- `internal/modules/messaging/application/service/message_service_test.go` — 构造 helper 注入 changes repo
- `internal/modules/messaging/application/service/reaction_commands_test.go` — 新增 memChangeRepo fake
- `internal/modules/messaging/adapter/http/handler/message_handler.go` — ListChanges handler + 接口
- `internal/modules/messaging/adapter/http/handler/error_test.go` / `handler_test.go` — fake 补方法 + 端点表
- `internal/modules/messaging/adapter/http/router/router.go` — 注册路由
- `internal/modules/messaging/module.go` — 装配 changes repo
- `api/openapi.yaml` — /changes 端点 + schema

---

## Task 1: message_change Ent schema 与生成

**Files:**
- Create: `internal/ent/schema/message_change.go`
- Modify (generated): `internal/ent/messagechange*.go` 等

**Interfaces:**
- Produces: ent 表 `message_change`，字段 `biz_id, conversation_id, change_seq, change_type, message_id, actor_id, payload, created_at, updated_at`；生成 setter `SetBizID/SetConversationID/SetChangeSeq/SetChangeType/SetMessageID/SetActorID/SetPayload`；查询谓词 `entmessagechange.ConversationID(x)`, `entmessagechange.ChangeSeqGT(n)`；字段常量 `FieldChangeSeq`

- [ ] **Step 1: 创建 schema 文件**

照抄 `internal/ent/schema/reaction.go` 的模式（biz_id + 逻辑外键 Comment + TimeMixin）。

```go
package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// MessageChange 是会话内变更的有序流（离线增量同步游标载体）。
//
// change_seq 复用 conversation.last_seq 空间（行锁分配，零间隙）；胖日志，payload 自带 apply 数据。
type MessageChange struct {
	ent.Schema
}

// Fields 返回字段定义。
func (MessageChange) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("biz_id").Unique().Immutable(), // = changeId
		// 逻辑外键 → conversations.biz_id
		field.Int64("conversation_id").Comment("逻辑外键 → conversations.biz_id"),
		// change_seq: 复用 conversation.last_seq 空间，会话内严格递增（游标）
		field.Int64("change_seq"),
		// change_type: 1-created 2-edited 3-reaction_add 4-reaction_remove
		field.Int8("change_type"),
		// 逻辑外键 → messages.biz_id（变更目标消息）
		field.Int64("message_id").Comment("逻辑外键 → messages.biz_id"),
		// 逻辑外键 → users.biz_id（触发者，系统条目为 0）
		field.Int64("actor_id").Comment("逻辑外键 → users.biz_id（系统为 0）"),
		// payload: 胖日志 apply 数据（与 WS 帧同构）
		field.JSON("payload", map[string]any{}).Optional(),
	}
}

// Indexes 返回索引定义。
func (MessageChange) Indexes() []ent.Index {
	return []ent.Index{
		// 游标查询主路径 + 零间隙兜底（并发不可能产生重复 change_seq）
		index.Fields("conversation_id", "change_seq").Unique(),
	}
}

// Mixin 返回公共字段。
func (MessageChange) Mixin() []ent.Mixin {
	return []ent.Mixin{TimeMixin{}}
}
```

- [ ] **Step 2: 生成 ent 代码**

Run（受限环境关沙箱 + goproxy）:
```bash
env GOCACHE=/tmp/go-build-cache GOMODCACHE=/tmp/go-mod-cache GOFLAGS=-buildvcs=false GOPROXY=https://goproxy.cn,direct make generate
```
Expected: 生成 `internal/ent/messagechange/`、`internal/ent/messagechange.go` 等；`git status` 显示新增生成文件

- [ ] **Step 3: 确认 go.mod 无 churn**

Run: `git diff --stat go.mod go.sum`
Expected: 无输出（generate 不应改动依赖）。若有 churn：`git checkout go.mod go.sum`

- [ ] **Step 4: 确认字段生成正确**

Run: `grep -n "FieldChangeSeq\|change_seq" internal/ent/messagechange/messagechange.go`
Expected: 出现 `FieldChangeSeq = "change_seq"`

- [ ] **Step 5: 提交**

```bash
git add internal/ent
git commit -m "feat(messaging): message_change 变更日志表 schema"
```

---

## Task 2: messagechange domain 聚合

**Files:**
- Create: `internal/modules/messaging/domain/messagechange/messagechange.go`
- Create: `internal/modules/messaging/domain/messagechange/repository.go`
- Create: `internal/modules/messaging/domain/messagechange/errors.go`
- Test: `internal/modules/messaging/domain/messagechange/messagechange_test.go`

**Interfaces:**
- Produces:
  - `messagechange.ChangeType` int8 + 常量 `ChangeCreated=1, ChangeEdited=2, ChangeReactionAdd=3, ChangeReactionRemove=4`
  - `messagechange.Change` 聚合，getter `ID()/ConversationID()/ChangeSeq()/Type()/MessageID()/ActorID()/Payload() map[string]any/CreatedAt()`
  - `messagechange.New(id, conversationID, changeSeq int64, ct ChangeType, messageID, actorID int64, payload map[string]any) (*Change, error)`
  - `messagechange.UnmarshalFromDB(id, conversationID, changeSeq int64, ct ChangeType, messageID, actorID int64, payload map[string]any, createdAt time.Time) *Change`
  - `messagechange.Repository` 接口：`Append(ctx, *Change) error`；`ListAfter(ctx, conversationID, afterChangeSeq int64, limit int) ([]*Change, error)`
  - `messagechange.ErrInvalidChangeType`

- [ ] **Step 1: 写 domain 单测（先失败）**

```go
package messagechange

import (
	"errors"
	"testing"
	"time"
)

func TestNewValidatesChangeType(t *testing.T) {
	t.Parallel()
	if _, err := New(1, 100, 5, ChangeType(99), 8001, 9, nil); !errors.Is(err, ErrInvalidChangeType) {
		t.Fatalf("非法 change_type 应返回 ErrInvalidChangeType, got %v", err)
	}
}

func TestNewAndGetters(t *testing.T) {
	t.Parallel()
	payload := map[string]any{"emoji": "👍"}
	c, err := New(11, 100, 5, ChangeReactionAdd, 8001, 9, payload)
	if err != nil {
		t.Fatalf("意外错误: %v", err)
	}
	if c.ID() != 11 || c.ConversationID() != 100 || c.ChangeSeq() != 5 {
		t.Fatalf("基础字段错误: %+v", c)
	}
	if c.Type() != ChangeReactionAdd || c.MessageID() != 8001 || c.ActorID() != 9 {
		t.Fatalf("类型/目标字段错误: %+v", c)
	}
	if c.Payload()["emoji"] != "👍" || c.CreatedAt().IsZero() {
		t.Fatalf("payload/时间错误: %+v", c)
	}
}

func TestNewNilPayloadDefaultsEmpty(t *testing.T) {
	t.Parallel()
	c, _ := New(1, 1, 1, ChangeCreated, 1, 0, nil)
	if c.Payload() == nil {
		t.Fatal("nil payload 应兜底为空 map")
	}
}

func TestUnmarshalFromDB(t *testing.T) {
	t.Parallel()
	now := time.Now()
	c := UnmarshalFromDB(11, 100, 5, ChangeEdited, 8001, 9, map[string]any{"content": "x"}, now)
	if c.ID() != 11 || c.Type() != ChangeEdited || !c.CreatedAt().Equal(now) {
		t.Fatalf("重建字段错误: %+v", c)
	}
	// nil payload 兜底。
	c2 := UnmarshalFromDB(1, 1, 1, ChangeCreated, 1, 0, nil, now)
	if c2.Payload() == nil {
		t.Fatal("nil payload 应兜底为空 map")
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `env GOCACHE=$TMPDIR/go-build GOFLAGS=-buildvcs=false go test ./internal/modules/messaging/domain/messagechange/...`
Expected: FAIL（包不存在/未定义）

- [ ] **Step 3: 写 errors.go**

```go
package messagechange

import "errors"

// ErrInvalidChangeType 表示非法的变更类型。
var ErrInvalidChangeType = errors.New("messagechange: invalid change type")
```

- [ ] **Step 4: 写 messagechange.go**

```go
// Package messagechange 是会话变更流聚合：离线增量同步的有序变更日志。
//
// 聚合边界 = 包边界；Change 记录会话内一次变更（新消息/编辑/reaction），
// change_seq 复用 conversation.last_seq 空间。不得 import message/reaction 聚合。
package messagechange

import (
	"time"

	"github.com/maguowei/gotobeta/internal/pkg/apperr"
)

// ChangeType 表示变更类型。
type ChangeType int8

const (
	// ChangeCreated 新消息（含撤回系统条目）。
	ChangeCreated ChangeType = 1
	// ChangeEdited 消息编辑。
	ChangeEdited ChangeType = 2
	// ChangeReactionAdd 添加表情回应。
	ChangeReactionAdd ChangeType = 3
	// ChangeReactionRemove 取消表情回应。
	ChangeReactionRemove ChangeType = 4
)

// Change 是变更流聚合根。
type Change struct {
	id             int64
	conversationID int64
	changeSeq      int64
	changeType     ChangeType
	messageID      int64
	actorID        int64
	payload        map[string]any
	createdAt      time.Time
}

func (c *Change) ID() int64             { return c.id }
func (c *Change) ConversationID() int64 { return c.conversationID }
func (c *Change) ChangeSeq() int64      { return c.changeSeq }
func (c *Change) Type() ChangeType      { return c.changeType }
func (c *Change) MessageID() int64      { return c.messageID }
func (c *Change) ActorID() int64        { return c.actorID }
func (c *Change) Payload() map[string]any { return c.payload }
func (c *Change) CreatedAt() time.Time  { return c.createdAt }

func isValidChangeType(ct ChangeType) bool {
	switch ct {
	case ChangeCreated, ChangeEdited, ChangeReactionAdd, ChangeReactionRemove:
		return true
	default:
		return false
	}
}

// New 创建一条变更记录。changeSeq 由应用层在事务内分配后传入。
func New(id, conversationID, changeSeq int64, ct ChangeType, messageID, actorID int64, payload map[string]any) (*Change, error) {
	if !isValidChangeType(ct) {
		return nil, ErrInvalidChangeType
	}
	if payload == nil {
		payload = map[string]any{}
	}
	return &Change{
		id: id, conversationID: conversationID, changeSeq: changeSeq,
		changeType: ct, messageID: messageID, actorID: actorID,
		payload: payload, createdAt: time.Now(),
	}, nil
}

// UnmarshalFromDB 从数据库重建变更记录。
func UnmarshalFromDB(id, conversationID, changeSeq int64, ct ChangeType, messageID, actorID int64, payload map[string]any, createdAt time.Time) *Change {
	if payload == nil {
		payload = map[string]any{}
	}
	return &Change{
		id: id, conversationID: conversationID, changeSeq: changeSeq,
		changeType: ct, messageID: messageID, actorID: actorID,
		payload: payload, createdAt: createdAt,
	}
}

// 确保 apperr 被引用（New 校验也可直接返回哨兵错误；此处保留 apperr 供未来扩展参数校验）。
var _ = apperr.InvalidParam
```

> 注：如果 lint 报 `apperr` 未使用，删掉最后两行的 `var _ = apperr.InvalidParam` 及 import。先按纯哨兵错误实现，无需 apperr。

- [ ] **Step 5: 写 repository.go**

```go
package messagechange

import "context"

// Repository 是变更流仓储接口。
type Repository interface {
	// Append 追加一条变更记录（在业务写事务内调用，保证原子）。
	Append(ctx context.Context, c *Change) error
	// ListAfter 返回会话内 change_seq > afterChangeSeq 的变更，按 change_seq 升序，最多 limit 条。
	ListAfter(ctx context.Context, conversationID, afterChangeSeq int64, limit int) ([]*Change, error)
}
```

- [ ] **Step 6: 修正 apperr 未使用（若需）**

若 Step 4 保留了 `var _ = apperr.InvalidParam` 且 lint 通过则保留；否则删除该行与 apperr import，运行：
Run: `env GOCACHE=$TMPDIR/go-build GOFLAGS=-buildvcs=false go vet ./internal/modules/messaging/domain/messagechange/...`
Expected: 无 unused import 错误

- [ ] **Step 7: 运行测试确认通过**

Run: `env GOCACHE=$TMPDIR/go-build GOFLAGS=-buildvcs=false go test ./internal/modules/messaging/domain/messagechange/...`
Expected: PASS

- [ ] **Step 8: 提交**

```bash
git add internal/modules/messaging/domain/messagechange
git commit -m "feat(messaging): 变更流 messagechange 聚合与仓储接口"
```

---

## Task 3: MessageChangeRepository 仓储实现

**Files:**
- Create: `internal/modules/messaging/infra/persistence/message_change_repository.go`
- Test: `internal/modules/messaging/infra/persistence/message_change_repository_test.go`

**Interfaces:**
- Consumes: Task 1 的 ent（`ent.Client`, `entmessagechange` 谓词）、Task 2 的 `messagechange.Change`/`messagechange.Repository`
- Produces: `persistence.NewMessageChangeRepository(client *ent.Client, logger *slog.Logger) *MessageChangeRepository` 实现 `messagechange.Repository`；映射函数 `messageChangeToEntity(row *ent.MessageChange) *messagechange.Change`

- [ ] **Step 1: 写映射单测（先失败）**

参考 `reaction_repository_test.go` 的项目惯例（仅单测构造+toDomain 映射，查询由集成测试覆盖）。

```go
package persistence

import (
	"testing"
	"time"

	"github.com/maguowei/gotobeta/internal/ent"
	"github.com/maguowei/gotobeta/internal/modules/messaging/domain/messagechange"
)

func TestMessageChangeToEntity(t *testing.T) {
	now := time.Now()
	row := &ent.MessageChange{
		BizID: 11, ConversationID: 100, ChangeSeq: 5, ChangeType: 3,
		MessageID: 8001, ActorID: 9, Payload: map[string]any{"emoji": "👍"}, CreatedAt: now,
	}
	c := messageChangeToEntity(row)
	if c.ID() != 11 || c.ConversationID() != 100 || c.ChangeSeq() != 5 {
		t.Fatalf("映射基础字段错误: %+v", c)
	}
	if c.Type() != messagechange.ChangeReactionAdd || c.MessageID() != 8001 || c.ActorID() != 9 {
		t.Fatalf("映射类型/目标错误: %+v", c)
	}
	if c.Payload()["emoji"] != "👍" || !c.CreatedAt().Equal(now) {
		t.Fatalf("映射 payload/时间错误: %+v", c)
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `env GOCACHE=$TMPDIR/go-build GOFLAGS=-buildvcs=false go test ./internal/modules/messaging/infra/persistence/ -run TestMessageChangeToEntity`
Expected: FAIL（messageChangeToEntity 未定义）

- [ ] **Step 3: 写仓储实现**

照抄 `reaction_repository.go` 的 `entdb.ClientFromCtx(ctx, r.client)` 事务感知模式（Append 会在外层事务内调用，必须走 ClientFromCtx）。

```go
package persistence

import (
	"context"
	"log/slog"

	"github.com/maguowei/gotobeta/internal/ent"
	entmessagechange "github.com/maguowei/gotobeta/internal/ent/messagechange"
	"github.com/maguowei/gotobeta/internal/infra/entdb"
	"github.com/maguowei/gotobeta/internal/modules/messaging/domain/messagechange"
)

// MessageChangeRepository 是变更流仓储的 Ent 实现。
type MessageChangeRepository struct {
	client *ent.Client
	logger *slog.Logger
}

// NewMessageChangeRepository 创建仓储。
func NewMessageChangeRepository(client *ent.Client, logger *slog.Logger) *MessageChangeRepository {
	return &MessageChangeRepository{client: client, logger: logger}
}

// Append 追加一条变更记录（事务感知：外层事务内调用时走事务 client）。
func (r *MessageChangeRepository) Append(ctx context.Context, c *messagechange.Change) error {
	client := entdb.ClientFromCtx(ctx, r.client)
	_, err := client.MessageChange.Create().
		SetBizID(c.ID()).
		SetConversationID(c.ConversationID()).
		SetChangeSeq(c.ChangeSeq()).
		SetChangeType(int8(c.Type())).
		SetMessageID(c.MessageID()).
		SetActorID(c.ActorID()).
		SetPayload(c.Payload()).
		SetCreatedAt(c.CreatedAt()).
		Save(ctx)
	return err
}

// ListAfter 返回会话内 change_seq > afterChangeSeq 的变更，按 change_seq 升序。
func (r *MessageChangeRepository) ListAfter(ctx context.Context, conversationID, afterChangeSeq int64, limit int) ([]*messagechange.Change, error) {
	client := entdb.ClientFromCtx(ctx, r.client)
	rows, err := client.MessageChange.Query().
		Where(entmessagechange.ConversationID(conversationID), entmessagechange.ChangeSeqGT(afterChangeSeq)).
		Order(ent.Asc(entmessagechange.FieldChangeSeq)).
		Limit(limit).
		All(ctx)
	if err != nil {
		return nil, err
	}
	items := make([]*messagechange.Change, 0, len(rows))
	for _, row := range rows {
		items = append(items, messageChangeToEntity(row))
	}
	return items, nil
}

func messageChangeToEntity(row *ent.MessageChange) *messagechange.Change {
	return messagechange.UnmarshalFromDB(
		row.BizID, row.ConversationID, row.ChangeSeq,
		messagechange.ChangeType(row.ChangeType), row.MessageID, row.ActorID,
		row.Payload, row.CreatedAt,
	)
}
```

- [ ] **Step 4: 运行测试确认通过**

Run: `env GOCACHE=$TMPDIR/go-build GOFLAGS=-buildvcs=false go test ./internal/modules/messaging/infra/persistence/ -run TestMessageChangeToEntity`
Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add internal/modules/messaging/infra/persistence/message_change_repository.go internal/modules/messaging/infra/persistence/message_change_repository_test.go
git commit -m "feat(messaging): MessageChangeRepository 仓储实现"
```

---

## Task 4: ListChanges 查询用例（读路径）

**Files:**
- Create: `internal/modules/messaging/application/query/change_queries.go`
- Create: `internal/modules/messaging/application/result/change_result.go`
- Modify: `internal/modules/messaging/application/service/message_service.go`（struct + 构造函数注入 changes repo）
- Modify: `internal/modules/messaging/application/service/message_queries.go`（ListChanges 方法）
- Modify: `internal/modules/messaging/application/service/message_service_test.go`（构造 helper 注入 changes）
- Create: `internal/modules/messaging/application/service/change_queries_test.go`

**Interfaces:**
- Consumes: Task 2 的 `messagechange.Repository`
- Produces:
  - `query.ListChangesQuery{WorkspaceID, OperatorUserID, ConversationID, AfterChangeSeq int64, Limit int}`
  - `result.ChangeResult{ChangeSeq int64, ChangeType int8, MessageID int64, ActorID int64, Payload map[string]any}`
  - `result.ChangesPage{Changes []*ChangeResult, NextCursor int64, HasMore bool}`
  - `MessageService.ListChanges(ctx, q ListChangesQuery) (*result.ChangesPage, error)`
  - `MessageService` 构造函数新增第 4 个参数 `changes messagechange.Repository`（紧跟 reactions 之后）

- [ ] **Step 1: 写 query 与 result**

`change_queries.go`:
```go
package query

// ListChangesQuery 增量拉取会话变更流入参。
type ListChangesQuery struct {
	WorkspaceID    int64
	OperatorUserID int64
	ConversationID int64
	AfterChangeSeq int64
	Limit          int
}
```

`change_result.go`:
```go
package result

// ChangeResult 是单条变更视图。
type ChangeResult struct {
	ChangeSeq   int64          `json:"changeSeq"`
	ChangeType  int8           `json:"changeType"`
	MessageID   int64          `json:"messageId"`
	ActorID     int64          `json:"actorId"`
	Payload     map[string]any `json:"payload"`
}

// ChangesPage 是一页变更流结果。
type ChangesPage struct {
	Changes    []*ChangeResult
	NextCursor int64
	HasMore    bool
}
```

- [ ] **Step 2: 注入 changes repo 到 MessageService**

Modify `message_service.go`：struct 增加字段（在 `reactions` 之后）:
```go
	reactions     reaction.Repository
	changes       messagechange.Repository
```
import 增加：
```go
	"github.com/maguowei/gotobeta/internal/modules/messaging/domain/messagechange"
```
构造函数 `NewMessageService` 参数增加（在 `reactions reaction.Repository,` 之后）：
```go
	reactions reaction.Repository,
	changes messagechange.Repository,
```
赋值增加：
```go
		reactions:     reactions,
		changes:       changes,
```

- [ ] **Step 3: 写 ListChanges 单测（先失败）**

`change_queries_test.go`:
```go
package service

import (
	"context"
	"testing"

	messagingquery "github.com/maguowei/gotobeta/internal/modules/messaging/application/query"
)

func TestListChangesNonMemberForbidden(t *testing.T) {
	convRepo := newMemConvRepo()
	msgRepo := newMemMsgRepo()
	seedActiveMember(convRepo, 100, 9)
	svc := newMsgService(convRepo, msgRepo, &capturePublisher{})

	if _, err := svc.ListChanges(context.Background(), messagingquery.ListChangesQuery{
		WorkspaceID: 1, OperatorUserID: 999, ConversationID: 100, AfterChangeSeq: 0, Limit: 50,
	}); err == nil {
		t.Fatal("非成员拉取变更应被拒绝")
	}
}

func TestListChangesReturnsAfterSeqAndHasMore(t *testing.T) {
	convRepo := newMemConvRepo()
	msgRepo := newMemMsgRepo()
	pub := &capturePublisher{}
	seedActiveMember(convRepo, 100, 9)
	svc := newMsgService(convRepo, msgRepo, pub)

	// 发 3 条消息 → 变更流应有 3 条 created。
	svc.SendMessage(context.Background(), textCmd(100, 9, "c1", "a"))
	svc.SendMessage(context.Background(), textCmd(100, 9, "c2", "b"))
	svc.SendMessage(context.Background(), textCmd(100, 9, "c3", "c"))

	// limit=2 → 取前 2，hasMore=true。
	page, err := svc.ListChanges(context.Background(), messagingquery.ListChangesQuery{
		WorkspaceID: 1, OperatorUserID: 9, ConversationID: 100, AfterChangeSeq: 0, Limit: 2,
	})
	if err != nil {
		t.Fatalf("拉取失败: %v", err)
	}
	if len(page.Changes) != 2 {
		t.Fatalf("应返回 2 条, got %d", len(page.Changes))
	}
	if !page.HasMore {
		t.Fatal("取满 limit 应 hasMore=true")
	}
	if page.NextCursor != page.Changes[1].ChangeSeq {
		t.Fatalf("nextCursor 应为末条 changeSeq: %d vs %d", page.NextCursor, page.Changes[1].ChangeSeq)
	}

	// 带 nextCursor 续拉 → 剩 1 条，hasMore=false。
	page2, _ := svc.ListChanges(context.Background(), messagingquery.ListChangesQuery{
		WorkspaceID: 1, OperatorUserID: 9, ConversationID: 100, AfterChangeSeq: page.NextCursor, Limit: 2,
	})
	if len(page2.Changes) != 1 || page2.HasMore {
		t.Fatalf("续拉应剩 1 条且 hasMore=false: len=%d hasMore=%v", len(page2.Changes), page2.HasMore)
	}
}
```

> 依赖：`newMsgService` 需注入 memChangeRepo（Step 5 更新 helper），且 SendMessage 需写 changelog（Task 6）。本任务先让 helper 编译通过 + ListChanges 存在；`TestListChangesReturnsAfterSeqAndHasMore` 会在 Task 6 完成后才全绿。本步只要求 `TestListChangesNonMemberForbidden` 通过。

- [ ] **Step 4: 写 ListChanges 方法**

Modify `message_queries.go`，追加：
```go
// ListChanges 增量拉取会话变更流，需为该会话活跃成员。
func (s *MessageService) ListChanges(ctx context.Context, q messagingquery.ListChangesQuery) (*messagingresult.ChangesPage, error) {
	if _, err := s.requireActiveMember(ctx, q.ConversationID, q.OperatorUserID); err != nil {
		return nil, err
	}
	limit := q.Limit
	if limit <= 0 {
		limit = s.pageSize
	}
	if limit > s.maxPageSize {
		limit = s.maxPageSize
	}
	changes, err := s.changes.ListAfter(ctx, q.ConversationID, q.AfterChangeSeq, limit)
	if err != nil {
		return nil, wrapInfrastructureError("拉取变更流失败", err)
	}
	items := make([]*messagingresult.ChangeResult, 0, len(changes))
	for _, c := range changes {
		items = append(items, &messagingresult.ChangeResult{
			ChangeSeq:  c.ChangeSeq(),
			ChangeType: int8(c.Type()),
			MessageID:  c.MessageID(),
			ActorID:    c.ActorID(),
			Payload:    c.Payload(),
		})
	}
	page := &messagingresult.ChangesPage{Changes: items, HasMore: len(items) == limit}
	if len(items) > 0 {
		page.NextCursor = items[len(items)-1].ChangeSeq
	} else {
		page.NextCursor = q.AfterChangeSeq
	}
	return page, nil
}
```

- [ ] **Step 5: 更新 service 测试 helper 注入 memChangeRepo**

在 `message_service_test.go` 增加内存 change repo，并更新 `newMsgServiceWithWindow`：
```go
// memChangeRepo 是内存版变更流仓储。
type memChangeRepo struct{ items []*messagechange.Change }

func newMemChangeRepo() *memChangeRepo { return &memChangeRepo{} }

func (r *memChangeRepo) Append(_ context.Context, c *messagechange.Change) error {
	r.items = append(r.items, c)
	return nil
}

func (r *memChangeRepo) ListAfter(_ context.Context, convID, afterSeq int64, limit int) ([]*messagechange.Change, error) {
	var out []*messagechange.Change
	for _, c := range r.items {
		if c.ConversationID() == convID && c.ChangeSeq() > afterSeq {
			out = append(out, c)
			if len(out) >= limit {
				break
			}
		}
	}
	return out, nil
}
```
import 增加 `"github.com/maguowei/gotobeta/internal/modules/messaging/domain/messagechange"`。

更新构造 helper（增加 changes 参数注入）：
```go
func newMsgServiceWithWindow(convRepo *memConvRepo, msgRepo *memMsgRepo, pub *capturePublisher, window time.Duration) *MessageService {
	return NewMessageService(msgRepo, convRepo, newMemReactionRepo(), newMemChangeRepo(), &memSeqAlloc{}, allowChecker{}, pub, &fakeIDGen{}, directTxRunner{}, window, 50, slog.Default(), nil)
}
```

> 同步更新 `message_service_test.go` 中所有直接调 `NewMessageService(...)` 的位置（`TestSendMessageRecordsMetrics`、`TestRecallExpiredWindow`），在 `newMemReactionRepo(),` 后插入 `newMemChangeRepo(),`。用 grep 找全：
> Run: `grep -rn "NewMessageService(" internal/modules/messaging/application/service/`

- [ ] **Step 6: 更新 reaction 测试 helper（若引用 NewMessageService）**

Run: `grep -rn "NewMessageService(" internal/modules/messaging/`
对每个测试调用点，在 `newMemReactionRepo(),`（或对应 reaction repo 实参）之后插入 changes repo 实参。reaction_commands_test.go 若有独立构造需同样处理。

- [ ] **Step 7: 运行测试**

Run: `env GOCACHE=$TMPDIR/go-build GOFLAGS=-buildvcs=false go test ./internal/modules/messaging/application/service/ -run "TestListChangesNonMemberForbidden|TestSendMessage"`
Expected: `TestListChangesNonMemberForbidden` PASS；SendMessage 相关 PASS（helper 编译通过）

- [ ] **Step 8: 提交**

```bash
git add internal/modules/messaging/application internal/modules/messaging/application/service
git commit -m "feat(messaging): ListChanges 查询用例与 changes repo 注入"
```

---

## Task 5: 组合根装配 changes repo

**Files:**
- Modify: `internal/modules/messaging/module.go`

**Interfaces:**
- Consumes: Task 3 的 `NewMessageChangeRepository`、Task 4 的 `NewMessageService` 新签名
- Produces: 装配后 msgSvc 持有真实 changes repo

- [ ] **Step 1: 装配 changes repo**

Modify `module.go`，在 `reactionRepo := ...` 之后：
```go
	reactionRepo := messagingpersist.NewReactionRepository(client, logger)
	changeRepo := messagingpersist.NewMessageChangeRepository(client, logger)
```
更新 `NewMessageService` 调用（在 `reactionRepo,` 之后插入 `changeRepo,`）：
```go
	msgSvc := messagingsvc.NewMessageService(
		msgRepo, convRepo, reactionRepo, changeRepo, seqAllocator, checker, publisher, idGen, txRunner,
		recallWindow(cfg), cfg.IM.MessagePageSize, logger, metrics,
	)
```

- [ ] **Step 2: 编译确认**

Run: `env GOCACHE=$TMPDIR/go-build GOFLAGS=-buildvcs=false go build ./internal/modules/messaging/...`
Expected: 无输出（编译通过）

- [ ] **Step 3: 提交**

```bash
git add internal/modules/messaging/module.go
git commit -m "feat(messaging): 组合根装配 changes 仓储"
```

---

## Task 6: SendMessage 写路径接入 changelog

**Files:**
- Modify: `internal/modules/messaging/application/service/message_commands.go`（SendMessage 事务内追加 Append）

**Interfaces:**
- Consumes: `s.changes.Append`, `s.idGenerator.NextID`, `messagechange.New/ChangeCreated`
- Produces: SendMessage 提交后变更流多一条 created 记录（change_seq = message.seq）

- [ ] **Step 1: 在 SendMessage 事务内追加 changelog**

Modify `message_commands.go`，import 增加：
```go
	"github.com/maguowei/gotobeta/internal/modules/messaging/domain/messagechange"
```
在 SendMessage 的 `RunInTx` 回调内，`conv.ApplyMessage(...)` 与 `s.conversations.Save(...)` 之后、`msg = m` 之前，追加：
```go
			changeID, err := s.idGenerator.NextID(txCtx)
			if err != nil {
				return wrapInfrastructureError("生成变更 ID 失败", err)
			}
			chg, err := messagechange.New(changeID, cmd.ConversationID, seq, messagechange.ChangeCreated, m.ID(), m.SenderID(), map[string]any{})
			if err != nil {
				return err
			}
			if err := s.changes.Append(txCtx, chg); err != nil {
				return wrapInfrastructureError("追加变更流失败", err)
			}
```

> change_seq 复用同一个 `seq`（Task 已定：created 的 change_seq == message.seq）。

- [ ] **Step 2: 运行 SendMessage 与 ListChanges 测试**

Run: `env GOCACHE=$TMPDIR/go-build GOFLAGS=-buildvcs=false go test ./internal/modules/messaging/application/service/ -run "TestSendMessage|TestListChanges"`
Expected: 全部 PASS（含 Task 4 的 `TestListChangesReturnsAfterSeqAndHasMore` 现在应全绿）

- [ ] **Step 3: 提交**

```bash
git add internal/modules/messaging/application/service/message_commands.go
git commit -m "feat(messaging): SendMessage 写入 created 变更流"
```

---

## Task 7: 撤回写路径接入 changelog

**Files:**
- Modify: `internal/modules/messaging/application/service/message_commands.go`（RecallMessage 系统条目也写 changelog）

**Interfaces:**
- Consumes: 同 Task 6
- Produces: 撤回提交后变更流多一条 created 记录（针对系统撤回条目，change_seq = 系统条目 seq，payload 含 recalledMsgId）

- [ ] **Step 1: 在 RecallMessage 事务内为系统条目追加 changelog**

Modify RecallMessage 的 `RunInTx` 回调，在系统条目 `conv.ApplyMessage(...)` 与 `s.conversations.Save(...)` 之后、`sysMsg = sys` 之前追加。系统条目的 payload 复用其 content（含 recalledMsgId），actor 为操作者：
```go
			changeID, err := s.idGenerator.NextID(txCtx)
			if err != nil {
				return wrapInfrastructureError("生成变更 ID 失败", err)
			}
			chg, err := messagechange.New(changeID, cmd.ConversationID, seq, messagechange.ChangeCreated, sys.ID(), cmd.OperatorUserID, map[string]any{
				"recalledMsgId": msg.ID(),
			})
			if err != nil {
				return err
			}
			if err := s.changes.Append(txCtx, chg); err != nil {
				return wrapInfrastructureError("追加变更流失败", err)
			}
```

> 注：`seq` 是系统条目分配的 seq（RecallMessage 事务内已有 `seq, err := s.seqAllocator.Next(...)`）。change_seq 复用它。

- [ ] **Step 2: 运行撤回测试**

Run: `env GOCACHE=$TMPDIR/go-build GOFLAGS=-buildvcs=false go test ./internal/modules/messaging/application/service/ -run "TestRecall"`
Expected: PASS（现有撤回测试不受影响）

- [ ] **Step 3: 提交**

```bash
git add internal/modules/messaging/application/service/message_commands.go
git commit -m "feat(messaging): 撤回系统条目写入变更流"
```

---

## Task 8: EditMessage 写路径改造（事务 + seq + changelog）

**Files:**
- Modify: `internal/modules/messaging/application/service/message_commands.go`（EditMessage 包进事务，分配 seq，写 changelog）

**Interfaces:**
- Consumes: `s.txRunner.RunInTx`, `s.seqAllocator.Next`, `s.changes.Append`, `messagechange.ChangeEdited`
- Produces: EditMessage 提交后变更流多一条 edited 记录（change_seq = 新分配 seq，payload 含 content+editedAt）

- [ ] **Step 1: 改造 EditMessage 为事务写路径**

当前 EditMessage 是：`msg.Edit(...)` → `s.messages.Save(ctx, msg)` → 发事件。改为事务内分配 seq、Save、写 changelog：

替换 EditMessage 中从 `if err := s.messages.Save(ctx, msg); err != nil {` 到发事件之前的段落为：
```go
	err = s.txRunner.RunInTx(ctx, func(txCtx context.Context) error {
		seq, err := s.seqAllocator.Next(txCtx, cmd.ConversationID)
		if err != nil {
			return wrapInfrastructureError("分配 seq 失败", err)
		}
		if err := s.messages.Save(txCtx, msg); err != nil {
			return wrapInfrastructureError("保存编辑内容失败", err)
		}
		changeID, err := s.idGenerator.NextID(txCtx)
		if err != nil {
			return wrapInfrastructureError("生成变更 ID 失败", err)
		}
		chg, err := messagechange.New(changeID, cmd.ConversationID, seq, messagechange.ChangeEdited, msg.ID(), cmd.OperatorUserID, map[string]any{
			"content":  msg.Content(),
			"editedAt": msg.EditedAt(),
		})
		if err != nil {
			return err
		}
		if err := s.changes.Append(txCtx, chg); err != nil {
			return wrapInfrastructureError("追加变更流失败", err)
		}
		return nil
	})
	if err != nil {
		loggerx.WithError(ctx, s.logger, "edit message failed", err, slog.Int64("messageId", cmd.MessageID))
		return nil, err
	}
```

> 注意 `err` 变量：EditMessage 开头是 `msg, err := s.messages.FindByID(...)`，`err` 已声明，此处用 `err =`（非 `:=`）。确认 `msg.Edit(...)` 调用仍在事务外（它只改内存对象，Save 才落库）。

- [ ] **Step 2: 运行编辑测试**

Run: `env GOCACHE=$TMPDIR/go-build GOFLAGS=-buildvcs=false go test ./internal/modules/messaging/application/service/ -run "TestEditMessage"`
Expected: PASS（现有编辑测试 + 事务内 directTxRunner 直通）

- [ ] **Step 3: 补编辑写 changelog 的断言测试**

在 `edit_commands_test.go` 追加：
```go
func TestEditMessageWritesChange(t *testing.T) {
	convRepo := newMemConvRepo()
	msgRepo := newMemMsgRepo()
	changeRepo := newMemChangeRepo()
	pub := &capturePublisher{}
	seedActiveMember(convRepo, 100, 9)
	svc := NewMessageService(msgRepo, convRepo, newMemReactionRepo(), changeRepo, &memSeqAlloc{}, allowChecker{}, pub, &fakeIDGen{}, directTxRunner{}, 2*time.Minute, 50, slog.Default(), nil)

	sent, _ := svc.SendMessage(context.Background(), textCmd(100, 9, "c1", "old"))
	if _, err := svc.EditMessage(context.Background(), editCmd(100, 9, sent.MessageID, "new")); err != nil {
		t.Fatalf("编辑失败: %v", err)
	}
	// 变更流应含 1 条 created + 1 条 edited。
	var edited *messagechange.Change
	for _, c := range changeRepo.items {
		if c.Type() == messagechange.ChangeEdited {
			edited = c
		}
	}
	if edited == nil {
		t.Fatal("应写入 edited 变更")
	}
	if edited.MessageID() != sent.MessageID || edited.Payload()["content"] == nil {
		t.Fatalf("edited 变更字段错误: %+v", edited)
	}
}
```
import 增加 `"github.com/maguowei/gotobeta/internal/modules/messaging/domain/messagechange"`。

Run: `env GOCACHE=$TMPDIR/go-build GOFLAGS=-buildvcs=false go test ./internal/modules/messaging/application/service/ -run "TestEditMessageWritesChange"`
Expected: PASS

- [ ] **Step 4: 提交**

```bash
git add internal/modules/messaging/application/service/message_commands.go internal/modules/messaging/application/service/edit_commands_test.go
git commit -m "feat(messaging): 编辑写路径改造为事务并写入变更流"
```

---

## Task 9: AddReaction/RemoveReaction 写路径改造

**Files:**
- Modify: `internal/modules/messaging/application/service/reaction_commands.go`

**Interfaces:**
- Consumes: `s.txRunner.RunInTx`, `s.seqAllocator.Next`, `s.changes.Append`, `messagechange.ChangeReactionAdd/ChangeReactionRemove`
- Produces: AddReaction 成功（非幂等命中）后变更流多一条 reaction_add；RemoveReaction 成功（确有删除）后多一条 reaction_remove。幂等 no-op 不写变更。

- [ ] **Step 1: 改造 AddReaction 为事务写路径**

import 增加 `messagechange` 与 `context`（已有）。替换 AddReaction 中从 `if err := s.reactions.Add(ctx, rc); err != nil {` 到发事件之前的段落：
```go
	err = s.txRunner.RunInTx(ctx, func(txCtx context.Context) error {
		if err := s.reactions.Add(txCtx, rc); err != nil {
			return err // 含 reaction.ErrAlreadyExists，外层判定
		}
		seq, err := s.seqAllocator.Next(txCtx, cmd.ConversationID)
		if err != nil {
			return wrapInfrastructureError("分配 seq 失败", err)
		}
		changeID, err := s.idGenerator.NextID(txCtx)
		if err != nil {
			return wrapInfrastructureError("生成变更 ID 失败", err)
		}
		chg, err := messagechange.New(changeID, cmd.ConversationID, seq, messagechange.ChangeReactionAdd, cmd.MessageID, cmd.OperatorUserID, map[string]any{
			"userId": cmd.OperatorUserID,
			"emoji":  cmd.Emoji,
		})
		if err != nil {
			return err
		}
		return s.changes.Append(txCtx, chg)
	})
	if err != nil {
		if stderrors.Is(err, reaction.ErrAlreadyExists) {
			return nil // 幂等：已回应过，no-op 不发事件、不写变更（事务已回滚）
		}
		return wrapInfrastructureError("保存表情回应失败", err)
	}
```

> 关键：`reaction.Add` 命中唯一约束返回 `ErrAlreadyExists` → 事务回滚 → seq/changelog 都不写，保证幂等无副作用。原 `id, err := s.idGenerator.NextID(ctx)` 生成 reaction id 的代码保持在事务外（rc 构造不变）。

- [ ] **Step 2: 改造 RemoveReaction 为事务写路径**

替换 RemoveReaction 中从 `removed, err := s.reactions.Remove(...)` 到发事件之前：
```go
	var removed bool
	err := s.txRunner.RunInTx(ctx, func(txCtx context.Context) error {
		var rerr error
		removed, rerr = s.reactions.Remove(txCtx, cmd.MessageID, cmd.OperatorUserID, cmd.Emoji)
		if rerr != nil {
			return wrapInfrastructureError("删除表情回应失败", rerr)
		}
		if !removed {
			return nil // 未回应过，幂等 no-op（不写变更）
		}
		seq, err := s.seqAllocator.Next(txCtx, cmd.ConversationID)
		if err != nil {
			return wrapInfrastructureError("分配 seq 失败", err)
		}
		changeID, err := s.idGenerator.NextID(txCtx)
		if err != nil {
			return wrapInfrastructureError("生成变更 ID 失败", err)
		}
		chg, err := messagechange.New(changeID, cmd.ConversationID, seq, messagechange.ChangeReactionRemove, cmd.MessageID, cmd.OperatorUserID, map[string]any{
			"userId": cmd.OperatorUserID,
			"emoji":  cmd.Emoji,
		})
		if err != nil {
			return err
		}
		return s.changes.Append(txCtx, chg)
	})
	if err != nil {
		return err
	}
	if !removed {
		return nil
	}
```

> 注意变量声明：AddReaction 原有 `rc, err := reaction.New(...)`，`err` 已声明，事务用 `err =`。RemoveReaction 原本没有前置 `err`，用 `err :=`。按实际编译错误调整 `:=`/`=`。

- [ ] **Step 3: 更新 reaction 测试构造并补断言**

`reaction_commands_test.go` 中所有 `NewMessageService(...)` 调用注入 `newMemChangeRepo()`（在 reaction repo 实参后）。补一条断言：
```go
func TestAddReactionWritesChange(t *testing.T) {
	convRepo := newMemConvRepo()
	msgRepo := newMemMsgRepo()
	changeRepo := newMemChangeRepo()
	seedActiveMember(convRepo, 100, 9)
	svc := NewMessageService(msgRepo, convRepo, newMemReactionRepo(), changeRepo, &memSeqAlloc{}, allowChecker{}, &capturePublisher{}, &fakeIDGen{}, directTxRunner{}, 2*time.Minute, 50, slog.Default(), nil)

	sent, _ := svc.SendMessage(context.Background(), textCmd(100, 9, "c1", "hi"))
	if err := svc.AddReaction(context.Background(), messagingcmd.AddReactionCommand{
		WorkspaceID: 1, ConversationID: 100, MessageID: sent.MessageID, OperatorUserID: 9, Emoji: "👍",
	}); err != nil {
		t.Fatalf("加 reaction 失败: %v", err)
	}
	// 重复添加（幂等）不应新增变更。
	_ = svc.AddReaction(context.Background(), messagingcmd.AddReactionCommand{
		WorkspaceID: 1, ConversationID: 100, MessageID: sent.MessageID, OperatorUserID: 9, Emoji: "👍",
	})
	var addCount int
	for _, c := range changeRepo.items {
		if c.Type() == messagechange.ChangeReactionAdd {
			addCount++
		}
	}
	if addCount != 1 {
		t.Fatalf("幂等添加应只写 1 条 reaction_add 变更, got %d", addCount)
	}
}
```
import 增加 messagechange 与 messagingcmd（若缺）。

> 注意：memReactionRepo 需支持唯一约束幂等（重复 Add 返回 ErrAlreadyExists）。检查其实现——若不支持，本断言的"幂等只写 1 条"会失败。Run: `grep -n "ErrAlreadyExists\|func.*memReactionRepo.*Add" internal/modules/messaging/application/service/reaction_commands_test.go`，若 fake 未实现幂等，补上去重逻辑。

- [ ] **Step 4: 运行 reaction 测试**

Run: `env GOCACHE=$TMPDIR/go-build GOFLAGS=-buildvcs=false go test ./internal/modules/messaging/application/service/ -run "Reaction"`
Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add internal/modules/messaging/application/service/reaction_commands.go internal/modules/messaging/application/service/reaction_commands_test.go
git commit -m "feat(messaging): reaction 增删写路径改造为事务并写入变更流"
```

---

## Task 10: HTTP 端点（handler + response + router）

**Files:**
- Create: `internal/modules/messaging/adapter/http/response/change_response.go`
- Modify: `internal/modules/messaging/adapter/http/handler/message_handler.go`
- Modify: `internal/modules/messaging/adapter/http/router/router.go`
- Modify: `internal/modules/messaging/adapter/http/handler/error_test.go` / `handler_test.go`

**Interfaces:**
- Consumes: Task 4 的 `ListChanges`, `ListChangesQuery`, `ChangesPage`, `ChangeResult`
- Produces: `GET .../changes` 端点；`response.ToChangesResponse`；`MessageUseCase` 接口新增 `ListChanges`

- [ ] **Step 1: 写 change_response.go**

```go
package response

import (
	messagingresult "github.com/maguowei/gotobeta/internal/modules/messaging/application/result"
)

// ChangeItem 是单条变更响应。
type ChangeItem struct {
	ChangeSeq  int64          `json:"changeSeq"`
	ChangeType int8           `json:"changeType"`
	MessageID  int64          `json:"messageId,string"`
	ActorID    int64          `json:"actorId,string"`
	Payload    map[string]any `json:"payload"`
}

// ChangesResponse 是变更流分页响应。
type ChangesResponse struct {
	Changes    []ChangeItem `json:"changes"`
	NextCursor int64        `json:"nextCursor"`
	HasMore    bool         `json:"hasMore"`
}

// ToChangesResponse 转换变更流结果页。
func ToChangesResponse(page *messagingresult.ChangesPage) ChangesResponse {
	items := make([]ChangeItem, 0, len(page.Changes))
	for _, c := range page.Changes {
		items = append(items, ChangeItem{
			ChangeSeq:  c.ChangeSeq,
			ChangeType: c.ChangeType,
			MessageID:  c.MessageID,
			ActorID:    c.ActorID,
			Payload:    c.Payload,
		})
	}
	return ChangesResponse{Changes: items, NextCursor: page.NextCursor, HasMore: page.HasMore}
}
```

- [ ] **Step 2: handler 接口加 ListChanges 并写 handler**

Modify `message_handler.go`，`MessageUseCase` 接口增加：
```go
	ListChanges(ctx context.Context, q messagingquery.ListChangesQuery) (*messagingresult.ChangesPage, error)
```
新增 handler（放 PullMessages 之后）：
```go
// ListChanges 增量拉取会话变更流。
func (h *MessageHandler) ListChanges(c *gin.Context) {
	claims, ok := httpmiddleware.RequireClaims(c)
	if !ok {
		return
	}
	wsID, cid, ok := parseWsConv(c)
	if !ok {
		return
	}
	afterChangeSeq, err := parseNonNegativeQuery(c.Query("afterChangeSeq"))
	if err != nil {
		httpresponse.ErrorWithCode(c, httpresponse.CodeInvalidParam, "无效的 afterChangeSeq")
		return
	}
	limit, err := parseNonNegativeQuery(c.Query("limit"))
	if err != nil {
		httpresponse.ErrorWithCode(c, httpresponse.CodeInvalidParam, "无效的 limit")
		return
	}
	page, err := h.usecase.ListChanges(c.Request.Context(), messagingquery.ListChangesQuery{
		WorkspaceID: wsID, OperatorUserID: claims.UserID, ConversationID: cid,
		AfterChangeSeq: afterChangeSeq, Limit: int(limit),
	})
	if err != nil {
		httpresponse.Error(c, err)
		return
	}
	httpresponse.Success(c, messagingresp.ToChangesResponse(page))
}
```

- [ ] **Step 3: router 注册路由**

Modify `router.go`，在 PullMessages 路由（`group.GET(".../messages", mh.PullMessages)`）之后：
```go
	group.GET("/workspaces/:ws/conversations/:cid/changes", mh.ListChanges)
```

- [ ] **Step 4: 补 handler 测试 fake 方法与端点表**

`error_test.go` 的 `errUC` 加：
```go
func (u errUC) ListChanges(_ context.Context, _ messagingquery.ListChangesQuery) (*messagingresult.ChangesPage, error) {
	return nil, u.err
}
```
`handler_test.go` 的 `fakeUC` 加：
```go
func (fakeUC) ListChanges(_ context.Context, _ messagingquery.ListChangesQuery) (*messagingresult.ChangesPage, error) {
	return &messagingresult.ChangesPage{Changes: []*messagingresult.ChangeResult{{ChangeSeq: 1, ChangeType: 1, MessageID: 8001}}, NextCursor: 1, HasMore: false}, nil
}
```
`error_test.go` 两个端点表（TestMessagingUseCaseErrors 的 endpoints、TestMessagingMissingClaims 的 endpoints）各加：
```go
		{"changes", "GET", "/api/v1/workspaces/1/conversations/100/changes?afterChangeSeq=0&limit=10", ""},
```
（missing-claims 表用无 name 版本 `{"GET", "/api/v1/workspaces/1/conversations/100/changes?afterChangeSeq=0&limit=10", ""}`）

- [ ] **Step 5: 运行 handler 测试**

Run: `env GOCACHE=$TMPDIR/go-build GOFLAGS=-buildvcs=false go test ./internal/modules/messaging/adapter/...`
Expected: PASS

- [ ] **Step 6: 提交**

```bash
git add internal/modules/messaging/adapter
git commit -m "feat(messaging): 变更流拉取 HTTP 端点"
```

---

## Task 11: OpenAPI 契约

**Files:**
- Modify: `api/openapi.yaml`

**Interfaces:**
- Consumes: Task 10 的端点形态
- Produces: `/changes` 端点 + `ChangesResponse`/`ChangeItem` schema

- [ ] **Step 1: 加端点**

在 messages 相关 path 区域（PullMessages `get` 之后合适位置），加 `GET .../changes`。参考现有 `messages` GET 的 parameters/security 结构：
```yaml
  /api/v1/workspaces/{ws}/conversations/{cid}/changes:
    get:
      operationId: listChanges
      summary: 拉取会话变更流
      description: 增量拉取会话内变更（新消息/编辑/撤回/表情回应），客户端带 afterChangeSeq 游标离线追平；需为会话活跃成员。
      tags:
        - messaging
      security:
        - bearerAuth: []
      parameters:
        - $ref: '#/components/parameters/WorkspaceID'
        - $ref: '#/components/parameters/ConversationID'
        - name: afterChangeSeq
          in: query
          description: 客户端游标，返回该值之后的变更；0 表示从头
          schema:
            type: integer
            format: int64
            default: 0
        - name: limit
          in: query
          description: 单页最大条数（默认 50，上限 200）
          schema:
            type: integer
            default: 50
      responses:
        '200':
          description: 变更流分页
          content:
            application/json:
              example:
                code: 0
                message: success
                data:
                  changes:
                    - changeSeq: 1
                      changeType: 1
                      messageId: "8001"
                      actorId: "9"
                      payload: {}
                  nextCursor: 1
                  hasMore: false
              schema:
                $ref: '#/components/schemas/ChangesResponse'
        '403':
          description: 非会话成员
          content:
            application/json:
              example:
                code: 40301
                message: 不是该会话成员
                data: null
              schema:
                $ref: '#/components/schemas/ErrorResponse'
```

- [ ] **Step 2: 加 schema**

在 components/schemas 加（MessageItem 附近）：
```yaml
    ChangeItem:
      description: 单条会话变更
      type: object
      properties:
        changeSeq:
          type: integer
          format: int64
          example: 1
        changeType:
          type: integer
          description: 1-新消息 2-编辑 3-表情添加 4-表情取消
          example: 1
        messageId:
          type: string
          example: "8001"
        actorId:
          type: string
          example: "9"
        payload:
          type: object
          example: {}
    ChangesResponse:
      description: 变更流分页响应
      type: object
      required:
        - code
        - message
        - data
      properties:
        code:
          type: integer
          example: 0
        message:
          type: string
          example: success
        data:
          type: object
          properties:
            changes:
              type: array
              items:
                $ref: '#/components/schemas/ChangeItem'
            nextCursor:
              type: integer
              format: int64
              example: 1
            hasMore:
              type: boolean
              example: false
```

- [ ] **Step 3: 校验 openapi**

Run: `make lint-openapi`
Expected: `Quality Score: 100/100`

- [ ] **Step 4: 提交**

```bash
git add api/openapi.yaml
git commit -m "docs(openapi): 会话变更流拉取端点"
```

---

## Task 12: 集成测试（核心追平场景）

**Files:**
- Create: `internal/integration/messaging_change_suite_test.go`

**Interfaces:**
- Consumes: 全链路真实实现
- Produces: 验证发消息→编辑→加 reaction→撤回后，ListChanges 一次追平全部变更、顺序/类型/changeSeq 连续；增量续拉正确

- [ ] **Step 1: 写集成测试 suite**

照抄 `messaging_edit_suite_test.go` 的 SetupSuite（testcontainers MySQL + seed + 装配 MessageService，注意注入 `NewMessageChangeRepository`）。核心测试：
```go
//go:build integration

package integration_test

// ... imports 照抄 messaging_edit_suite_test.go，另加 messagechange 无需（用 result.ChangeType 数值断言）

func (s *MessagingChangeSuite) TestChangeStreamCatchUp() {
	ctx := context.Background()
	const ownerID int64 = 9201

	ws, _ := s.wsSvc.CreateWorkspace(ctx, workspacecmd.CreateWorkspaceCommand{Slug: "chg-team", Name: "Chg", OwnerUserID: ownerID})
	conv, _ := s.convSvc.CreateConversation(ctx, messagingcmd.CreateConversationCommand{
		WorkspaceID: ws.ID, OperatorUserID: ownerID, Type: int8(conversation.TypeGroup), Name: "g",
	})

	// 发消息 → 编辑 → 加 reaction。
	msg, _ := s.msgSvc.SendMessage(ctx, messagingcmd.SendMessageCommand{
		WorkspaceID: ws.ID, ConversationID: conv.ID, SenderUserID: ownerID,
		ClientMsgID: "c1", ContentType: 1, Content: map[string]any{"text": "原始"},
	})
	s.msgSvc.EditMessage(ctx, messagingcmd.EditMessageCommand{
		WorkspaceID: ws.ID, ConversationID: conv.ID, OperatorUserID: ownerID,
		MessageID: msg.MessageID, Content: map[string]any{"text": "编辑后"},
	})
	s.msgSvc.AddReaction(ctx, messagingcmd.AddReactionCommand{
		WorkspaceID: ws.ID, ConversationID: conv.ID, MessageID: msg.MessageID, OperatorUserID: ownerID, Emoji: "👍",
	})

	// 一次追平：afterChangeSeq=0 应拉回 created + edited + reaction_add。
	page, err := s.msgSvc.ListChanges(ctx, messagingquery.ListChangesQuery{
		WorkspaceID: ws.ID, OperatorUserID: ownerID, ConversationID: conv.ID, AfterChangeSeq: 0, Limit: 50,
	})
	s.Require().NoError(err)
	s.Require().Len(page.Changes, 3)
	// change_seq 严格递增无间隙。
	s.Equal(int8(1), page.Changes[0].ChangeType) // created
	s.Equal(int8(2), page.Changes[1].ChangeType) // edited
	s.Equal(int8(3), page.Changes[2].ChangeType) // reaction_add
	s.Greater(page.Changes[1].ChangeSeq, page.Changes[0].ChangeSeq)
	s.Greater(page.Changes[2].ChangeSeq, page.Changes[1].ChangeSeq)
	// edited payload 带新内容。
	s.NotNil(page.Changes[1].Payload["content"])

	// 增量续拉：从第 1 条之后拉，应剩 2 条。
	page2, _ := s.msgSvc.ListChanges(ctx, messagingquery.ListChangesQuery{
		WorkspaceID: ws.ID, OperatorUserID: ownerID, ConversationID: conv.ID,
		AfterChangeSeq: page.Changes[0].ChangeSeq, Limit: 50,
	})
	s.Require().Len(page2.Changes, 2)
}

func (s *MessagingChangeSuite) TestRecallAppearsAsCreatedInStream() {
	ctx := context.Background()
	const ownerID int64 = 9202
	ws, _ := s.wsSvc.CreateWorkspace(ctx, workspacecmd.CreateWorkspaceCommand{Slug: "chg-team2", Name: "Chg2", OwnerUserID: ownerID})
	conv, _ := s.convSvc.CreateConversation(ctx, messagingcmd.CreateConversationCommand{
		WorkspaceID: ws.ID, OperatorUserID: ownerID, Type: int8(conversation.TypeGroup), Name: "g",
	})
	msg, _ := s.msgSvc.SendMessage(ctx, messagingcmd.SendMessageCommand{
		WorkspaceID: ws.ID, ConversationID: conv.ID, SenderUserID: ownerID,
		ClientMsgID: "c1", ContentType: 1, Content: map[string]any{"text": "hi"},
	})
	s.Require().NoError(s.msgSvc.RecallMessage(ctx, messagingcmd.RecallMessageCommand{
		WorkspaceID: ws.ID, ConversationID: conv.ID, OperatorUserID: ownerID, MessageID: msg.MessageID,
	}))
	page, _ := s.msgSvc.ListChanges(ctx, messagingquery.ListChangesQuery{
		WorkspaceID: ws.ID, OperatorUserID: ownerID, ConversationID: conv.ID, AfterChangeSeq: 0, Limit: 50,
	})
	// 原消息 created + 撤回系统条目 created，共 2 条，末条 payload 含 recalledMsgId。
	s.Require().Len(page.Changes, 2)
	last := page.Changes[len(page.Changes)-1]
	s.Equal(int8(1), last.ChangeType)
	s.NotNil(last.Payload["recalledMsgId"])
}
```
SetupSuite 与 `func TestMessagingChangeSuite(t *testing.T)` 照抄 edit suite 结构（改 struct 名 `MessagingChangeSuite`、noopPublisher 名避免重复声明用 `changeNoopPublisher`）。

- [ ] **Step 2: 编译集成测试**

Run: `env GOCACHE=$TMPDIR/go-build GOFLAGS=-buildvcs=false go build -tags integration ./internal/integration/...`
Expected: 无输出

- [ ] **Step 3: 提交**

```bash
git add internal/integration/messaging_change_suite_test.go
git commit -m "test(messaging): 变更流追平集成测试"
```

---

## Task 13: 全量验证

- [ ] **Step 1: make verify**

Run: `env GOFLAGS=-buildvcs=false make verify`（用默认缓存；受限环境覆盖率步骤关沙箱）
Expected: 全部检查通过（lint 0 issues、openapi 100/100、test-architecture 通过、覆盖率 ≥70%、build 成功）

- [ ] **Step 2: 若覆盖率不足**

补 domain/service 边界测试（change_type 全枚举、limit 钳制上界、空结果 nextCursor=afterChangeSeq）。

- [ ] **Step 3: 最终提交（若 Step 2 有补充）**

```bash
git add -A
git commit -m "test(messaging): 补变更流边界测试至覆盖率门禁"
```

---

## Self-Review 记录

- **Spec 覆盖**：统一流（Task 6/7/8/9 全变更进 changelog）✓；复用 last_seq（Task 6/8/9 事务内 seqAllocator.Next）✓；胖日志（payload 带 content/emoji）✓；created 不带正文（Task 6 payload 空 map）✓；同 seq（Task 6 change_seq=message.seq）✓；撤回 R1（Task 7 系统条目 created 帧）✓；游标零间隙（行锁分配 + 唯一索引）✓；API 契约（Task 10/11）✓；测试矩阵（domain/service/infra/集成 Task 2/3/4/8/9/12）✓
- **change_type 一致性**：全程 `1=created 2=edited 3=reaction_add 4=reaction_remove`，撤回不占独立类型 ✓
- **类型一致性**：`ListChanges`/`ChangesPage`/`ChangeResult`/`Append`/`ListAfter` 签名跨任务一致 ✓
- **风险点标注**：Task 4 Step 5/6 需 grep 找全 NewMessageService 调用点更新；Task 9 Step 3 需确认 memReactionRepo fake 幂等；Task 8/9 的 `:=`/`=` 按编译错误调整
