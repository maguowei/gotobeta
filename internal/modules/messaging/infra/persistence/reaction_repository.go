package persistence

import (
	"context"
	"log/slog"

	"github.com/maguowei/gotobeta/internal/ent"
	entreaction "github.com/maguowei/gotobeta/internal/ent/reaction"
	"github.com/maguowei/gotobeta/internal/infra/entdb"
	"github.com/maguowei/gotobeta/internal/modules/messaging/domain/reaction"
)

// ReactionRepository 是表情回应仓储的 Ent 实现。
type ReactionRepository struct {
	client *ent.Client
	logger *slog.Logger
}

// NewReactionRepository 创建仓储。
func NewReactionRepository(client *ent.Client, logger *slog.Logger) *ReactionRepository {
	return &ReactionRepository{client: client, logger: logger}
}

// Add 保存一条表情回应；命中唯一约束（已存在）转为 reaction.ErrAlreadyExists（幂等）。
func (r *ReactionRepository) Add(ctx context.Context, rc *reaction.Reaction) error {
	client := entdb.ClientFromCtx(ctx, r.client)
	_, err := client.Reaction.Create().
		SetBizID(rc.ID()).
		SetConversationID(rc.ConversationID()).
		SetMessageID(rc.MessageID()).
		SetUserID(rc.UserID()).
		SetEmoji(rc.Emoji()).
		SetCreatedAt(rc.CreatedAt()).
		Save(ctx)
	if ent.IsConstraintError(err) {
		return reaction.ErrAlreadyExists
	}
	return err
}

// Remove 删除指定回应，返回是否删除了记录。
func (r *ReactionRepository) Remove(ctx context.Context, messageID, userID int64, emoji string) (bool, error) {
	client := entdb.ClientFromCtx(ctx, r.client)
	affected, err := client.Reaction.Delete().
		Where(entreaction.MessageID(messageID), entreaction.UserID(userID), entreaction.Emoji(emoji)).
		Exec(ctx)
	if err != nil {
		return false, err
	}
	return affected > 0, nil
}

// ListByMessageID 返回消息的全部回应，按创建时间升序。
func (r *ReactionRepository) ListByMessageID(ctx context.Context, messageID int64) ([]*reaction.Reaction, error) {
	client := entdb.ClientFromCtx(ctx, r.client)
	rows, err := client.Reaction.Query().
		Where(entreaction.MessageID(messageID)).
		Order(ent.Asc(entreaction.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	items := make([]*reaction.Reaction, 0, len(rows))
	for _, row := range rows {
		items = append(items, reactionToEntity(row))
	}
	return items, nil
}

func reactionToEntity(row *ent.Reaction) *reaction.Reaction {
	return reaction.UnmarshalFromDB(
		row.BizID, row.ConversationID, row.MessageID, row.UserID, row.Emoji, row.CreatedAt,
	)
}
