package service

import (
	"context"
	stderrors "errors"
	"strings"
	"testing"

	usercmd "github.com/maguowei/gotobeta/internal/modules/user/application/command"
	userquery "github.com/maguowei/gotobeta/internal/modules/user/application/query"
	userresult "github.com/maguowei/gotobeta/internal/modules/user/application/result"
	"github.com/maguowei/gotobeta/internal/modules/user/domain/identity"
	"github.com/maguowei/gotobeta/internal/modules/user/domain/oauthstate"
	userdomain "github.com/maguowei/gotobeta/internal/modules/user/domain/user"
)

// errInfra 模拟仓储/基础设施返回的底层错误，用于命中 apperr.WrapInternal 分支。
var errInfra = stderrors.New("infra failure")

// registerDisabledUser 注册一个用户并把其状态改为 disabled，用于命中 EnsureCanLogin 的 Forbidden 分支。
func registerDisabledUser(t *testing.T, svc *AuthService, store *fakeStore, email string) int64 {
	t.Helper()
	out, err := svc.Register(t.Context(), usercmd.RegisterCommand{Email: email, Password: "password-123", DisplayName: "User"})
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	original := store.users[out.User.ID]
	disabled := userdomain.UnmarshalFromDB(userdomain.FromDB{
		ID:              original.ID(),
		Email:           original.Email(),
		EmailNormalized: original.EmailNormalized(),
		PasswordHash:    original.PasswordHash(),
		PasswordHashAlg: "test",
		DisplayName:     original.DisplayName(),
		Status:          userdomain.StatusDisabled,
		CreatedAt:       testNow,
		UpdatedAt:       testNow,
	})
	store.users[out.User.ID] = disabled
	return out.User.ID
}

func TestAuthServiceRegisterValidationAndConflict(t *testing.T) {
	tests := []struct {
		name string
		cmd  usercmd.RegisterCommand
	}{
		{name: "invalid email", cmd: usercmd.RegisterCommand{Email: "not-an-email", Password: "password-123"}},
		{name: "short password", cmd: usercmd.RegisterCommand{Email: "a@b.com", Password: "short"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc, _, _, _, _ := newTestAuthService()
			if _, err := svc.Register(t.Context(), tc.cmd); err == nil {
				t.Fatalf("Register() error = nil, want validation error")
			}
		})
	}

	t.Run("email already registered", func(t *testing.T) {
		svc, _, _, _, _ := newTestAuthService()
		cmd := usercmd.RegisterCommand{Email: "dup@example.com", Password: "password-123", DisplayName: "Dup"}
		if _, err := svc.Register(t.Context(), cmd); err != nil {
			t.Fatalf("Register() error = %v", err)
		}
		_, err := svc.Register(t.Context(), cmd)
		if err == nil || !strings.Contains(err.Error(), "邮箱已注册") {
			t.Fatalf("Register() error = %v, want 邮箱已注册", err)
		}
	})
}

func TestAuthServiceLoginErrorPaths(t *testing.T) {
	t.Run("invalid email", func(t *testing.T) {
		svc, _, _, _, _ := newTestAuthService()
		if _, err := svc.Login(t.Context(), usercmd.LoginCommand{Email: "bad", Password: "password-123"}); err == nil {
			t.Fatalf("Login() error = nil, want invalid email")
		}
	})

	t.Run("user not found", func(t *testing.T) {
		svc, _, _, _, _ := newTestAuthService()
		_, err := svc.Login(t.Context(), usercmd.LoginCommand{Email: "missing@example.com", Password: "password-123"})
		if err == nil || !strings.Contains(err.Error(), "邮箱或密码不正确") {
			t.Fatalf("Login() error = %v, want unauthorized", err)
		}
	})

	t.Run("wrong password", func(t *testing.T) {
		svc, _, _, _, _ := newTestAuthService()
		if _, err := svc.Register(t.Context(), usercmd.RegisterCommand{Email: "p@example.com", Password: "password-123"}); err != nil {
			t.Fatalf("Register() error = %v", err)
		}
		_, err := svc.Login(t.Context(), usercmd.LoginCommand{Email: "p@example.com", Password: "wrong-password"})
		if err == nil || !strings.Contains(err.Error(), "邮箱或密码不正确") {
			t.Fatalf("Login() error = %v, want unauthorized", err)
		}
	})

	t.Run("disabled account", func(t *testing.T) {
		svc, store, _, _, _ := newTestAuthService()
		registerDisabledUser(t, svc, store, "disabled@example.com")
		_, err := svc.Login(t.Context(), usercmd.LoginCommand{Email: "disabled@example.com", Password: "password-123"})
		if err == nil || !strings.Contains(err.Error(), "账号已停用") {
			t.Fatalf("Login() error = %v, want forbidden", err)
		}
	})
}

