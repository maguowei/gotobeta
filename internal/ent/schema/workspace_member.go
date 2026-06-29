package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// WorkspaceMember 记录用户与工作区的成员关系（角色由 RBAC 表承载）。
type WorkspaceMember struct {
	ent.Schema
}

// Fields 返回字段定义。
func (WorkspaceMember) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("biz_id").Unique().Immutable(),
		field.Int64("workspace_id"),
		field.Int64("user_id"),
		// status: 1-正常 2-禁用
		field.Int8("status").Default(1),
		field.Time("joined_at").Default(time.Now),
	}
}

// Indexes 返回索引定义。
func (WorkspaceMember) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("workspace_id", "user_id").Unique(),
		index.Fields("user_id"),
	}
}

// Mixin 返回公共字段。
func (WorkspaceMember) Mixin() []ent.Mixin {
	return []ent.Mixin{TimeMixin{}}
}
