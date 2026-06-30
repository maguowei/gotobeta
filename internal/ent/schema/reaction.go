package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// Reaction 是消息表情回应（消息侧附属数据，不占 timeline seq）。
type Reaction struct {
	ent.Schema
}

// Fields 返回字段定义。
func (Reaction) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("biz_id").Unique().Immutable(),
		// 逻辑外键 → conversations.biz_id（用于成员扇出与作用域校验）
		field.Int64("conversation_id").Comment("逻辑外键 → conversations.biz_id"),
		// 逻辑外键 → messages.biz_id（被回应的消息）
		field.Int64("message_id").Comment("逻辑外键 → messages.biz_id"),
		// 逻辑外键 → users.biz_id（回应者）
		field.Int64("user_id").Comment("逻辑外键 → users.biz_id"),
		field.String("emoji").MaxLen(64),
	}
}

// Indexes 返回索引定义。
func (Reaction) Indexes() []ent.Index {
	return []ent.Index{
		// 同一用户对同一消息的同一 emoji 唯一，保证 add 幂等。
		index.Fields("message_id", "user_id", "emoji").Unique(),
		index.Fields("message_id"),
	}
}

// Mixin 返回公共字段。
func (Reaction) Mixin() []ent.Mixin {
	return []ent.Mixin{TimeMixin{}}
}
