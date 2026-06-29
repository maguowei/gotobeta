package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// RbacPermissionVersion 是权限缓存版本：版本号变更即精准失效缓存。
type RbacPermissionVersion struct {
	ent.Schema
}

// Fields 返回字段定义。
func (RbacPermissionVersion) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("workspace_id"),
		// subject_type: 1-用户 2-角色
		field.Int8("subject_type"),
		field.Int64("subject_id"),
		field.Int64("version").Default(1),
	}
}

// Indexes 返回索引定义。
func (RbacPermissionVersion) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("workspace_id", "subject_type", "subject_id").Unique(),
	}
}

// Mixin 返回公共字段。
func (RbacPermissionVersion) Mixin() []ent.Mixin {
	return []ent.Mixin{TimeMixin{}}
}
