package service

import (
	"context"
	stderrors "errors"
	"fmt"
	"log/slog"
	"strings"
	"testing"
	"time"

	usercmd "github.com/maguowei/gotobeta/internal/modules/user/application/command"
	userquery "github.com/maguowei/gotobeta/internal/modules/user/application/query"
	"github.com/maguowei/gotobeta/internal/modules/user/domain/actiontoken"
	"github.com/maguowei/gotobeta/internal/modules/user/domain/identity"
	"github.com/maguowei/gotobeta/internal/modules/user/domain/oauthstate"
	"github.com/maguowei/gotobeta/internal/modules/user/domain/refreshtoken"
	userdomain "github.com/maguowei/gotobeta/internal/modules/user/domain/user"
)

var testNow = time.Date(2026, 5, 30, 12, 0, 0, 0, time.UTC)

func TestAuthServiceRegisterLoginRefreshAndLogout(t *testing.T) {
	svc, repo, _, _, _ := newTestAuthService()

	registered, err := svc.Register(t.Context(), usercmd.RegisterCommand{
		Email:       " Alice@Example.COM ",
		Password:    "password-123",
		DisplayName: "Alice",
	})
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	if registered.User.ID == 0 || registered.Tokens.AccessToken == "" || registered.Tokens.RefreshToken == "" {
		t.Fatalf("Register() output = %+v", registered)
	}
	if len(repo.actionByHash) != 1 {
		t.Fatalf("action token count = %d, want 1", len(repo.actionByHash))
	}

	loggedIn, err := svc.Login(t.Context(), usercmd.LoginCommand{
		Email:    "alice@example.com",
		Password: "password-123",
	})
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if loggedIn.Tokens.RefreshToken == registered.Tokens.RefreshToken {
		t.Fatalf("Login() reused refresh token")
	}

	refreshed, err := svc.Refresh(t.Context(), usercmd.RefreshTokenCommand{RefreshToken: loggedIn.Tokens.RefreshToken})
	if err != nil {
		t.Fatalf("Refresh() error = %v", err)
	}
	if refreshed.Tokens.RefreshToken == loggedIn.Tokens.RefreshToken {
		t.Fatalf("Refresh() did not rotate refresh token")
	}
	if repo.revokedReason != "rotated" || repo.replacedByTokenID == "" {
		t.Fatalf("refresh revoke reason=%q replacedBy=%q", repo.revokedReason, repo.replacedByTokenID)
	}

	if err := svc.Logout(t.Context(), usercmd.LogoutCommand{RefreshToken: refreshed.Tokens.RefreshToken}); err != nil {
		t.Fatalf("Logout() error = %v", err)
	}
	if repo.revokedReason != "logout" {
		t.Fatalf("Logout() revoke reason = %q, want logout", repo.revokedReason)
	}
}

// fakeTokenRevoker 记录被吊销的 jti 与 TTL。
type fakeTokenRevoker struct {
	revoked map[string]time.Duration
}

func (r *fakeTokenRevoker) Revoke(_ context.Context, jti string, ttl time.Duration) error {
	if r.revoked == nil {
		r.revoked = map[string]time.Duration{}
	}
	r.revoked[jti] = ttl
	return nil
}

func TestLogoutRevokesAccessToken(t *testing.T) {
	svc, _, _, _, _ := newTestAuthService()
	rev := &fakeTokenRevoker{}
	svc.tokenRevoker = rev

	err := svc.Logout(t.Context(), usercmd.LogoutCommand{
		RefreshToken:         "whatever",
		AccessTokenID:        "jti-123",
		AccessTokenExpiresAt: svc.now().Add(10 * time.Minute),
	})
	if err != nil {
		t.Fatalf("Logout() error = %v", err)
	}
	ttl, ok := rev.revoked["jti-123"]
	if !ok {
		t.Fatal("logout 应把 access token 的 jti 加入吊销黑名单")
	}
	if ttl <= 0 || ttl > 10*time.Minute {
		t.Fatalf("吊销 TTL = %s, 应为剩余有效期(<=10m)", ttl)
	}
}

