package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

// Todo 是演示实体。
type Todo struct {
	ent.Schema
}

// Fields 返回字段。
func (Todo) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("biz_id").Unique().Immutable(),
		field.String("title"),
		field.String("status").Default("pending"),
		// version 供乐观并发控制：repository.Save 以当前版本为更新条件并自增。
		field.Int("version").Default(1),
	}
}

// Mixin 返回公共字段。
func (Todo) Mixin() []ent.Mixin {
	return []ent.Mixin{TimeMixin{}}
}
