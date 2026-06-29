package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// UserIdentity 是用户的第三方登录身份。
type UserIdentity struct {
	ent.Schema
}

// Fields 返回字段定义。
func (UserIdentity) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("biz_id").Unique().Immutable(),
		field.Int64("user_biz_id"),
		field.String("provider").MaxLen(32),
		field.String("provider_user_id").MaxLen(255),
		field.String("provider_email").MaxLen(320).Default(""),
		field.String("provider_email_normalized").MaxLen(320).Default(""),
		field.Bool("provider_email_verified").Default(false),
		field.String("display_name").MaxLen(100).Default(""),
		field.String("avatar_url").MaxLen(1024).Default(""),
		field.String("profile_url").MaxLen(1024).Default(""),
		field.Time("linked_at"),
		field.Time("last_login_at").Optional().Nillable(),
	}
}

// Indexes 返回索引定义。
func (UserIdentity) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("provider", "provider_user_id").Unique(),
		index.Fields("user_biz_id", "provider").Unique(),
		index.Fields("user_biz_id"),
		index.Fields("provider", "provider_email_normalized"),
	}
}

// Mixin 返回公共字段。
func (UserIdentity) Mixin() []ent.Mixin {
	return []ent.Mixin{TimeMixin{}}
}