func TestLogoutSkipsRevokeForExpiredAccessToken(t *testing.T) {
	svc, _, _, _, _ := newTestAuthService()
	rev := &fakeTokenRevoker{}
	svc.tokenRevoker = rev

	err := svc.Logout(t.Context(), usercmd.LogoutCommand{
		RefreshToken:         "whatever",
		AccessTokenID:        "jti-expired",
		AccessTokenExpiresAt: svc.now().Add(-time.Minute),
	})
	if err != nil {
		t.Fatalf("Logout() error = %v", err)
	}
	if _, ok := rev.revoked["jti-expired"]; ok {
		t.Fatal("已过期 access token 不应写入黑名单")
	}
}

func TestAuthServiceProfilePasswordAndEmailActions(t *testing.T) {
	svc, _, email, _, _ := newTestAuthService()
	out, err := svc.Register(t.Context(), usercmd.RegisterCommand{
		Email:       "bob@example.com",
		Password:    "password-123",
		DisplayName: "Bob",
	})
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	userID := out.User.ID

	current, err := svc.CurrentUser(t.Context(), userquery.GetCurrentUserQuery{UserID: userID})
	if err != nil {
		t.Fatalf("CurrentUser() error = %v", err)
	}
	if current.Email != "bob@example.com" {
		t.Fatalf("CurrentUser().Email = %q", current.Email)
	}

	updated, err := svc.UpdateProfile(t.Context(), usercmd.UpdateProfileCommand{
		UserID:      userID,
		DisplayName: "Bobby",
		AvatarURL:   "https://example.com/avatar.png",
	})
	if err != nil {
		t.Fatalf("UpdateProfile() error = %v", err)
	}
	if updated.DisplayName != "Bobby" || updated.AvatarURL == "" {
		t.Fatalf("UpdateProfile() = %+v", updated)
	}

	if err := svc.ChangePassword(t.Context(), usercmd.ChangePasswordCommand{
		UserID:      userID,
		OldPassword: "password-123",
		NewPassword: "password-456",
	}); err != nil {
		t.Fatalf("ChangePassword() error = %v", err)
	}
	if _, err := svc.Login(t.Context(), usercmd.LoginCommand{Email: "bob@example.com", Password: "password-123"}); err == nil {
		t.Fatalf("Login() with old password succeeded")
	}

	if err := svc.ForgotPassword(t.Context(), usercmd.ForgotPasswordCommand{Email: "bob@example.com"}); err != nil {
		t.Fatalf("ForgotPassword() error = %v", err)
	}
	if email.passwordResetToken == "" {
		t.Fatalf("password reset token was not sent")
	}
	if err := svc.ResetPassword(t.Context(), usercmd.ResetPasswordCommand{
		Token:       email.passwordResetToken,
		NewPassword: "password-789",
	}); err != nil {
		t.Fatalf("ResetPassword() error = %v", err)
	}
	if _, err := svc.Login(t.Context(), usercmd.LoginCommand{Email: "bob@example.com", Password: "password-789"}); err != nil {
		t.Fatalf("Login() after reset error = %v", err)
	}

	if err := svc.SendEmailVerification(t.Context(), userID); err != nil {
		t.Fatalf("SendEmailVerification() error = %v", err)
	}
	if email.verificationToken == "" {
		t.Fatalf("verification token was not sent")
	}
	if err := svc.VerifyEmail(t.Context(), usercmd.VerifyEmailCommand{Token: email.verificationToken}); err != nil {
		t.Fatalf("VerifyEmail() error = %v", err)
	}
	verified, err := svc.CurrentUser(t.Context(), userquery.GetCurrentUserQuery{UserID: userID})
	if err != nil {
		t.Fatalf("CurrentUser() after verify error = %v", err)
	}
	if !verified.EmailVerified {
		t.Fatalf("EmailVerified = false, want true")
	}
}

