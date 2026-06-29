package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// Attachment 是附件元数据（消息体只存引用 key）。
type Attachment struct {
	ent.Schema
}

// Fields 返回字段定义。
func (Attachment) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("biz_id").Unique().Immutable(),
		field.Int64("workspace_id"),
		field.Int64("uploader_id"),
		field.String("object_key").MaxLen(255).Unique(),
		field.String("file_name").MaxLen(255),
		field.String("content_type").MaxLen(100),
		field.Int64("size_bytes").Default(0),
		// status: 1-待提交 2-已提交
		field.Int8("status").Default(1),
		// metadata: 宽高/时长等 ← AI 缝
		field.JSON("metadata", map[string]any{}).Optional(),
	}
}

// Indexes 返回索引定义。
func (Attachment) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("uploader_id"),
		index.Fields("workspace_id"),
	}
}

// Mixin 返回公共字段。
func (Attachment) Mixin() []ent.Mixin {
	return []ent.Mixin{TimeMixin{}}
}