func TestAuthServiceRefreshErrorPaths(t *testing.T) {
	t.Run("invalid refresh token", func(t *testing.T) {
		svc, _, _, _, _ := newTestAuthService()
		_, err := svc.Refresh(t.Context(), usercmd.RefreshTokenCommand{RefreshToken: "unknown"})
		if err == nil || !strings.Contains(err.Error(), "refresh token 无效") {
			t.Fatalf("Refresh() error = %v, want invalid", err)
		}
	})

	t.Run("disabled account", func(t *testing.T) {
		svc, store, _, _, _ := newTestAuthService()
		out, err := svc.Register(t.Context(), usercmd.RegisterCommand{Email: "rd@example.com", Password: "password-123"})
		if err != nil {
			t.Fatalf("Register() error = %v", err)
		}
		// 把用户改为 disabled，但保留已签发的 refresh token。
		disabled := userdomain.UnmarshalFromDB(userdomain.FromDB{
			ID:              out.User.ID,
			Email:           "rd@example.com",
			EmailNormalized: "rd@example.com",
			Status:          userdomain.StatusDisabled,
			CreatedAt:       testNow,
			UpdatedAt:       testNow,
		})
		store.users[out.User.ID] = disabled
		_, err = svc.Refresh(t.Context(), usercmd.RefreshTokenCommand{RefreshToken: out.Tokens.RefreshToken})
		if err == nil || !strings.Contains(err.Error(), "账号已停用") {
			t.Fatalf("Refresh() error = %v, want forbidden", err)
		}
	})
}

func TestAuthServiceLogoutMissingTokenIsNoop(t *testing.T) {
	svc, _, _, _, _ := newTestAuthService()
	if err := svc.Logout(t.Context(), usercmd.LogoutCommand{RefreshToken: "unknown"}); err != nil {
		t.Fatalf("Logout() error = %v, want nil noop", err)
	}
}

func TestAuthServiceUpdateProfileUserNotFound(t *testing.T) {
	svc, _, _, _, _ := newTestAuthService()
	_, err := svc.UpdateProfile(t.Context(), usercmd.UpdateProfileCommand{UserID: 404, DisplayName: "X"})
	if err == nil || !strings.Contains(err.Error(), "用户不存在") {
		t.Fatalf("UpdateProfile() error = %v, want not found", err)
	}
}

func TestAuthServiceChangePasswordErrorPaths(t *testing.T) {
	t.Run("weak new password", func(t *testing.T) {
		svc, _, _, _, _ := newTestAuthService()
		err := svc.ChangePassword(t.Context(), usercmd.ChangePasswordCommand{UserID: 1, NewPassword: "short"})
		if err == nil {
			t.Fatalf("ChangePassword() error = nil, want validation error")
		}
	})

	t.Run("user not found", func(t *testing.T) {
		svc, _, _, _, _ := newTestAuthService()
		err := svc.ChangePassword(t.Context(), usercmd.ChangePasswordCommand{UserID: 404, NewPassword: "password-123"})
		if err == nil || !strings.Contains(err.Error(), "用户不存在") {
			t.Fatalf("ChangePassword() error = %v, want not found", err)
		}
	})

	t.Run("wrong old password", func(t *testing.T) {
		svc, _, _, _, _ := newTestAuthService()
		out, err := svc.Register(t.Context(), usercmd.RegisterCommand{Email: "cp@example.com", Password: "password-123"})
		if err != nil {
			t.Fatalf("Register() error = %v", err)
		}
		err = svc.ChangePassword(t.Context(), usercmd.ChangePasswordCommand{UserID: out.User.ID, OldPassword: "wrong", NewPassword: "password-456"})
		if err == nil || !strings.Contains(err.Error(), "原密码不正确") {
			t.Fatalf("ChangePassword() error = %v, want unauthorized", err)
		}
	})
}

