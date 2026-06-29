package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// RbacPermission 是权限定义（动作目录）。workspace_id=0 为平台通用模板。
type RbacPermission struct {
	ent.Schema
}

// Fields 返回字段定义。
func (RbacPermission) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("biz_id").Unique().Immutable(),
		field.Int64("workspace_id").Default(0),
		field.String("code").MaxLen(128),
		field.String("name").MaxLen(100),
		field.String("resource_type").MaxLen(64).Default(""),
		field.String("action_key").MaxLen(64).Default(""),
		// status: 1-正常 2-禁用
		field.Int8("status").Default(1),
	}
}

// Indexes 返回索引定义。
func (RbacPermission) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("workspace_id", "code").Unique(),
		index.Fields("workspace_id", "resource_type", "action_key"),
	}
}

// Mixin 返回公共字段。
func (RbacPermission) Mixin() []ent.Mixin {
	return []ent.Mixin{TimeMixin{}}
}
