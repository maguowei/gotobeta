package persistence

import (
	"context"
	"time"

	"github.com/maguowei/gotobeta/internal/ent"
	entoauthstate "github.com/maguowei/gotobeta/internal/ent/oauthloginstate"
	"github.com/maguowei/gotobeta/internal/infra/entdb"
	"github.com/maguowei/gotobeta/internal/modules/user/domain/oauthstate"
)

// OAuthStateRepository 是 OAuth state 的 Ent 实现。
type OAuthStateRepository struct {
	client *ent.Client
}

var _ oauthstate.Repository = (*OAuthStateRepository)(nil)

// NewOAuthStateRepository 创建 OAuthStateRepository。
func NewOAuthStateRepository(client *ent.Client) *OAuthStateRepository {
	return &OAuthStateRepository{client: client}
}

// Create 创建 OAuth state。
func (r *OAuthStateRepository) Create(ctx context.Context, state *oauthstate.OAuthState) error {
	client := entdb.ClientFromCtx(ctx, r.client)
	return client.OAuthLoginState.Create().
		SetStateHash(state.StateHash).
		SetProvider(state.Provider).
		SetRedirectURL(state.RedirectURL).
		SetExpiresAt(state.ExpiresAt).
		SetNillableConsumedAt(state.ConsumedAt).
		SetCreatedAt(state.CreatedAt).
		SetUpdatedAt(state.UpdatedAt).
		Exec(ctx)
}

// Consume 消费 OAuth state。
func (r *OAuthStateRepository) Consume(ctx context.Context, provider string, stateHash string, now time.Time) (*oauthstate.OAuthState, error) {
	client := entdb.ClientFromCtx(ctx, r.client)
	row, err := client.OAuthLoginState.Query().
		Where(entoauthstate.Provider(provider), entoauthstate.StateHash(stateHash), entoauthstate.ConsumedAtIsNil(), entoauthstate.ExpiresAtGT(now)).
		Only(ctx)
	if err != nil {
		return nil, mapEntNotFound(err, oauthstate.ErrNotFound)
	}
	if err := client.OAuthLoginState.UpdateOne(row).SetConsumedAt(now).SetUpdatedAt(now).Exec(ctx); err != nil {
		return nil, err
	}
	row.ConsumedAt = &now
	return toOAuthState(row), nil
}

func toOAuthState(row *ent.OAuthLoginState) *oauthstate.OAuthState {
	return &oauthstate.OAuthState{
		StateHash:   row.StateHash,
		Provider:    row.Provider,
		RedirectURL: row.RedirectURL,
		ExpiresAt:   row.ExpiresAt,
		ConsumedAt:  row.ConsumedAt,
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
	}
}