func TestAuthServiceForgotPasswordErrorPaths(t *testing.T) {
	t.Run("invalid email", func(t *testing.T) {
		svc, _, _, _, _ := newTestAuthService()
		if err := svc.ForgotPassword(t.Context(), usercmd.ForgotPasswordCommand{Email: "bad"}); err == nil {
			t.Fatalf("ForgotPassword() error = nil, want invalid email")
		}
	})

	t.Run("unknown email is silent noop", func(t *testing.T) {
		svc, _, email, _, _ := newTestAuthService()
		if err := svc.ForgotPassword(t.Context(), usercmd.ForgotPasswordCommand{Email: "ghost@example.com"}); err != nil {
			t.Fatalf("ForgotPassword() error = %v, want nil", err)
		}
		if email.passwordResetToken != "" {
			t.Fatalf("ForgotPassword() leaked reset token for unknown email")
		}
	})
}

func TestAuthServiceResetPasswordErrorPaths(t *testing.T) {
	t.Run("weak new password", func(t *testing.T) {
		svc, _, _, _, _ := newTestAuthService()
		if err := svc.ResetPassword(t.Context(), usercmd.ResetPasswordCommand{Token: "t", NewPassword: "short"}); err == nil {
			t.Fatalf("ResetPassword() error = nil, want validation error")
		}
	})

	t.Run("invalid token", func(t *testing.T) {
		svc, _, _, _, _ := newTestAuthService()
		err := svc.ResetPassword(t.Context(), usercmd.ResetPasswordCommand{Token: "unknown", NewPassword: "password-123"})
		if err == nil || !strings.Contains(err.Error(), "密码重置 token 无效") {
			t.Fatalf("ResetPassword() error = %v, want invalid token", err)
		}
	})
}

func TestAuthServiceSendEmailVerificationUserNotFound(t *testing.T) {
	svc, _, _, _, _ := newTestAuthService()
	err := svc.SendEmailVerification(t.Context(), 404)
	if err == nil || !strings.Contains(err.Error(), "用户不存在") {
		t.Fatalf("SendEmailVerification() error = %v, want not found", err)
	}
}

func TestAuthServiceVerifyEmailInvalidToken(t *testing.T) {
	svc, _, _, _, _ := newTestAuthService()
	err := svc.VerifyEmail(t.Context(), usercmd.VerifyEmailCommand{Token: "unknown"})
	if err == nil || !strings.Contains(err.Error(), "邮箱验证 token 无效") {
		t.Fatalf("VerifyEmail() error = %v, want invalid token", err)
	}
}

func TestAuthServiceStartOAuthErrorPaths(t *testing.T) {
	t.Run("unsupported provider", func(t *testing.T) {
		svc, _, _, _, _ := newTestAuthService()
		_, err := svc.StartOAuth(t.Context(), usercmd.StartOAuthCommand{Provider: "unknown"})
		if err == nil || !strings.Contains(err.Error(), "不支持的 OAuth provider") {
			t.Fatalf("StartOAuth() error = %v, want unsupported provider", err)
		}
	})
}

