package persistence

import (
	"context"
	"time"

	"github.com/maguowei/gotobeta/internal/ent"
	entuser "github.com/maguowei/gotobeta/internal/ent/user"
	entidentity "github.com/maguowei/gotobeta/internal/ent/useridentity"
	"github.com/maguowei/gotobeta/internal/infra/entdb"
	userdomain "github.com/maguowei/gotobeta/internal/modules/user/domain/user"
)

// UserRepository 是用户聚合的 Ent 实现。
type UserRepository struct {
	client *ent.Client
}

var _ userdomain.Repository = (*UserRepository)(nil)

// NewUserRepository 创建 UserRepository。
func NewUserRepository(client *ent.Client) *UserRepository {
	return &UserRepository{client: client}
}

// CreateUser 创建用户。
func (r *UserRepository) CreateUser(ctx context.Context, user *userdomain.User) error {
	client := entdb.ClientFromCtx(ctx, r.client)
	return client.User.Create().
		SetBizID(user.ID()).
		SetEmail(user.Email()).
		SetEmailNormalized(user.EmailNormalized()).
		SetNillableEmailVerifiedAt(user.EmailVerifiedAt()).
		SetNillablePasswordHash(optionalString(user.PasswordHash())).
		SetNillablePasswordHashAlg(optionalString(user.PasswordHashAlg())).
		SetNillablePasswordSetAt(user.PasswordSetAt()).
		SetDisplayName(user.DisplayName()).
		SetAvatarURL(user.AvatarURL()).
		SetStatus(string(user.Status())).
		SetNillableLastLoginAt(user.LastLoginAt()).
		SetCreatedAt(user.CreatedAt()).
		SetUpdatedAt(user.UpdatedAt()).
		Exec(ctx)
}

// FindUserByID 按业务 ID 查询用户。
func (r *UserRepository) FindUserByID(ctx context.Context, id int64) (*userdomain.User, error) {
	client := entdb.ClientFromCtx(ctx, r.client)
	row, err := client.User.Query().Where(entuser.BizID(id)).Only(ctx)
	if err != nil {
		return nil, mapEntNotFound(err, userdomain.ErrNotFound)
	}
	return toUser(row), nil
}

// FindUserByEmail 按归一化邮箱查询用户。
func (r *UserRepository) FindUserByEmail(ctx context.Context, normalizedEmail string) (*userdomain.User, error) {
	client := entdb.ClientFromCtx(ctx, r.client)
	row, err := client.User.Query().Where(entuser.EmailNormalized(normalizedEmail)).Only(ctx)
	if err != nil {
		return nil, mapEntNotFound(err, userdomain.ErrNotFound)
	}
	return toUser(row), nil
}

// SaveUser 保存用户变更。
func (r *UserRepository) SaveUser(ctx context.Context, user *userdomain.User) error {
	client := entdb.ClientFromCtx(ctx, r.client)
	affected, err := client.User.Update().
		Where(entuser.BizID(user.ID())).
		SetEmail(user.Email()).
		SetEmailNormalized(user.EmailNormalized()).
		SetNillableEmailVerifiedAt(user.EmailVerifiedAt()).
		SetNillablePasswordHash(optionalString(user.PasswordHash())).
		SetNillablePasswordHashAlg(optionalString(user.PasswordHashAlg())).
		SetNillablePasswordSetAt(user.PasswordSetAt()).
		SetDisplayName(user.DisplayName()).
		SetAvatarURL(user.AvatarURL()).
		SetStatus(string(user.Status())).
		SetNillableLastLoginAt(user.LastLoginAt()).
		SetUpdatedAt(user.UpdatedAt()).
		Save(ctx)
	if err != nil {
		return err
	}
	if affected == 0 {
		return userdomain.ErrNotFound
	}
	return nil
}

// UpdateUserLastLogin 更新用户最近登录时间。
func (r *UserRepository) UpdateUserLastLogin(ctx context.Context, userID int64, now time.Time) error {
	client := entdb.ClientFromCtx(ctx, r.client)
	affected, err := client.User.Update().
		Where(entuser.BizID(userID)).
		SetLastLoginAt(now).
		SetUpdatedAt(now).
		Save(ctx)
	if err != nil {
		return err
	}
	if affected == 0 {
		return userdomain.ErrNotFound
	}
	return nil
}

// CountLoginMethods 返回用户可用登录方式数量。
func (r *UserRepository) CountLoginMethods(ctx context.Context, userID int64) (int, error) {
	user, err := r.FindUserByID(ctx, userID)
	if err != nil {
		return 0, err
	}
	count := 0
	if user.HasPassword() {
		count++
	}
	client := entdb.ClientFromCtx(ctx, r.client)
	identityCount, err := client.UserIdentity.Query().Where(entidentity.UserBizID(userID)).Count(ctx)
	if err != nil {
		return 0, err
	}
	return count + identityCount, nil
}

func toUser(row *ent.User) *userdomain.User {
	return userdomain.UnmarshalFromDB(userdomain.FromDB{
		ID:              row.BizID,
		Email:           row.Email,
		EmailNormalized: row.EmailNormalized,
		EmailVerifiedAt: row.EmailVerifiedAt,
		PasswordHash:    valueString(row.PasswordHash),
		PasswordHashAlg: valueString(row.PasswordHashAlg),
		PasswordSetAt:   row.PasswordSetAt,
		DisplayName:     row.DisplayName,
		AvatarURL:       row.AvatarURL,
		Status:          userdomain.Status(row.Status),
		LastLoginAt:     row.LastLoginAt,
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
	})
}