func TestAuthServiceOAuthFlowAndIdentities(t *testing.T) {
	svc, repo, _, providers, _ := newTestAuthService()
	providers.items[identity.ProviderGitHub] = &fakeOAuthProvider{
		authURL: "https://github.com/login/oauth/authorize",
		profile: &oauthstate.Profile{
			Provider:       identity.ProviderGitHub,
			ProviderUserID: "gh-42",
			Email:          "carol@example.com",
			EmailVerified:  true,
			DisplayName:    "Carol",
			AvatarURL:      "https://example.com/carol.png",
			ProfileURL:     "https://github.com/carol",
		},
	}

	start, err := svc.StartOAuth(t.Context(), usercmd.StartOAuthCommand{Provider: identity.ProviderGitHub})
	if err != nil {
		t.Fatalf("StartOAuth() error = %v", err)
	}
	if start.State == "" || !strings.Contains(start.AuthURL, "state=") {
		t.Fatalf("StartOAuth() = %+v", start)
	}

	callback, err := svc.HandleOAuthCallback(t.Context(), usercmd.HandleOAuthCallbackCommand{
		Provider: identity.ProviderGitHub,
		State:    start.State,
		Code:     "provider-code",
	})
	if err != nil {
		t.Fatalf("HandleOAuthCallback() error = %v", err)
	}
	if !strings.Contains(callback.RedirectURL, "code=") || callback.LoginCode == "" {
		t.Fatalf("HandleOAuthCallback() = %+v", callback)
	}

	authOut, err := svc.ExchangeOAuthLoginCode(t.Context(), usercmd.ExchangeOAuthLoginCodeCommand{Code: callback.LoginCode})
	if err != nil {
		t.Fatalf("ExchangeOAuthLoginCode() error = %v", err)
	}
	if authOut.User.Email != "carol@example.com" || !authOut.User.EmailVerified {
		t.Fatalf("ExchangeOAuthLoginCode() user = %+v", authOut.User)
	}

	identities, err := svc.ListIdentities(t.Context(), userquery.ListIdentitiesQuery{UserID: authOut.User.ID})
	if err != nil {
		t.Fatalf("ListIdentities() error = %v", err)
	}
	if len(identities) != 1 || identities[0].Provider != identity.ProviderGitHub {
		t.Fatalf("ListIdentities() = %+v", identities)
	}

	if err := svc.UnlinkIdentity(t.Context(), usercmd.UnlinkIdentityCommand{UserID: authOut.User.ID, Provider: identity.ProviderGitHub}); err == nil {
		t.Fatalf("UnlinkIdentity() succeeded with only one login method")
	}
	if err := svc.ChangePassword(t.Context(), usercmd.ChangePasswordCommand{UserID: authOut.User.ID, NewPassword: "password-123"}); err != nil {
		t.Fatalf("ChangePassword() for OAuth user error = %v", err)
	}
	if err := svc.UnlinkIdentity(t.Context(), usercmd.UnlinkIdentityCommand{UserID: authOut.User.ID, Provider: identity.ProviderGitHub}); err != nil {
		t.Fatalf("UnlinkIdentity() error = %v", err)
	}
	if got := repo.identityCount(authOut.User.ID); got != 0 {
		t.Fatalf("identity count = %d, want 0", got)
	}
}

