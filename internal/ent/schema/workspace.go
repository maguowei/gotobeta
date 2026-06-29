package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// Workspace 是多租户根：用户加入的工作区（Slack 风）。
type Workspace struct {
	ent.Schema
}

// Fields 返回字段定义。
func (Workspace) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("biz_id").Unique().Immutable(),
		field.String("slug").MaxLen(50).Unique(),
		field.String("name").MaxLen(100),
		field.Int64("owner_user_id"),
		// status: 1-正常 2-停用
		field.Int8("status").Default(1),
		field.JSON("settings", map[string]any{}).Optional(),
	}
}

// Indexes 返回索引定义。
func (Workspace) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("owner_user_id"),
	}
}

// Mixin 返回公共字段。
func (Workspace) Mixin() []ent.Mixin {
	return []ent.Mixin{TimeMixin{}}
}
