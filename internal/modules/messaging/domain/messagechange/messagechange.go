// Package messagechange 是会话变更流聚合：离线增量同步的有序变更日志。
//
// 聚合边界 = 包边界；Change 记录会话内一次变更（新消息/编辑/reaction），
// change_seq 复用 conversation.last_seq 空间。不得 import message/reaction 聚合。
package messagechange

import "time"

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

func (c *Change) ID() int64               { return c.id }
func (c *Change) ConversationID() int64   { return c.conversationID }
func (c *Change) ChangeSeq() int64        { return c.changeSeq }
func (c *Change) Type() ChangeType        { return c.changeType }
func (c *Change) MessageID() int64        { return c.messageID }
func (c *Change) ActorID() int64          { return c.actorID }
func (c *Change) Payload() map[string]any { return c.payload }
func (c *Change) CreatedAt() time.Time    { return c.createdAt }

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