func TestAuthServiceOAuthRejectsUnverifiedEmailLinkToExistingUser(t *testing.T) {
	svc, repo, _, providers, _ := newTestAuthService()
	registered, err := svc.Register(t.Context(), usercmd.RegisterCommand{
		Email:       "victim@example.com",
		Password:    "password-123",
		DisplayName: "Victim",
	})
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	providers.items[identity.ProviderGitHub] = &fakeOAuthProvider{
		authURL: "https://github.com/login/oauth/authorize",
		profile: &oauthstate.Profile{
			Provider:       identity.ProviderGitHub,
			ProviderUserID: "gh-attacker",
			Email:          "victim@example.com",
			EmailVerified:  false,
			DisplayName:    "Attacker",
		},
	}

	start, err := svc.StartOAuth(t.Context(), usercmd.StartOAuthCommand{Provider: identity.ProviderGitHub})
	if err != nil {
		t.Fatalf("StartOAuth() error = %v", err)
	}
	if _, err := svc.HandleOAuthCallback(t.Context(), usercmd.HandleOAuthCallbackCommand{
		Provider: identity.ProviderGitHub,
		State:    start.State,
		Code:     "provider-code",
	}); err == nil {
		t.Fatalf("HandleOAuthCallback() error = nil, want unverified email rejection")
	} else if !strings.Contains(err.Error(), "OAuth provider 邮箱未验证") {
		t.Fatalf("HandleOAuthCallback() error = %q, want unverified email rejection", err.Error())
	}
	if got := repo.identityCount(registered.User.ID); got != 0 {
		t.Fatalf("identity count = %d, want 0", got)
	}
}

func TestAuthServiceOAuthVerifiedEmailCanLinkExistingUser(t *testing.T) {
	svc, _, _, providers, _ := newTestAuthService()
	registered, err := svc.Register(t.Context(), usercmd.RegisterCommand{
		Email:       "owner@example.com",
		Password:    "password-123",
		DisplayName: "Owner",
	})
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	providers.items[identity.ProviderGitHub] = &fakeOAuthProvider{
		authURL: "https://github.com/login/oauth/authorize",
		profile: &oauthstate.Profile{
			Provider:       identity.ProviderGitHub,
			ProviderUserID: "gh-owner",
			Email:          "owner@example.com",
			EmailVerified:  true,
			DisplayName:    "Owner",
		},
	}

	start, err := svc.StartOAuth(t.Context(), usercmd.StartOAuthCommand{Provider: identity.ProviderGitHub})
	if err != nil {
		t.Fatalf("StartOAuth() error = %v", err)
	}
	callback, err := svc.HandleOAuthCallback(t.Context(), usercmd.HandleOAuthCallbackCommand{
		Provider: identity.ProviderGitHub,
		State:    start.State,
		Code:     "provider-code",
	})
	if err != nil {
		t.Fatalf("HandleOAuthCallback() error = %v", err)
	}
	authOut, err := svc.ExchangeOAuthLoginCode(t.Context(), usercmd.ExchangeOAuthLoginCodeCommand{Code: callback.LoginCode})
	if err != nil {
		t.Fatalf("ExchangeOAuthLoginCode() error = %v", err)
	}
	if authOut.User.ID != registered.User.ID {
		t.Fatalf("OAuth user ID = %d, want existing user ID %d", authOut.User.ID, registered.User.ID)
	}
}

func TestAuthServiceRefreshRejectsConcurrentReplayCASFailure(t *testing.T) {
	svc, repo, _, _, _ := newTestAuthService()
	loggedIn, err := svc.Register(t.Context(), usercmd.RegisterCommand{
		Email:       "rotate@example.com",
		Password:    "password-123",
		DisplayName: "Rotate",
	})
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	repo.rejectRefreshRevoke = true

	_, err = svc.Refresh(t.Context(), usercmd.RefreshTokenCommand{RefreshToken: loggedIn.Tokens.RefreshToken})
	if err == nil {
		t.Fatalf("Refresh() error = nil, want replay rejection")
	}
	if !strings.Contains(err.Error(), "refresh token 无效") {
		t.Fatalf("Refresh() error = %q, want invalid refresh token", err.Error())
	}
}

