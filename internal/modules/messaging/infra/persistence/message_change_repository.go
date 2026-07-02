package persistence

import (
	"context"
	"log/slog"

	"github.com/maguowei/gotobeta/internal/ent"
	entmessagechange "github.com/maguowei/gotobeta/internal/ent/messagechange"
	"github.com/maguowei/gotobeta/internal/infra/entdb"
	"github.com/maguowei/gotobeta/internal/modules/messaging/domain/messagechange"
)

// MessageChangeRepository 是变更流仓储的 Ent 实现。
type MessageChangeRepository struct {
	client *ent.Client
	logger *slog.Logger
}

// NewMessageChangeRepository 创建仓储。
func NewMessageChangeRepository(client *ent.Client, logger *slog.Logger) *MessageChangeRepository {
	return &MessageChangeRepository{client: client, logger: logger}
}

// Append 追加一条变更记录（事务感知：外层事务内调用时走事务 client）。
func (r *MessageChangeRepository) Append(ctx context.Context, c *messagechange.Change) error {
	client := entdb.ClientFromCtx(ctx, r.client)
	_, err := client.MessageChange.Create().
		SetBizID(c.ID()).
		SetConversationID(c.ConversationID()).
		SetChangeSeq(c.ChangeSeq()).
		SetChangeType(int8(c.Type())).
		SetMessageID(c.MessageID()).
		SetActorID(c.ActorID()).
		SetPayload(c.Payload()).
		SetCreatedAt(c.CreatedAt()).
		Save(ctx)
	return err
}

// ListAfter 返回会话内 change_seq > afterChangeSeq 的变更，按 change_seq 升序。
func (r *MessageChangeRepository) ListAfter(ctx context.Context, conversationID, afterChangeSeq int64, limit int) ([]*messagechange.Change, error) {
	client := entdb.ClientFromCtx(ctx, r.client)
	rows, err := client.MessageChange.Query().
		Where(entmessagechange.ConversationID(conversationID), entmessagechange.ChangeSeqGT(afterChangeSeq)).
		Order(ent.Asc(entmessagechange.FieldChangeSeq)).
		Limit(limit).
		All(ctx)
	if err != nil {
		return nil, err
	}
	items := make([]*messagechange.Change, 0, len(rows))
	for _, row := range rows {
		items = append(items, messageChangeToEntity(row))
	}
	return items, nil
}

func messageChangeToEntity(row *ent.MessageChange) *messagechange.Change {
	return messagechange.UnmarshalFromDB(
		row.BizID, row.ConversationID, row.ChangeSeq,
		messagechange.ChangeType(row.ChangeType), row.MessageID, row.ActorID,
		row.Payload, row.CreatedAt,
	)
}
