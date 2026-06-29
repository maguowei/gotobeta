package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// RbacAclEntry 是实例级例外授权（私有频道特批 / 冻结），可审计、可过期、可回收。
type RbacAclEntry struct {
	ent.Schema
}

// Fields 返回字段定义。
func (RbacAclEntry) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("biz_id").Unique().Immutable(),
		field.Int64("workspace_id"),
		// subject_type: 1-用户 2-角色
		field.Int8("subject_type"),
		field.Int64("subject_id"),
		field.String("resource_type").MaxLen(64),
		field.String("resource_id").MaxLen(128),
		field.String("action_code").MaxLen(128),
		// effect: 1-允许 2-拒绝
		field.Int8("effect"),
		field.String("reason").MaxLen(255).Default(""),
		// source_type: 1-手工 2-审批 3-系统策略
		field.Int8("source_type").Default(1),
		field.Time("expires_at").Optional().Nillable(),
		field.Int64("created_by").Default(0),
	}
}

// Indexes 返回索引定义。
func (RbacAclEntry) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("workspace_id", "subject_type", "subject_id", "action_code", "effect"),
		index.Fields("workspace_id", "resource_type", "resource_id", "action_code", "effect"),
		index.Fields("workspace_id", "expires_at"),
	}
}

// Mixin 返回公共字段。
func (RbacAclEntry) Mixin() []ent.Mixin {
	return []ent.Mixin{TimeMixin{}}
}