func TestAuthServiceStartOAuthRejectsUntrustedRedirect(t *testing.T) {
	svc, _, _, providers, _ := newTestAuthService()
	providers.items[identity.ProviderGitHub] = &fakeOAuthProvider{
		authURL: "https://github.com/login/oauth/authorize",
		profile: &oauthstate.Profile{
			Provider:       identity.ProviderGitHub,
			ProviderUserID: "gh-43",
			Email:          "mallory@example.com",
		},
	}

	_, err := svc.StartOAuth(t.Context(), usercmd.StartOAuthCommand{
		Provider:    identity.ProviderGitHub,
		RedirectURL: "https://evil.example/auth/callback",
	})
	if err == nil {
		t.Fatalf("StartOAuth() error = nil, want untrusted redirect rejection")
	}
}

func newTestAuthService() (*AuthService, *fakeStore, *fakeEmailSender, *fakeOAuthProviders, *fakeSecrets) {
	store := newFakeStore()
	email := &fakeEmailSender{}
	providers := &fakeOAuthProviders{items: map[string]*fakeOAuthProvider{}}
	secrets := &fakeSecrets{}
	svc := NewAuthService(
		Repositories{
			Users:         &fakeUserRepo{s: store},
			Identities:    &fakeIdentityRepo{s: store},
			RefreshTokens: &fakeRefreshTokenRepo{s: store},
			ActionTokens:  &fakeActionTokenRepo{s: store},
			OAuthStates:   &fakeOAuthStateRepo{s: store},
		},
		&fakeIDGenerator{next: 1000},
		fakeTxRunner{},
		fakePasswordHasher{},
		secrets,
		fakeAccessIssuer{},
		nil,
		providers,
		email,
		Config{
			RefreshTTL:         24 * time.Hour,
			EmailTokenTTL:      time.Hour,
			PasswordResetTTL:   time.Hour,
			OAuthStateTTL:      10 * time.Minute,
			OAuthLoginCodeTTL:  2 * time.Minute,
			SuccessRedirectURL: "https://app.example.com/auth/success",
		},
		slog.New(slog.DiscardHandler),
	)
	svc.now = func() time.Time { return testNow }
	return svc, store, email, providers, secrets
}

type fakeTxRunner struct{}

