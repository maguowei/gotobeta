package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// RbacPermissionChangeLog 是授权变更审计日志。
type RbacPermissionChangeLog struct {
	ent.Schema
}

// Fields 返回字段定义。
func (RbacPermissionChangeLog) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("biz_id").Unique().Immutable(),
		field.Int64("workspace_id"),
		// change_type: 业务自定义变更类型
		field.Int8("change_type"),
		// target_type: 1-用户 2-角色 3-权限 4-ACL
		field.Int8("target_type"),
		field.Int64("target_id").Default(0),
		field.Int64("operator_id").Default(0),
		field.String("request_id").MaxLen(64).Default(""),
		field.JSON("before_json", map[string]any{}).Optional(),
		field.JSON("after_json", map[string]any{}).Optional(),
		field.String("reason").MaxLen(255).Default(""),
	}
}

// Indexes 返回索引定义。
func (RbacPermissionChangeLog) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("workspace_id", "target_type", "target_id", "created_at"),
		index.Fields("operator_id", "created_at"),
	}
}

// Mixin 返回公共字段。
func (RbacPermissionChangeLog) Mixin() []ent.Mixin {
	return []ent.Mixin{TimeMixin{}}
}
