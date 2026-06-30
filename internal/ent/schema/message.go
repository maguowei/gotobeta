package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// Message 是会话 Timeline 的类型化条目（读扩散，每会话一行 seq）。
type Message struct {
	ent.Schema
}

// Fields 返回字段定义。
func (Message) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("biz_id").Unique().Immutable(), // = msgID
		// 逻辑外键 → conversations.biz_id（不建数据库外键，一致性由应用层 + 唯一索引保证）
		field.Int64("conversation_id").Comment("逻辑外键 → conversations.biz_id"),
		field.Int64("seq"),
		// sender_type: 1-user 2-bot 3-system ← AI 缝
		field.Int8("sender_type").Default(1),
		// 逻辑外键 → users.biz_id（sender_type=1 时；系统条目为 0）
		field.Int64("sender_id").Comment("逻辑外键 → users.biz_id（sender_type=1）"),
		// client_msg_id: 幂等键；系统/撤回条目为 NULL（NULL 不参与唯一约束）
		field.String("client_msg_id").MaxLen(64).Optional().Nillable(),
		// content_type: 1-text 2-image 3-file 4-voice 10-recall 11-system 20-card
		field.Int8("content_type").Default(1),
		// content: content blocks 结构化消息体 ← AI 缝
		field.JSON("content", map[string]any{}).Optional(),
		// 逻辑外键 → messages.biz_id（同会话内被引用消息，0 表示无引用）
		field.Int64("reply_to_msg_id").Default(0).Comment("逻辑外键 → messages.biz_id（同会话，0=无引用）"),
		// status: 1-正常 2-已撤回 3-已删除
		field.Int8("status").Default(1),
		field.Time("server_time").Default(time.Now),
		// edited_at: 末次原地编辑时间；NULL 表示从未编辑（供客户端显示「已编辑」标记）
		field.Time("edited_at").Optional().Nillable(),
		// metadata: AI 打标/情绪/摘要扩展位 ← AI 缝
		field.JSON("metadata", map[string]any{}).Optional(),
	}
}

// Indexes 返回索引定义。
func (Message) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("conversation_id", "seq").Unique(),
		index.Fields("conversation_id", "client_msg_id").Unique(),
		index.Fields("conversation_id", "created_at"),
	}
}

// Mixin 返回公共字段。
func (Message) Mixin() []ent.Mixin {
	return []ent.Mixin{TimeMixin{}}
}
