package persistence

import (
	"context"
	"time"

	"github.com/maguowei/gotobeta/internal/ent"
	entrefresh "github.com/maguowei/gotobeta/internal/ent/authrefreshtoken"
	"github.com/maguowei/gotobeta/internal/infra/entdb"
	"github.com/maguowei/gotobeta/internal/modules/user/domain/refreshtoken"
)

// RefreshTokenRepository 是 refresh token 的 Ent 实现。
type RefreshTokenRepository struct {
	client *ent.Client
}

var _ refreshtoken.Repository = (*RefreshTokenRepository)(nil)

// NewRefreshTokenRepository 创建 RefreshTokenRepository。
func NewRefreshTokenRepository(client *ent.Client) *RefreshTokenRepository {
	return &RefreshTokenRepository{client: client}
}

// Create 创建 refresh token 记录。
func (r *RefreshTokenRepository) Create(ctx context.Context, token *refreshtoken.RefreshToken) error {
	client := entdb.ClientFromCtx(ctx, r.client)
	return client.AuthRefreshToken.Create().
		SetTokenID(token.TokenID).
		SetUserBizID(token.UserID).
		SetTokenHash(token.TokenHash).
		SetNillableReplacedByTokenID(optionalString(token.ReplacedByTokenID)).
		SetExpiresAt(token.ExpiresAt).
		SetNillableRevokedAt(token.RevokedAt).
		SetRevokeReason(token.RevokeReason).
		SetCreatedAt(token.CreatedAt).
		SetUpdatedAt(token.UpdatedAt).
		Exec(ctx)
}

// FindByHash 查询有效 refresh token。
func (r *RefreshTokenRepository) FindByHash(ctx context.Context, tokenHash string, now time.Time) (*refreshtoken.RefreshToken, error) {
	client := entdb.ClientFromCtx(ctx, r.client)
	row, err := client.AuthRefreshToken.Query().
		Where(entrefresh.TokenHash(tokenHash), entrefresh.RevokedAtIsNil(), entrefresh.ExpiresAtGT(now)).
		Only(ctx)
	if err != nil {
		return nil, mapEntNotFound(err, refreshtoken.ErrNotFound)
	}
	return toRefreshToken(row), nil
}

// Revoke 撤销 refresh token。
func (r *RefreshTokenRepository) Revoke(ctx context.Context, tokenID string, replacedByTokenID string, reason string, now time.Time) error {
	client := entdb.ClientFromCtx(ctx, r.client)
	update := client.AuthRefreshToken.Update().
		Where(entrefresh.TokenID(tokenID), entrefresh.RevokedAtIsNil()).
		SetRevokedAt(now).
		SetRevokeReason(reason).
		SetUpdatedAt(now)
	if replacedByTokenID != "" {
		update.SetReplacedByTokenID(replacedByTokenID)
	}
	affected, err := update.Save(ctx)
	if err != nil {
		return err
	}
	if affected == 0 {
		return refreshtoken.ErrNotFound
	}
	return nil
}

func toRefreshToken(row *ent.AuthRefreshToken) *refreshtoken.RefreshToken {
	return &refreshtoken.RefreshToken{
		TokenID:           row.TokenID,
		UserID:            row.UserBizID,
		TokenHash:         row.TokenHash,
		ReplacedByTokenID: valueString(row.ReplacedByTokenID),
		ExpiresAt:         row.ExpiresAt,
		RevokedAt:         row.RevokedAt,
		RevokeReason:      row.RevokeReason,
		CreatedAt:         row.CreatedAt,
		UpdatedAt:         row.UpdatedAt,
	}
}
