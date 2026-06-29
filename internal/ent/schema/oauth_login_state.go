package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// OAuthLoginState 保存 OAuth state 哈希，防止 CSRF 与重复 callback。
type OAuthLoginState struct {
	ent.Schema
}

// Fields 返回字段定义。
func (OAuthLoginState) Fields() []ent.Field {
	return []ent.Field{
		field.String("state_hash").MaxLen(64).Unique().Immutable(),
		field.String("provider").MaxLen(32),
		field.String("redirect_url").MaxLen(1024),
		field.Time("expires_at"),
		field.Time("consumed_at").Optional().Nillable(),
	}
}

// Indexes 返回索引定义。
func (OAuthLoginState) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("provider", "consumed_at", "expires_at"),
	}
}

// Mixin 返回公共字段。
func (OAuthLoginState) Mixin() []ent.Mixin {
	return []ent.Mixin{TimeMixin{}}
}
