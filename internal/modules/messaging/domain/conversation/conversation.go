// Package conversation 是会话聚合：私聊/群聊/频道统一为读扩散 timeline。
//
// 聚合边界 = 包边界；会话成员（Member）作为聚合内实体随会话管理。
package conversation

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/maguowei/gotobeta/internal/pkg/apperr"
)

// Type 表示会话类型。
type Type int8

const (
	// TypeDM 单聊。
	TypeDM Type = 1
	// TypeGroup 群聊。
	TypeGroup Type = 2
	// TypeChannel 频道。
	TypeChannel Type = 3
)

// Visibility 表示可见性。
type Visibility int8

const (
	// VisibilityPublic 公开。
	VisibilityPublic Visibility = 1
	// VisibilityPrivate 私有。
	VisibilityPrivate Visibility = 2
)

// Status 表示会话状态。
type Status int8

const (
	// StatusActive 正常。
	StatusActive Status = 1
	// StatusArchived 归档。
	StatusArchived Status = 2
	// StatusDissolved 解散。
	StatusDissolved Status = 3
)

// Conversation 是会话聚合根。
type Conversation struct {
	id            int64
	workspaceID   int64
	convType      Type
	visibility    Visibility
	name          string
	topic         string
	creatorID     int64
	dmKey         *string
	lastSeq       int64
	lastMsgID     int64
	lastMsgDigest string
	lastMsgAt     *time.Time
	memberCount   int
	status        Status
	metadata      map[string]any
	createdAt     time.Time
	updatedAt     time.Time
}

func (c *Conversation) ID() int64                { return c.id }
func (c *Conversation) WorkspaceID() int64       { return c.workspaceID }
func (c *Conversation) Type() Type               { return c.convType }
func (c *Conversation) Visibility() Visibility   { return c.visibility }
func (c *Conversation) Name() string             { return c.name }
func (c *Conversation) Topic() string            { return c.topic }
func (c *Conversation) CreatorID() int64         { return c.creatorID }
func (c *Conversation) DMKey() *string           { return c.dmKey }
func (c *Conversation) LastSeq() int64           { return c.lastSeq }
func (c *Conversation) LastMsgID() int64         { return c.lastMsgID }
func (c *Conversation) LastMsgDigest() string    { return c.lastMsgDigest }
func (c *Conversation) LastMsgAt() *time.Time    { return c.lastMsgAt }
func (c *Conversation) MemberCount() int         { return c.memberCount }
func (c *Conversation) Status() Status           { return c.status }
func (c *Conversation) Metadata() map[string]any { return c.metadata }
func (c *Conversation) CreatedAt() time.Time     { return c.createdAt }
func (c *Conversation) UpdatedAt() time.Time     { return c.updatedAt }

// DMKey 为单聊生成确定性去重键：workspace:minUID#maxUID。
func DMKey(workspaceID, userA, userB int64) string {
	ids := []int64{userA, userB}
	slices.Sort(ids)
	return fmt.Sprintf("%d:%d#%d", workspaceID, ids[0], ids[1])
}

// NewDM 创建单聊会话。
func NewDM(id, workspaceID, userA, userB, creatorID int64) (*Conversation, error) {
	if userA == userB {
		return nil, apperr.InvalidParam("单聊不能是自己")
	}
	key := DMKey(workspaceID, userA, userB)
	now := time.Now()
	return &Conversation{
		id: id, workspaceID: workspaceID, convType: TypeDM, visibility: VisibilityPrivate,
		creatorID: creatorID, dmKey: &key, status: StatusActive, memberCount: 2,
		metadata: map[string]any{}, createdAt: now, updatedAt: now,
	}, nil
}

// NewGroup 创建群聊会话。
func NewGroup(id, workspaceID int64, name string, creatorID int64) (*Conversation, error) {
	if strings.TrimSpace(name) == "" {
		return nil, apperr.InvalidParam("群聊名称不能为空")
	}
	now := time.Now()
	return &Conversation{
		id: id, workspaceID: workspaceID, convType: TypeGroup, visibility: VisibilityPrivate,
		name: name, creatorID: creatorID, status: StatusActive, memberCount: 1,
		metadata: map[string]any{}, createdAt: now, updatedAt: now,
	}, nil
}

// NewChannel 创建频道会话。
func NewChannel(id, workspaceID int64, name string, visibility Visibility, creatorID int64) (*Conversation, error) {
	if strings.TrimSpace(name) == "" {
		return nil, apperr.InvalidParam("频道名称不能为空")
	}
	if visibility != VisibilityPublic && visibility != VisibilityPrivate {
		return nil, apperr.InvalidParam("频道可见性非法")
	}
	now := time.Now()
	return &Conversation{
		id: id, workspaceID: workspaceID, convType: TypeChannel, visibility: visibility,
		name: name, creatorID: creatorID, status: StatusActive, memberCount: 1,
		metadata: map[string]any{}, createdAt: now, updatedAt: now,
	}, nil
}

// ApplyMessage 在成功投递一条消息后推进会话的末条消息游标。
func (c *Conversation) ApplyMessage(seq, msgID int64, digest string, at time.Time) {
	c.lastSeq = seq
	c.lastMsgID = msgID
	c.lastMsgDigest = digest
	c.lastMsgAt = &at
	c.updatedAt = at
}

// Archive 归档会话。
func (c *Conversation) Archive() error {
	if c.status == StatusDissolved {
		return apperr.InvalidParam("会话已解散，无法归档")
	}
	c.status = StatusArchived
	c.updatedAt = time.Now()
	return nil
}

// IncrMemberCount 调整成员计数（delta 可为负），不低于 0。
func (c *Conversation) IncrMemberCount(delta int) {
	c.memberCount += delta
	if c.memberCount < 0 {
		c.memberCount = 0
	}
	c.updatedAt = time.Now()
}

// UnmarshalFromDB 从数据库重建会话聚合。
func UnmarshalFromDB(id, workspaceID int64, convType Type, visibility Visibility, name, topic string, creatorID int64, dmKey *string, lastSeq, lastMsgID int64, lastMsgDigest string, lastMsgAt *time.Time, memberCount int, status Status, metadata map[string]any, createdAt, updatedAt time.Time) *Conversation {
	if metadata == nil {
		metadata = map[string]any{}
	}
	return &Conversation{
		id: id, workspaceID: workspaceID, convType: convType, visibility: visibility,
		name: name, topic: topic, creatorID: creatorID, dmKey: dmKey,
		lastSeq: lastSeq, lastMsgID: lastMsgID, lastMsgDigest: lastMsgDigest, lastMsgAt: lastMsgAt,
		memberCount: memberCount, status: status, metadata: metadata,
		createdAt: createdAt, updatedAt: updatedAt,
	}
}
