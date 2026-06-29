package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// Conversation 是会话/频道（读扩散，每会话一行）。
type Conversation struct {
	ent.Schema
}

// Fields 返回字段定义。
func (Conversation) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("biz_id").Unique().Immutable(), // = convID
		// 逻辑外键 → workspaces.biz_id（不建数据库外键，一致性由应用层 + 唯一索引保证）
		field.Int64("workspace_id").Comment("逻辑外键 → workspaces.biz_id"),
		// type: 1-单聊DM 2-群聊 3-频道channel
		field.Int8("type"),
		// visibility: 1-public 2-private
		field.Int8("visibility").Default(2),
		field.String("name").MaxLen(100).Optional().Default(""),
		field.String("topic").MaxLen(255).Optional().Default(""),
		// 逻辑外键 → users.biz_id（会话创建者）
		field.Int64("creator_id").Comment("逻辑外键 → users.biz_id"),
		// dm_key: 单聊去重键 workspace:minUID#maxUID，非单聊为 NULL
		field.String("dm_key").MaxLen(64).Optional().Nillable().Unique(),
		field.Int64("last_seq").Default(0),
		field.Int64("last_msg_id").Default(0),
		field.String("last_msg_digest").MaxLen(255).Optional().Default(""),
		field.Time("last_msg_at").Optional().Nillable(),
		field.Int("member_count").Default(0),
		// status: 1-正常 2-归档 3-解散
		field.Int8("status").Default(1),
		field.JSON("metadata", map[string]any{}).Optional(),
	}
}

// Indexes 返回索引定义。
func (Conversation) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("workspace_id", "type"),
		index.Fields("last_msg_at"),
	}
}

// Mixin 返回公共字段。
func (Conversation) Mixin() []ent.Mixin {
	return []ent.Mixin{TimeMixin{}}
}