func TestAuthServiceHandleOAuthCallbackErrorPaths(t *testing.T) {
	t.Run("unsupported provider", func(t *testing.T) {
		svc, _, _, _, _ := newTestAuthService()
		_, err := svc.HandleOAuthCallback(t.Context(), usercmd.HandleOAuthCallbackCommand{Provider: "unknown", State: "s", Code: "c"})
		if err == nil || !strings.Contains(err.Error(), "不支持的 OAuth provider") {
			t.Fatalf("HandleOAuthCallback() error = %v, want unsupported provider", err)
		}
	})

	t.Run("invalid state", func(t *testing.T) {
		svc, _, _, providers, _ := newTestAuthService()
		providers.items[identity.ProviderGitHub] = &fakeOAuthProvider{
			authURL: "https://github.com/login/oauth/authorize",
			profile: &oauthstate.Profile{Provider: identity.ProviderGitHub, ProviderUserID: "gh-1", Email: "x@example.com", EmailVerified: true},
		}
		_, err := svc.HandleOAuthCallback(t.Context(), usercmd.HandleOAuthCallbackCommand{Provider: identity.ProviderGitHub, State: "never-issued", Code: "c"})
		if err == nil || !strings.Contains(err.Error(), "OAuth state 无效") {
			t.Fatalf("HandleOAuthCallback() error = %v, want invalid state", err)
		}
	})

	t.Run("exchange failure", func(t *testing.T) {
		svc, _, _, _, _ := newTestAuthService()
		// provider 的 Exchange 始终失败，但 StartOAuth/AuthCodeURL 正常，以便先拿到合法 state。
		svc.oauthProviders = &errExchangeProviders{provider: &errExchangeProvider{authURL: "https://github.com/login/oauth/authorize", err: errInfra}}
		start, err := svc.StartOAuth(t.Context(), usercmd.StartOAuthCommand{Provider: identity.ProviderGitHub})
		if err != nil {
			t.Fatalf("StartOAuth() error = %v", err)
		}
		_, err = svc.HandleOAuthCallback(t.Context(), usercmd.HandleOAuthCallbackCommand{Provider: identity.ProviderGitHub, State: start.State, Code: "c"})
		if err == nil || !strings.Contains(err.Error(), "OAuth 换取用户资料失败") {
			t.Fatalf("HandleOAuthCallback() error = %v, want exchange failure", err)
		}
	})
}

func TestAuthServiceExchangeOAuthLoginCodeInvalidCode(t *testing.T) {
	svc, _, _, _, _ := newTestAuthService()
	_, err := svc.ExchangeOAuthLoginCode(t.Context(), usercmd.ExchangeOAuthLoginCodeCommand{Code: "unknown"})
	if err == nil || !strings.Contains(err.Error(), "OAuth 登录码无效") {
		t.Fatalf("ExchangeOAuthLoginCode() error = %v, want invalid code", err)
	}
}

func TestAuthServiceUnlinkIdentityCountError(t *testing.T) {
	store := newFakeStore()
	svc := newTestAuthServiceWithUsers(&errUserRepo{fakeUserRepo: fakeUserRepo{s: store}, countErr: errInfra})
	err := svc.UnlinkIdentity(t.Context(), usercmd.UnlinkIdentityCommand{UserID: 1, Provider: identity.ProviderGitHub})
	if err == nil || !strings.Contains(err.Error(), "查询登录方式失败") {
		t.Fatalf("UnlinkIdentity() error = %v, want infra error", err)
	}
}

func TestAuthServiceCurrentUserErrorPaths(t *testing.T) {
	t.Run("not found", func(t *testing.T) {
		svc, _, _, _, _ := newTestAuthService()
		_, err := svc.CurrentUser(t.Context(), userquery.GetCurrentUserQuery{UserID: 404})
		if err == nil || !strings.Contains(err.Error(), "用户不存在") {
			t.Fatalf("CurrentUser() error = %v, want not found", err)
		}
	})

	t.Run("infra error", func(t *testing.T) {
		store := newFakeStore()
		svc := newTestAuthServiceWithUsers(&errUserRepo{fakeUserRepo: fakeUserRepo{s: store}, findByIDErr: errInfra})
		_, err := svc.CurrentUser(t.Context(), userquery.GetCurrentUserQuery{UserID: 1})
		if err == nil || !strings.Contains(err.Error(), "查询用户失败") {
			t.Fatalf("CurrentUser() error = %v, want infra error", err)
		}
	})
}

func TestAuthServiceListIdentitiesInfraError(t *testing.T) {
	svc := newTestAuthServiceWithIdentities(&errIdentityRepo{listErr: errInfra})
	_, err := svc.ListIdentities(t.Context(), userquery.ListIdentitiesQuery{UserID: 1})
	if err == nil || !strings.Contains(err.Error(), "查询第三方身份失败") {
		t.Fatalf("ListIdentities() error = %v, want infra error", err)
	}
}

// TestMapUserLookupError 验证用户查找错误映射：not found -> NotFound，其余 -> Internal。
func TestMapUserLookupError(t *testing.T) {
	if err := mapUserLookupError(userdomain.ErrNotFound); err == nil || !strings.Contains(err.Error(), "用户不存在") {
		t.Fatalf("mapUserLookupError(ErrNotFound) = %v, want NotFound", err)
	}
	if err := mapUserLookupError(errInfra); err == nil || strings.Contains(err.Error(), "用户不存在") {
		t.Fatalf("mapUserLookupError(infra) = %v, want infra wrap", err)
	}
}

