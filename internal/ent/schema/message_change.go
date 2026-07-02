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
