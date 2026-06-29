package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// RbacUserRole 是用户在工作区内的角色授权。
type RbacUserRole struct {
	ent.Schema
}

// Fields 返回字段定义。
func (RbacUserRole) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("workspace_id"),
		field.Int64("user_id"),
		field.Int64("role_id"),
		// source_type: 1-手工 2-默认 4-临时
		field.Int8("source_type").Default(1),
		field.Time("effective_end_at").Optional().Nillable(),
	}
}

// Indexes 返回索引定义。
func (RbacUserRole) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("workspace_id", "user_id", "role_id").Unique(),
		index.Fields("role_id"),
		index.Fields("workspace_id", "user_id", "effective_end_at"),
	}
}

// Mixin 返回公共字段。
func (RbacUserRole) Mixin() []ent.Mixin {
	return []ent.Mixin{TimeMixin{}}
}