// TestAllowedOAuthRedirectURL 覆盖 redirect URL 校验的允许与拒绝分支。
func TestAllowedOAuthRedirectURL(t *testing.T) {
	const configured = "https://app.example.com/auth/success"
	tests := []struct {
		name      string
		requested string
		wantErr   bool
	}{
		{name: "empty uses configured", requested: "", wantErr: false},
		{name: "matching allowed", requested: configured, wantErr: false},
		{name: "different host rejected", requested: "https://evil.example/auth/success", wantErr: true},
		{name: "malformed rejected", requested: "://nope", wantErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := allowedOAuthRedirectURL(tc.requested, configured)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("allowedOAuthRedirectURL() error = nil, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("allowedOAuthRedirectURL() error = %v", err)
			}
			if got != configured {
				t.Fatalf("allowedOAuthRedirectURL() = %q, want %q", got, configured)
			}
		})
	}
}

// TestAllowedOAuthRedirectURLRejectsInvalidConfig 验证配置本身非法时返回 Internal 错误。
func TestAllowedOAuthRedirectURLRejectsInvalidConfig(t *testing.T) {
	_, err := allowedOAuthRedirectURL("", "not-a-valid-url")
	if err == nil {
		t.Fatalf("allowedOAuthRedirectURL() error = nil, want internal error for invalid config")
	}
}

// TestAppendLoginCode 覆盖已有 query 与无 query 两种拼接分支。
func TestAppendLoginCode(t *testing.T) {
	tests := []struct {
		name string
		base string
		want string
	}{
		{name: "no query", base: "https://app.example.com/cb", want: "https://app.example.com/cb?code=abc"},
		{name: "existing query", base: "https://app.example.com/cb?next=1", want: "https://app.example.com/cb?next=1&code=abc"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := appendLoginCode(tc.base, "abc"); got != tc.want {
				t.Fatalf("appendLoginCode() = %q, want %q", got, tc.want)
			}
		})
	}
}

// TestValidatePassword 覆盖密码长度校验的边界。
func TestValidatePassword(t *testing.T) {
	if err := validatePassword("  pwd  "); err == nil {
		t.Fatalf("validatePassword() error = nil, want too short")
	}
	if err := validatePassword("password-123"); err != nil {
		t.Fatalf("validatePassword() error = %v, want nil", err)
	}
}

// newTestAuthServiceWithUsers 用自定义 user 仓储构造服务，其余依赖沿用标准 fake，便于注入仓储错误。
func newTestAuthServiceWithUsers(users userdomain.Repository) *AuthService {
	svc, _, _, _, _ := newTestAuthService()
	svc.repos.Users = users
	return svc
}

// newTestAuthServiceWithIdentities 用自定义 identity 仓储构造服务。
func newTestAuthServiceWithIdentities(identities identity.Repository) *AuthService {
	svc, _, _, _, _ := newTestAuthService()
	svc.repos.Identities = identities
	return svc
}

// errUserRepo 在标准 fakeUserRepo 基础上注入可选错误。
type errUserRepo struct {
	fakeUserRepo
	countErr    error
	findByIDErr error
}

func (r *errUserRepo) CountLoginMethods(ctx context.Context, userID int64) (int, error) {
	if r.countErr != nil {
		return 0, r.countErr
	}
	return r.fakeUserRepo.CountLoginMethods(ctx, userID)
}

func (r *errUserRepo) FindUserByID(ctx context.Context, id int64) (*userdomain.User, error) {
	if r.findByIDErr != nil {
		return nil, r.findByIDErr
	}
	return r.fakeUserRepo.FindUserByID(ctx, id)
}

// errIdentityRepo 注入 List 错误，其余方法保持空实现（本测试不调用）。
type errIdentityRepo struct {
	listErr error
}

var _ identity.Repository = (*errIdentityRepo)(nil)

func (r *errIdentityRepo) Find(context.Context, string, string) (*identity.Identity, error) {
	return nil, identity.ErrNotFound
}
func (r *errIdentityRepo) Upsert(context.Context, *identity.Identity) error { return nil }
func (r *errIdentityRepo) List(context.Context, int64) ([]*identity.Identity, error) {
	return nil, r.listErr
}
func (r *errIdentityRepo) Delete(context.Context, int64, string) error { return nil }

