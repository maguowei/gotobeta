package persistence

import (
	"context"
	"log/slog"

	"github.com/maguowei/gotobeta/internal/ent"
	entmsg "github.com/maguowei/gotobeta/internal/ent/message"
	"github.com/maguowei/gotobeta/internal/infra/entdb"
	"github.com/maguowei/gotobeta/internal/modules/messaging/domain/message"
)

// MessageRepository 是消息仓储的 Ent 实现。
type MessageRepository struct {
	client *ent.Client
	logger *slog.Logger
}

// NewMessageRepository 创建仓储。
func NewMessageRepository(client *ent.Client, logger *slog.Logger) *MessageRepository {
	return &MessageRepository{client: client, logger: logger}
}

// Create 保存消息。
func (r *MessageRepository) Create(ctx context.Context, m *message.Message) error {
	client := entdb.ClientFromCtx(ctx, r.client)
	create := client.Message.Create().
		SetBizID(m.ID()).
		SetConversationID(m.ConversationID()).
		SetSeq(m.Seq()).
		SetSenderType(int8(m.SenderType())).
		SetSenderID(m.SenderID()).
		SetContentType(int8(m.ContentType())).
		SetContent(m.Content()).
		SetReplyToMsgID(m.ReplyToMsgID()).
		SetStatus(int8(m.Status())).
		SetServerTime(m.ServerTime()).
		SetMetadata(m.Metadata()).
		SetCreatedAt(m.CreatedAt()).
		SetUpdatedAt(m.UpdatedAt())
	if m.ClientMsgID() != nil {
		create.SetClientMsgID(*m.ClientMsgID())
	}
	if _, err := create.Save(ctx); err != nil {
		return err
	}
	return nil
}

// FindByID 按业务 ID 查找。
func (r *MessageRepository) FindByID(ctx context.Context, id int64) (*message.Message, error) {
	client := entdb.ClientFromCtx(ctx, r.client)
	row, err := client.Message.Query().Where(entmsg.BizID(id)).Only(ctx)
	if err != nil {
		return nil, mapEntNotFound(err, message.ErrNotFound)
	}
	return messageToEntity(row), nil
}

// FindByClientMsgID 按会话内幂等键查找。
func (r *MessageRepository) FindByClientMsgID(ctx context.Context, conversationID int64, clientMsgID string) (*message.Message, error) {
	client := entdb.ClientFromCtx(ctx, r.client)
	row, err := client.Message.Query().
		Where(entmsg.ConversationID(conversationID), entmsg.ClientMsgID(clientMsgID)).
		Only(ctx)
	if err != nil {
		return nil, mapEntNotFound(err, message.ErrNotFound)
	}
	return messageToEntity(row), nil
}

// Save 更新消息可变字段（撤回/删除）。
func (r *MessageRepository) Save(ctx context.Context, m *message.Message) error {
	client := entdb.ClientFromCtx(ctx, r.client)
	affected, err := client.Message.Update().
		Where(entmsg.BizID(m.ID())).
		SetStatus(int8(m.Status())).
		SetContent(m.Content()).
		SetNillableEditedAt(m.EditedAt()).
		SetMetadata(m.Metadata()).
		SetUpdatedAt(m.UpdatedAt()).
		Save(ctx)
	if err != nil {
		return err
	}
	if affected == 0 {
		return message.ErrNotFound
	}
	return nil
}

// ListAfterSeq 返回会话内 (afterSeq, +∞) 区间、按 seq 升序的消息。
func (r *MessageRepository) ListAfterSeq(ctx context.Context, conversationID, afterSeq int64, limit int) ([]*message.Message, error) {
	client := entdb.ClientFromCtx(ctx, r.client)
	rows, err := client.Message.Query().
		Where(entmsg.ConversationID(conversationID), entmsg.SeqGT(afterSeq)).
		Order(ent.Asc(entmsg.FieldSeq)).
		Limit(limit).
		All(ctx)
	if err != nil {
		return nil, err
	}
	items := make([]*message.Message, 0, len(rows))
	for _, row := range rows {
		items = append(items, messageToEntity(row))
	}
	return items, nil
}

func messageToEntity(row *ent.Message) *message.Message {
	return message.UnmarshalFromDB(
		row.BizID, row.ConversationID, row.Seq,
		message.SenderType(row.SenderType), row.SenderID, row.ClientMsgID,
		message.ContentType(row.ContentType), row.Content, row.ReplyToMsgID,
		message.Status(row.Status), row.ServerTime, row.EditedAt, row.Metadata,
		row.CreatedAt, row.UpdatedAt,
	)
}
