package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
	"golang.org/x/oauth2/google"

	"github.com/maguowei/gotobeta/internal/infra/config"
	"github.com/maguowei/gotobeta/internal/modules/user/domain/identity"
	"github.com/maguowei/gotobeta/internal/modules/user/domain/oauthstate"
)

// Registry 保存可用 OAuth provider。
type Registry struct {
	providers map[string]*Provider
}

// NewRegistry 根据配置创建 provider registry。
func NewRegistry(cfg config.AuthOAuthConfig) *Registry {
	providers := make(map[string]*Provider)
	if cfg.GitHub.Enabled {
		providers[identity.ProviderGitHub] = NewProvider(identity.ProviderGitHub, oauth2.Config{
			ClientID:     cfg.GitHub.ClientID,
			ClientSecret: cfg.GitHub.ClientSecret,
			RedirectURL:  cfg.GitHub.RedirectURL,
			Scopes:       []string{"read:user", "user:email"},
			Endpoint:     github.Endpoint,
		}, "https://api.github.com/user", "https://api.github.com/user/emails")
	}
	if cfg.Google.Enabled {
		providers[identity.ProviderGoogle] = NewProvider(identity.ProviderGoogle, oauth2.Config{
			ClientID:     cfg.Google.ClientID,
			ClientSecret: cfg.Google.ClientSecret,
			RedirectURL:  cfg.Google.RedirectURL,
			Scopes:       []string{"openid", "email", "profile"},
			Endpoint:     google.Endpoint,
		}, "https://openidconnect.googleapis.com/v1/userinfo", "")
	}
	return &Registry{providers: providers}
}

// Get 返回 provider。
func (r *Registry) Get(name string) (interface {
	AuthCodeURL(state string) string
	Exchange(ctx context.Context, code string) (*oauthstate.Profile, error)
}, bool) {
	provider, ok := r.providers[name]
	return provider, ok
}

// Provider 是 OAuth provider HTTP 实现。
type Provider struct {
	name      string
	cfg       oauth2.Config
	userURL   string
	emailsURL string
}

// NewProvider 创建 provider。
func NewProvider(name string, cfg oauth2.Config, userURL string, emailsURL string) *Provider {
	return &Provider{name: name, cfg: cfg, userURL: userURL, emailsURL: emailsURL}
}

// AuthCodeURL 生成授权地址。
func (p *Provider) AuthCodeURL(state string) string {
	return p.cfg.AuthCodeURL(state, oauth2.AccessTypeOnline)
}

// Exchange 使用授权码换取用户资料。
func (p *Provider) Exchange(ctx context.Context, code string) (*oauthstate.Profile, error) {
	token, err := p.cfg.Exchange(ctx, code)
	if err != nil {
		return nil, err
	}
	client := p.cfg.Client(ctx, token)
	switch p.name {
	case identity.ProviderGitHub:
		return p.fetchGitHubProfile(ctx, client)
	case identity.ProviderGoogle:
		return p.fetchGoogleProfile(ctx, client)
	default:
		return nil, fmt.Errorf("unsupported provider %q", p.name)
	}
}

func (p *Provider) fetchGitHubProfile(ctx context.Context, client *http.Client) (*oauthstate.Profile, error) {
	var user struct {
		ID        int64  `json:"id"`
		Login     string `json:"login"`
		Name      string `json:"name"`
		Email     string `json:"email"`
		AvatarURL string `json:"avatar_url"`
		HTMLURL   string `json:"html_url"`
	}
	if err := getJSON(ctx, client, p.userURL, &user); err != nil {
		return nil, err
	}
	email := user.Email
	emailVerified := false
	if strings.TrimSpace(email) == "" && p.emailsURL != "" {
		var emails []struct {
			Email    string `json:"email"`
			Primary  bool   `json:"primary"`
			Verified bool   `json:"verified"`
		}
		if err := getJSON(ctx, client, p.emailsURL, &emails); err != nil {
			return nil, err
		}
		for _, item := range emails {
			if item.Primary {
				email = item.Email
				emailVerified = item.Verified
				break
			}
		}
	}
	displayName := user.Name
	if strings.TrimSpace(displayName) == "" {
		displayName = user.Login
	}
	return &oauthstate.Profile{
		Provider:       identity.ProviderGitHub,
		ProviderUserID: fmt.Sprintf("%d", user.ID),
		Email:          email,
		EmailVerified:  emailVerified,
		DisplayName:    displayName,
		AvatarURL:      user.AvatarURL,
		ProfileURL:     user.HTMLURL,
	}, nil
}

func (p *Provider) fetchGoogleProfile(ctx context.Context, client *http.Client) (*oauthstate.Profile, error) {
	var user struct {
		Sub           string `json:"sub"`
		Email         string `json:"email"`
		EmailVerified bool   `json:"email_verified"`
		Name          string `json:"name"`
		Picture       string `json:"picture"`
		Profile       string `json:"profile"`
	}
	if err := getJSON(ctx, client, p.userURL, &user); err != nil {
		return nil, err
	}
	return &oauthstate.Profile{
		Provider:       identity.ProviderGoogle,
		ProviderUserID: user.Sub,
		Email:          user.Email,
		EmailVerified:  user.EmailVerified,
		DisplayName:    user.Name,
		AvatarURL:      user.Picture,
		ProfileURL:     user.Profile,
	}, nil
}

func getJSON(ctx context.Context, client *http.Client, url string, target any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("oauth provider returned status %d", resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(target)
}
