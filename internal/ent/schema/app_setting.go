package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

// AppSetting 提供一个通用占位表，确保基础模板在无 demo 模块时也能迁移。
type AppSetting struct {
	ent.Schema
}

// Fields 返回字段定义。
func (AppSetting) Fields() []ent.Field {
	return []ent.Field{
		field.String("key").Unique().Immutable(),
		field.String("value").Default(""),
	}
}

// Mixin 返回公共字段。
func (AppSetting) Mixin() []ent.Mixin {
	return []ent.Mixin{TimeMixin{}}
}
