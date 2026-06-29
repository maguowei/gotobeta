package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// AuthActionToken 保存邮箱验证、密码重置和 OAuth 登录码。
type AuthActionToken struct {
	ent.Schema
}

// Fields 返回字段定义。
func (AuthActionToken) Fields() []ent.Field {
	return []ent.Field{
		field.String("token_id").MaxLen(64).Unique().Immutable(),
		field.Int64("user_biz_id"),
		field.String("purpose").MaxLen(32),
		field.String("token_hash").MaxLen(64).Unique(),
		field.String("target_email_normalized").MaxLen(320).Default(""),
		field.Time("expires_at"),
		field.Time("consumed_at").Optional().Nillable(),
	}
}

// Indexes 返回索引定义。
func (AuthActionToken) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_biz_id", "purpose", "consumed_at", "expires_at"),
		index.Fields("expires_at"),
	}
}

// Mixin 返回公共字段。
func (AuthActionToken) Mixin() []ent.Mixin {
	return []ent.Mixin{TimeMixin{}}
}
