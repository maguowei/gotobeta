package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// User 是认证用户。
type User struct {
	ent.Schema
}

// Fields 返回字段定义。
func (User) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("biz_id").Unique().Immutable(),
		field.String("email").MaxLen(320),
		field.String("email_normalized").MaxLen(320).Unique(),
		field.Time("email_verified_at").Optional().Nillable(),
		field.String("password_hash").MaxLen(255).Optional().Nillable(),
		field.String("password_hash_alg").MaxLen(32).Optional().Nillable(),
		field.Time("password_set_at").Optional().Nillable(),
		field.String("display_name").MaxLen(100).Default(""),
		field.String("avatar_url").MaxLen(1024).Default(""),
		field.String("status").MaxLen(32).Default("active"),
		field.Time("last_login_at").Optional().Nillable(),
	}
}

// Indexes 返回索引定义。
func (User) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("status", "created_at"),
		index.Fields("email_verified_at"),
	}
}

// Mixin 返回公共字段。
func (User) Mixin() []ent.Mixin {
	return []ent.Mixin{TimeMixin{}}
}
