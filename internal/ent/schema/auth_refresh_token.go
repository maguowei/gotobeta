package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// AuthRefreshToken 保存 refresh token 哈希与轮换状态。
type AuthRefreshToken struct {
	ent.Schema
}

// Fields 返回字段定义。
func (AuthRefreshToken) Fields() []ent.Field {
	return []ent.Field{
		field.String("token_id").MaxLen(64).Unique().Immutable(),
		field.Int64("user_biz_id"),
		field.String("token_hash").MaxLen(64).Unique(),
		field.String("replaced_by_token_id").MaxLen(64).Optional().Nillable(),
		field.Time("expires_at"),
		field.Time("revoked_at").Optional().Nillable(),
		field.String("revoke_reason").MaxLen(64).Default(""),
	}
}

// Indexes 返回索引定义。
func (AuthRefreshToken) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_biz_id", "revoked_at", "expires_at"),
		index.Fields("expires_at"),
	}
}

// Mixin 返回公共字段。
func (AuthRefreshToken) Mixin() []ent.Mixin {
	return []ent.Mixin{TimeMixin{}}
}
