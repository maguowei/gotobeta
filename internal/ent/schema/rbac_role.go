package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// RbacRole 是角色。workspace_id=0 为平台模板角色，建工作区时复制为租户级。
type RbacRole struct {
	ent.Schema
}

// Fields 返回字段定义。
func (RbacRole) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("biz_id").Unique().Immutable(),
		field.Int64("workspace_id").Default(0),
		field.String("code").MaxLen(64),
		field.String("name").MaxLen(100),
		// role_type: 1-系统 2-工作区
		field.Int8("role_type").Default(2),
		// status: 1-正常 2-禁用
		field.Int8("status").Default(1),
	}
}

// Indexes 返回索引定义。
func (RbacRole) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("workspace_id", "code").Unique(),
		index.Fields("workspace_id", "status"),
	}
}

// Mixin 返回公共字段。
func (RbacRole) Mixin() []ent.Mixin {
	return []ent.Mixin{TimeMixin{}}
}
