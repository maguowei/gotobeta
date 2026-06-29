package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// ConversationMember 是会话成员（读水位 + 成员设置）。
type ConversationMember struct {
	ent.Schema
}

// Fields 返回字段定义。
func (ConversationMember) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("biz_id").Unique().Immutable(),
		field.Int64("conversation_id"),
		// member_type: 1-user 2-bot
		field.Int8("member_type").Default(1),
		field.Int64("member_id"),
		// role: 1-owner 2-admin 3-member
		field.Int8("role").Default(3),
		field.Int64("read_seq").Default(0),
		field.Time("last_read_at").Optional().Nillable(),
		field.Bool("is_muted").Default(false),
		field.Bool("is_pinned").Default(false),
		// status: 1-正常 2-已退出
		field.Int8("status").Default(1),
		field.Time("joined_at").Default(time.Now),
	}
}

// Indexes 返回索引定义。
func (ConversationMember) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("conversation_id", "member_type", "member_id").Unique(),
		index.Fields("member_type", "member_id"),
	}
}

// Mixin 返回公共字段。
func (ConversationMember) Mixin() []ent.Mixin {
	return []ent.Mixin{TimeMixin{}}
}
