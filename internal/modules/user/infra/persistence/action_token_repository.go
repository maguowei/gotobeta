package persistence

import (
	"context"
	"time"

	"github.com/maguowei/gotobeta/internal/ent"
	entaction "github.com/maguowei/gotobeta/internal/ent/authactiontoken"
	"github.com/maguowei/gotobeta/internal/infra/entdb"
	"github.com/maguowei/gotobeta/internal/modules/user/domain/actiontoken"
)

// ActionTokenRepository 是一次性动作 token 的 Ent 实现。
type ActionTokenRepository struct {
	client *ent.Client
}

var _ actiontoken.Repository = (*ActionTokenRepository)(nil)

// NewActionTokenRepository 创建 ActionTokenRepository。
func NewActionTokenRepository(client *ent.Client) *ActionTokenRepository {
	return &ActionTokenRepository{client: client}
}

// Create 创建一次性动作 token。
func (r *ActionTokenRepository) Create(ctx context.Context, token *actiontoken.ActionToken) error {
	client := entdb.ClientFromCtx(ctx, r.client)
	return client.AuthActionToken.Create().
		SetTokenID(token.TokenID).
		SetUserBizID(token.UserID).
		SetPurpose(token.Purpose).
		SetTokenHash(token.TokenHash).
		SetTargetEmailNormalized(token.TargetEmailNormalized).
		SetExpiresAt(token.ExpiresAt).
		SetNillableConsumedAt(token.ConsumedAt).
		SetCreatedAt(token.CreatedAt).
		SetUpdatedAt(token.UpdatedAt).
		Exec(ctx)
}

// Consume 消费一次性动作 token。
func (r *ActionTokenRepository) Consume(ctx context.Context, tokenHash string, purpose string, now time.Time) (*actiontoken.ActionToken, error) {
	client := entdb.ClientFromCtx(ctx, r.client)
	row, err := client.AuthActionToken.Query().
		Where(entaction.TokenHash(tokenHash), entaction.Purpose(purpose), entaction.ConsumedAtIsNil(), entaction.ExpiresAtGT(now)).
		Only(ctx)
	if err != nil {
		return nil, mapEntNotFound(err, actiontoken.ErrNotFound)
	}
	if err := client.AuthActionToken.UpdateOne(row).SetConsumedAt(now).SetUpdatedAt(now).Exec(ctx); err != nil {
		return nil, err
	}
	row.ConsumedAt = &now
	return toActionToken(row), nil
}

func toActionToken(row *ent.AuthActionToken) *actiontoken.ActionToken {
	return &actiontoken.ActionToken{
		TokenID:               row.TokenID,
		UserID:                row.UserBizID,
		Purpose:               row.Purpose,
		TokenHash:             row.TokenHash,
		TargetEmailNormalized: row.TargetEmailNormalized,
		ExpiresAt:             row.ExpiresAt,
		ConsumedAt:            row.ConsumedAt,
		CreatedAt:             row.CreatedAt,
		UpdatedAt:             row.UpdatedAt,
	}
}
