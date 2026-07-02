package service

import (
	"context"
	stderrors "errors"

	userquery "github.com/maguowei/gotobeta/internal/modules/user/application/query"
	userresult "github.com/maguowei/gotobeta/internal/modules/user/application/result"
	"github.com/maguowei/gotobeta/internal/modules/user/domain/identity"
	userdomain "github.com/maguowei/gotobeta/internal/modules/user/domain/user"
	"github.com/maguowei/gotobeta/internal/pkg/apperr"
)

// CurrentUser 查询当前用户。
func (s *AuthService) CurrentUser(ctx context.Context, query userquery.GetCurrentUserQuery) (*userresult.UserResult, error) {
	u, err := s.repos.Users.FindUserByID(ctx, query.UserID)
	if err != nil {
		if stderrors.Is(err, userdomain.ErrNotFound) {
			return nil, apperr.NotFound("用户不存在")
		}
		return nil, apperr.WrapInternal("查询用户失败", err)
	}
	return toUserResult(u), nil
}

// ListIdentities 列出当前用户绑定的第三方身份。
func (s *AuthService) ListIdentities(ctx context.Context, query userquery.ListIdentitiesQuery) ([]*userresult.IdentityResult, error) {
	items, err := s.repos.Identities.List(ctx, query.UserID)
	if err != nil {
		return nil, apperr.WrapInternal("查询第三方身份失败", err)
	}
	out := make([]*userresult.IdentityResult, 0, len(items))
	for _, item := range items {
		out = append(out, toIdentityResult(item))
	}
	return out, nil
}

func toIdentityResult(ident *identity.Identity) *userresult.IdentityResult {
	return &userresult.IdentityResult{
		Provider:      ident.Provider,
		ProviderEmail: ident.ProviderEmail,
		DisplayName:   ident.DisplayName,
		ProfileURL:    ident.ProfileURL,
		LinkedAt:      ident.LinkedAt,
		LastLoginAt:   ident.LastLoginAt,
	}
}
