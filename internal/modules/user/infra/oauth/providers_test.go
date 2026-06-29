package oauth

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"golang.org/x/oauth2"

	"github.com/maguowei/gotobeta/internal/infra/config"
	"github.com/maguowei/gotobeta/internal/modules/user/domain/identity"
)

func TestRegistryReturnsEnabledProviders(t *testing.T) {
	registry := NewRegistry(config.AuthOAuthConfig{
		GitHub: config.AuthOAuthProviderConfig{
			Enabled:     true,
			ClientID:    "github-client",
			RedirectURL: "https://app.example.com/github/callback",
		},
		Google: config.AuthOAuthProviderConfig{
			Enabled:     true,
			ClientID:    "google-client",
			RedirectURL: "https://app.example.com/google/callback",
		},
	})

	if _, ok := registry.Get(identity.ProviderGitHub); !ok {
		t.Fatalf("GitHub provider was not registered")
	}
	if _, ok := registry.Get(identity.ProviderGoogle); !ok {
		t.Fatalf("Google provider was not registered")
	}
	if _, ok := registry.Get("unknown"); ok {
		t.Fatalf("unknown provider was registered")
	}
}

func TestProviderFetchesGitHubProfile(t *testing.T) {
	ctx := context.WithValue(t.Context(), oauth2.HTTPClient, testOAuthHTTPClient(t, map[string]string{
		"oauth.example/token": `{"access_token":"access","token_type":"Bearer"}`,
		"api.example/user":    `{"id":42,"login":"alice","name":"","email":"","avatar_url":"https://example.com/a.png","html_url":"https://github.com/alice"}`,
		"api.example/emails":  `[{"email":"alice@example.com","primary":true,"verified":true}]`,
	}))

	provider := NewProvider(identity.ProviderGitHub, oauth2.Config{
		ClientID:     "client",
		ClientSecret: "secret",
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://oauth.example/auth",
			TokenURL: "https://oauth.example/token",
		},
	}, "https://api.example/user", "https://api.example/emails")

	if url := provider.AuthCodeURL("state-1"); !strings.Contains(url, "state=state-1") {
		t.Fatalf("AuthCodeURL() = %q", url)
	}
	profile, err := provider.Exchange(ctx, "provider-code")
	if err != nil {
		t.Fatalf("Exchange() error = %v", err)
	}
	if profile.ProviderUserID != "42" || profile.Email != "alice@example.com" || !profile.EmailVerified || profile.DisplayName != "alice" {
		t.Fatalf("GitHub profile = %+v", profile)
	}
}

func TestProviderFetchesGoogleProfile(t *testing.T) {
	ctx := context.WithValue(t.Context(), oauth2.HTTPClient, testOAuthHTTPClient(t, map[string]string{
		"oauth.example/token":  `{"access_token":"access","token_type":"Bearer"}`,
		"api.example/userinfo": `{"sub":"google-42","email":"alice@example.com","email_verified":true,"name":"Alice","picture":"https://example.com/a.png","profile":"https://profiles.example.com/alice"}`,
	}))

	provider := NewProvider(identity.ProviderGoogle, oauth2.Config{
		ClientID:     "client",
		ClientSecret: "secret",
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://oauth.example/auth",
			TokenURL: "https://oauth.example/token",
		},
	}, "https://api.example/userinfo", "")

	profile, err := provider.Exchange(ctx, "provider-code")
	if err != nil {
		t.Fatalf("Exchange() error = %v", err)
	}
	if profile.ProviderUserID != "google-42" || profile.Email != "alice@example.com" || profile.DisplayName != "Alice" {
		t.Fatalf("Google profile = %+v", profile)
	}
}

func TestProviderRejectsUnsupportedProvider(t *testing.T) {
	ctx := context.WithValue(t.Context(), oauth2.HTTPClient, testOAuthHTTPClient(t, map[string]string{
		"oauth.example/token": `{"access_token":"access","token_type":"Bearer"}`,
	}))

	provider := NewProvider("unknown", oauth2.Config{
		Endpoint: oauth2.Endpoint{TokenURL: "https://oauth.example/token"},
	}, "https://api.example/user", "")

	if _, err := provider.Exchange(ctx, "provider-code"); err == nil {
		t.Fatalf("Exchange() error = nil, want unsupported provider")
	}
}

func testOAuthHTTPClient(t *testing.T, routes map[string]string) *http.Client {
	t.Helper()
	return &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		body, ok := routes[req.URL.Host+req.URL.Path]
		if !ok {
			t.Fatalf("unexpected request %s", req.URL.String())
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(body)),
			Request:    req,
		}, nil
	})}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}
