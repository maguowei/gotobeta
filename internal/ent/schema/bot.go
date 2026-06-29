package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// Bot 是 Bot/Agent（AI 一等公民，第一期只建模）。
type Bot struct {
	ent.Schema
}

// Fields 返回字段定义。
func (Bot) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("biz_id").Unique().Immutable(),
		field.Int64("workspace_id"),
		field.String("name").MaxLen(100),
		// type: 1-系统bot 2-用户自建 3-Agent（预留）
		field.Int8("type").Default(1),
		field.Int64("owner_user_id").Default(0),
		// config: 未来 AI 配置（模型/提示词/权限范围）← AI 缝
		field.JSON("config", map[string]any{}).Optional(),
		// status: 1-正常 2-停用
		field.Int8("status").Default(1),
	}
}

// Indexes 返回索引定义。
func (Bot) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("workspace_id"),
	}
}

// Mixin 返回公共字段。
func (Bot) Mixin() []ent.Mixin {
	return []ent.Mixin{TimeMixin{}}
}
