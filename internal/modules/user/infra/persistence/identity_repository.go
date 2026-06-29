package persistence

import (
	"context"

	"github.com/maguowei/gotobeta/internal/ent"
	entidentity "github.com/maguowei/gotobeta/internal/ent/useridentity"
	"github.com/maguowei/gotobeta/internal/infra/entdb"
	identitydomain "github.com/maguowei/gotobeta/internal/modules/user/domain/identity"
)

// IdentityRepository 是第三方身份的 Ent 实现。
type IdentityRepository struct {
	client *ent.Client
}

var _ identitydomain.Repository = (*IdentityRepository)(nil)

// NewIdentityRepository 创建 IdentityRepository。
func NewIdentityRepository(client *ent.Client) *IdentityRepository {
	return &IdentityRepository{client: client}
}

// Find 查询第三方身份。
func (r *IdentityRepository) Find(ctx context.Context, provider string, providerUserID string) (*identitydomain.Identity, error) {
	client := entdb.ClientFromCtx(ctx, r.client)
	row, err := client.UserIdentity.Query().
		Where(entidentity.Provider(provider), entidentity.ProviderUserID(providerUserID)).
		Only(ctx)
	if err != nil {
		return nil, mapEntNotFound(err, identitydomain.ErrNotFound)
	}
	return toIdentity(row), nil
}

// Upsert 新增或更新第三方身份。
func (r *IdentityRepository) Upsert(ctx context.Context, identity *identitydomain.Identity) error {
	client := entdb.ClientFromCtx(ctx, r.client)
	existing, err := client.UserIdentity.Query().
		Where(entidentity.Provider(identity.Provider), entidentity.ProviderUserID(identity.ProviderUserID)).
		Only(ctx)
	if err != nil {
		if !ent.IsNotFound(err) {
			return err
		}
		return client.UserIdentity.Create().
			SetBizID(identity.ID).
			SetUserBizID(identity.UserID).
			SetProvider(identity.Provider).
			SetProviderUserID(identity.ProviderUserID).
			SetProviderEmail(identity.ProviderEmail).
			SetProviderEmailNormalized(identity.ProviderEmailNormalized).
			SetProviderEmailVerified(identity.ProviderEmailVerified).
			SetDisplayName(identity.DisplayName).
			SetAvatarURL(identity.AvatarURL).
			SetProfileURL(identity.ProfileURL).
			SetLinkedAt(identity.LinkedAt).
			SetNillableLastLoginAt(identity.LastLoginAt).
			SetCreatedAt(identity.CreatedAt).
			SetUpdatedAt(identity.UpdatedAt).
			Exec(ctx)
	}
	return client.UserIdentity.UpdateOne(existing).
		SetProviderEmail(identity.ProviderEmail).
		SetProviderEmailNormalized(identity.ProviderEmailNormalized).
		SetProviderEmailVerified(identity.ProviderEmailVerified).
		SetDisplayName(identity.DisplayName).
		SetAvatarURL(identity.AvatarURL).
		SetProfileURL(identity.ProfileURL).
		SetNillableLastLoginAt(identity.LastLoginAt).
		SetUpdatedAt(identity.UpdatedAt).
		Exec(ctx)
}

// List 列出用户第三方身份。
func (r *IdentityRepository) List(ctx context.Context, userID int64) ([]*identitydomain.Identity, error) {
	client := entdb.ClientFromCtx(ctx, r.client)
	rows, err := client.UserIdentity.Query().Where(entidentity.UserBizID(userID)).All(ctx)
	if err != nil {
		return nil, err
	}
	items := make([]*identitydomain.Identity, 0, len(rows))
	for _, row := range rows {
		items = append(items, toIdentity(row))
	}
	return items, nil
}

// Delete 删除用户第三方身份。
func (r *IdentityRepository) Delete(ctx context.Context, userID int64, provider string) error {
	client := entdb.ClientFromCtx(ctx, r.client)
	affected, err := client.UserIdentity.Delete().
		Where(entidentity.UserBizID(userID), entidentity.Provider(provider)).
		Exec(ctx)
	if err != nil {
		return err
	}
	if affected == 0 {
		return identitydomain.ErrNotFound
	}
	return nil
}

func toIdentity(row *ent.UserIdentity) *identitydomain.Identity {
	return &identitydomain.Identity{
		ID:                      row.BizID,
		UserID:                  row.UserBizID,
		Provider:                row.Provider,
		ProviderUserID:          row.ProviderUserID,
		ProviderEmail:           row.ProviderEmail,
		ProviderEmailNormalized: row.ProviderEmailNormalized,
		ProviderEmailVerified:   row.ProviderEmailVerified,
		DisplayName:             row.DisplayName,
		AvatarURL:               row.AvatarURL,
		ProfileURL:              row.ProfileURL,
		LinkedAt:                row.LinkedAt,
		LastLoginAt:             row.LastLoginAt,
		CreatedAt:               row.CreatedAt,
		UpdatedAt:               row.UpdatedAt,
	}
}