// TestAuthServiceOAuthReloginUsesExistingIdentity 覆盖 loginOAuthProfile 中“已存在第三方身份直接登录”的分支。
func TestAuthServiceOAuthReloginUsesExistingIdentity(t *testing.T) {
	svc, repo, _, providers, _ := newTestAuthService()
	providers.items[identity.ProviderGitHub] = &fakeOAuthProvider{
		authURL: "https://github.com/login/oauth/authorize",
		profile: &oauthstate.Profile{
			Provider:       identity.ProviderGitHub,
			ProviderUserID: "gh-relogin",
			Email:          "relogin@example.com",
			EmailVerified:  true,
			DisplayName:    "Relogin",
		},
	}

	// 第一次回调：创建用户与身份。
	runOAuthCallback(t, svc, identity.ProviderGitHub)
	if got := len(repo.identityBySubject); got != 1 {
		t.Fatalf("identity count after first login = %d, want 1", got)
	}

	// 第二次回调：identity 已存在，走 Find 成功 -> FindUserByID 分支，不应新建用户。
	usersBefore := len(repo.users)
	authOut := runOAuthCallback(t, svc, identity.ProviderGitHub)
	if authOut.User.Email != "relogin@example.com" {
		t.Fatalf("re-login user = %+v", authOut.User)
	}
	if len(repo.users) != usersBefore {
		t.Fatalf("re-login created a new user: before=%d after=%d", usersBefore, len(repo.users))
	}
}

// TestAuthServiceTokenGenerationFailurePropagates 覆盖 issueSessionWithID / createActionToken
// 在 secrets.NewToken 失败时返回 Internal 错误的分支。
func TestAuthServiceTokenGenerationFailurePropagates(t *testing.T) {
	svc, _, _, _, _ := newTestAuthService()
	svc.secrets = failingSecrets{}
	_, err := svc.Register(t.Context(), usercmd.RegisterCommand{Email: "tok@example.com", Password: "password-123"})
	if err == nil {
		t.Fatalf("Register() error = nil, want token generation failure")
	}
}

// runOAuthCallback 执行一次完整 StartOAuth + HandleOAuthCallback + ExchangeOAuthLoginCode，返回最终鉴权结果。
func runOAuthCallback(t *testing.T, svc *AuthService, provider string) *userresult.AuthResult {
	t.Helper()
	start, err := svc.StartOAuth(t.Context(), usercmd.StartOAuthCommand{Provider: provider})
	if err != nil {
		t.Fatalf("StartOAuth() error = %v", err)
	}
	callback, err := svc.HandleOAuthCallback(t.Context(), usercmd.HandleOAuthCallbackCommand{Provider: provider, State: start.State, Code: "c"})
	if err != nil {
		t.Fatalf("HandleOAuthCallback() error = %v", err)
	}
	authOut, err := svc.ExchangeOAuthLoginCode(t.Context(), usercmd.ExchangeOAuthLoginCodeCommand{Code: callback.LoginCode})
	if err != nil {
		t.Fatalf("ExchangeOAuthLoginCode() error = %v", err)
	}
	return authOut
}

// failingSecrets 的 NewToken 始终失败，用于命中 token 生成错误分支。
type failingSecrets struct{}

func (failingSecrets) NewToken() (string, error)     { return "", errInfra }
func (failingSecrets) HashToken(token string) string { return "hash:" + token }

// errExchangeProviders 总是返回一个在 Exchange 阶段失败的 provider。
type errExchangeProviders struct {
	provider *errExchangeProvider
}

func (p *errExchangeProviders) Get(string) (interface {
	AuthCodeURL(state string) string
	Exchange(ctx context.Context, code string) (*oauthstate.Profile, error)
}, bool) {
	return p.provider, true
}

// errExchangeProvider 的 Exchange 始终返回错误，AuthCodeURL 正常。
type errExchangeProvider struct {
	authURL string
	err     error
}

func (p *errExchangeProvider) AuthCodeURL(state string) string {
	return p.authURL + "?state=" + state
}

func (p *errExchangeProvider) Exchange(context.Context, string) (*oauthstate.Profile, error) {
	return nil, p.err
}