func (fakeTxRunner) RunInTx(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

type fakeIDGenerator struct {
	next int64
}

func (g *fakeIDGenerator) NextID(context.Context) (int64, error) {
	g.next++
	return g.next, nil
}

type fakePasswordHasher struct{}

func (fakePasswordHasher) Hash(password string) (string, string, error) {
	return "hash:" + password, "test", nil
}

func (fakePasswordHasher) Compare(hash string, password string) error {
	if hash != "hash:"+password {
		return stderrors.New("password mismatch")
	}
	return nil
}

type fakeSecrets struct {
	next int
}

func (s *fakeSecrets) NewToken() (string, error) {
	s.next++
	return fmt.Sprintf("token-%d", s.next), nil
}

func (fakeSecrets) HashToken(token string) string {
	return "hash:" + token
}

type fakeAccessIssuer struct{}

func (fakeAccessIssuer) IssueAccessToken(u *userdomain.User, now time.Time) (string, time.Time, error) {
	return fmt.Sprintf("access-%d", u.ID()), now.Add(15 * time.Minute), nil
}

type fakeEmailSender struct {
	verificationToken  string
	passwordResetToken string
}

func (s *fakeEmailSender) SendEmailVerification(_ context.Context, _ string, token string) error {
	s.verificationToken = token
	return nil
}

func (s *fakeEmailSender) SendPasswordReset(_ context.Context, _ string, token string) error {
	s.passwordResetToken = token
	return nil
}

type fakeOAuthProviders struct {
	items map[string]*fakeOAuthProvider
}

func (p *fakeOAuthProviders) Get(provider string) (interface {
	AuthCodeURL(state string) string
	Exchange(ctx context.Context, code string) (*oauthstate.Profile, error)
}, bool) {
	item, ok := p.items[provider]
	return item, ok
}

type fakeOAuthProvider struct {
	authURL string
	profile *oauthstate.Profile
}

func (p *fakeOAuthProvider) AuthCodeURL(state string) string {
	return p.authURL + "?state=" + state
}

func (p *fakeOAuthProvider) Exchange(context.Context, string) (*oauthstate.Profile, error) {
	return p.profile, nil
}

// fakeStore 保存所有聚合仓储的内存状态（测试辅助）。
type fakeStore struct {
	users               map[int64]*userdomain.User
	userByEmail         map[string]int64
	refreshByHash       map[string]*refreshtoken.RefreshToken
	actionByHash        map[string]*actiontoken.ActionToken
	oauthStateByHash    map[string]*oauthstate.OAuthState
	identityBySubject   map[string]*identity.Identity
	revokedReason       string
	replacedByTokenID   string
	rejectRefreshRevoke bool
}

func newFakeStore() *fakeStore {
	return &fakeStore{
		users:             map[int64]*userdomain.User{},
		userByEmail:       map[string]int64{},
		refreshByHash:     map[string]*refreshtoken.RefreshToken{},
		actionByHash:      map[string]*actiontoken.ActionToken{},
		oauthStateByHash:  map[string]*oauthstate.OAuthState{},
		identityBySubject: map[string]*identity.Identity{},
	}
}

func (s *fakeStore) identityCount(userID int64) int {
	count := 0
	for _, ident := range s.identityBySubject {
		if ident.UserID == userID {
			count++
		}
	}
	return count
}

func identityKey(provider string, providerUserID string) string {
	return provider + "|" + providerUserID
}

// fakeUserRepo 实现 userdomain.Repository。
type fakeUserRepo struct{ s *fakeStore }

var _ userdomain.Repository = (*fakeUserRepo)(nil)

func (r *fakeUserRepo) CreateUser(_ context.Context, u *userdomain.User) error {
	r.s.users[u.ID()] = u
	r.s.userByEmail[u.EmailNormalized()] = u.ID()
	return nil
}

func (r *fakeUserRepo) FindUserByID(_ context.Context, id int64) (*userdomain.User, error) {
	u, ok := r.s.users[id]
	if !ok {
		return nil, userdomain.ErrNotFound
	}
	return u, nil
}

func (r *fakeUserRepo) FindUserByEmail(_ context.Context, normalizedEmail string) (*userdomain.User, error) {
	id, ok := r.s.userByEmail[normalizedEmail]
	if !ok {
		return nil, userdomain.ErrNotFound
	}
	return r.s.users[id], nil
}

func (r *fakeUserRepo) SaveUser(_ context.Context, u *userdomain.User) error {
	r.s.users[u.ID()] = u
	r.s.userByEmail[u.EmailNormalized()] = u.ID()
	return nil
}

func (r *fakeUserRepo) UpdateUserLastLogin(_ context.Context, userID int64, now time.Time) error {
	u, ok := r.s.users[userID]
	if !ok {
		return userdomain.ErrNotFound
	}
	u.TouchLogin(now)
	return nil
}

func (r *fakeUserRepo) CountLoginMethods(_ context.Context, userID int64) (int, error) {
	u, ok := r.s.users[userID]
	if !ok {
		return 0, userdomain.ErrNotFound
	}
	count := r.s.identityCount(userID)
	if u.HasPassword() {
		count++
	}
	return count, nil
}

// fakeIdentityRepo 实现 identity.Repository。
type fakeIdentityRepo struct{ s *fakeStore }

var _ identity.Repository = (*fakeIdentityRepo)(nil)

func (r *fakeIdentityRepo) Find(_ context.Context, provider string, providerUserID string) (*identity.Identity, error) {
	ident, ok := r.s.identityBySubject[identityKey(provider, providerUserID)]
	if !ok {
		return nil, identity.ErrNotFound
	}
	return ident, nil
}

func (r *fakeIdentityRepo) Upsert(_ context.Context, ident *identity.Identity) error {
	r.s.identityBySubject[identityKey(ident.Provider, ident.ProviderUserID)] = ident
	return nil
}

func (r *fakeIdentityRepo) List(_ context.Context, userID int64) ([]*identity.Identity, error) {
	items := make([]*identity.Identity, 0)
	for _, ident := range r.s.identityBySubject {
		if ident.UserID == userID {
			items = append(items, ident)
		}
	}
	return items, nil
}

func (r *fakeIdentityRepo) Delete(_ context.Context, userID int64, provider string) error {
	for key, ident := range r.s.identityBySubject {
		if ident.UserID == userID && ident.Provider == provider {
			delete(r.s.identityBySubject, key)
			return nil
		}
	}
	return identity.ErrNotFound
}

// fakeRefreshTokenRepo 实现 refreshtoken.Repository。
type fakeRefreshTokenRepo struct{ s *fakeStore }

var _ refreshtoken.Repository = (*fakeRefreshTokenRepo)(nil)

func (r *fakeRefreshTokenRepo) Create(_ context.Context, token *refreshtoken.RefreshToken) error {
	r.s.refreshByHash[token.TokenHash] = token
	return nil
}

func (r *fakeRefreshTokenRepo) FindByHash(_ context.Context, tokenHash string, now time.Time) (*refreshtoken.RefreshToken, error) {
	token, ok := r.s.refreshByHash[tokenHash]
	if !ok || token.RevokedAt != nil || !token.ExpiresAt.After(now) {
		return nil, refreshtoken.ErrNotFound
	}
	return token, nil
}

func (r *fakeRefreshTokenRepo) Revoke(_ context.Context, tokenID string, replacedByTokenID string, reason string, now time.Time) error {
	for _, token := range r.s.refreshByHash {
		if token.TokenID == tokenID {
			if token.RevokedAt != nil || r.s.rejectRefreshRevoke {
				return refreshtoken.ErrNotFound
			}
			token.RevokedAt = &now
			token.ReplacedByTokenID = replacedByTokenID
			token.RevokeReason = reason
			r.s.revokedReason = reason
			r.s.replacedByTokenID = replacedByTokenID
			return nil
		}
	}
	return refreshtoken.ErrNotFound
}

// fakeActionTokenRepo 实现 actiontoken.Repository。
type fakeActionTokenRepo struct{ s *fakeStore }

var _ actiontoken.Repository = (*fakeActionTokenRepo)(nil)

func (r *fakeActionTokenRepo) Create(_ context.Context, token *actiontoken.ActionToken) error {
	r.s.actionByHash[token.TokenHash] = token
	return nil
}

func (r *fakeActionTokenRepo) Consume(_ context.Context, tokenHash string, purpose string, now time.Time) (*actiontoken.ActionToken, error) {
	token, ok := r.s.actionByHash[tokenHash]
	if !ok || token.Purpose != purpose || token.ConsumedAt != nil || !token.ExpiresAt.After(now) {
		return nil, actiontoken.ErrNotFound
	}
	token.ConsumedAt = &now
	return token, nil
}

// fakeOAuthStateRepo 实现 oauthstate.Repository。
type fakeOAuthStateRepo struct{ s *fakeStore }

var _ oauthstate.Repository = (*fakeOAuthStateRepo)(nil)

func (r *fakeOAuthStateRepo) Create(_ context.Context, state *oauthstate.OAuthState) error {
	r.s.oauthStateByHash[state.Provider+"|"+state.StateHash] = state
	return nil
}

func (r *fakeOAuthStateRepo) Consume(_ context.Context, provider string, stateHash string, now time.Time) (*oauthstate.OAuthState, error) {
	state, ok := r.s.oauthStateByHash[provider+"|"+stateHash]
	if !ok || state.ConsumedAt != nil || !state.ExpiresAt.After(now) {
		return nil, oauthstate.ErrNotFound
	}
	state.ConsumedAt = &now
	return state, nil
}
