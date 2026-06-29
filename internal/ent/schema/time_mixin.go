package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/mixin"
)

// TimeMixin 提供创建/更新时间字段。
type TimeMixin struct {
	mixin.Schema
}

// Fields 返回公共字段。
func (TimeMixin) Fields() []ent.Field {
	return []ent.Field{
		field.Time("created_at").
			Default(time.Now).
			Immutable().
			SchemaType(map[string]string{
				"mysql": "datetime(3)",
			}),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now).
			SchemaType(map[string]string{
				"mysql": "datetime(3)",
			}),
	}
}
