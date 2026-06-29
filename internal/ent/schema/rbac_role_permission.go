package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// RbacRolePermission 是角色与权限的关联。
type RbacRolePermission struct {
	ent.Schema
}

// Fields 返回字段定义。
func (RbacRolePermission) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("workspace_id").Default(0),
		field.Int64("role_id"),
		field.Int64("permission_id"),
	}
}

// Indexes 返回索引定义。
func (RbacRolePermission) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("role_id", "permission_id").Unique(),
		index.Fields("permission_id"),
		index.Fields("workspace_id", "role_id"),
	}
}

// Mixin 返回公共字段。
func (RbacRolePermission) Mixin() []ent.Mixin {
	return []ent.Mixin{TimeMixin{}}
}
