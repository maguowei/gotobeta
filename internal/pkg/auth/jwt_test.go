package auth

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestParseBearer(t *testing.T) {
	token, ok := ParseBearer("Bearer abc.def")
	if !ok || token != "abc.def" {
		t.Fatalf("ParseBearer() = %q, %v", token, ok)
	}
	if _, ok := ParseBearer("Basic abc"); ok {
		t.Fatalf("ParseBearer should reject non-bearer header")
	}
}

func TestParseTokenValidatesHMACClaims(t *testing.T) {
	cfg := testJWTConfig()
	token := signedToken(t, cfg, jwt.RegisteredClaims{
		Issuer:    cfg.Issuer,
		Audience:  jwt.ClaimStrings{cfg.Audience},
		Subject:   "user-1",
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
	})

	claims, err := ParseToken(token, cfg)
	if err != nil {
		t.Fatalf("ParseToken() error = %v", err)
	}
	if claims.Subject != "user-1" {
		t.Fatalf("subject = %q, want user-1", claims.Subject)
	}
}

func TestParseTokenRejectsWrongAudience(t *testing.T) {
	cfg := testJWTConfig()
	token := signedToken(t, cfg, jwt.RegisteredClaims{
		Issuer:    cfg.Issuer,
		Audience:  jwt.ClaimStrings{"other"},
		Subject:   "user-1",
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
	})

	if _, err := ParseToken(token, cfg); err == nil {
		t.Fatalf("ParseToken() error = nil, want audience validation error")
	}
}

func TestIssueAccessTokenIncludesUserClaims(t *testing.T) {
	cfg := testJWTConfig()
	now := time.Now()

	token, expiresAt, err := IssueAccessToken(42, "alice@example.com", TokenConfig{
		Issuer:     cfg.Issuer,
		Audience:   cfg.Audience,
		HMACSecret: cfg.HMACSecret,
		TTL:        15 * time.Minute,
		ClockSkew:  cfg.ClockSkew,
	}, now)
	if err != nil {
		t.Fatalf("IssueAccessToken() error = %v", err)
	}
	if !expiresAt.Equal(now.Add(15 * time.Minute)) {
		t.Fatalf("expiresAt = %s, want %s", expiresAt, now.Add(15*time.Minute))
	}

	claims, err := ParseToken(token, cfg)
	if err != nil {
		t.Fatalf("ParseToken() error = %v", err)
	}
	if claims.UserID != 42 || claims.Email != "alice@example.com" || claims.Subject != "42" {
		t.Fatalf("claims = %+v, want user claims", claims)
	}
}

func testJWTConfig() TokenConfig {
	return TokenConfig{
		Issuer:     "gotobeta",
		Audience:   "gotobeta-api",
		HMACSecret: "test-secret",
		ClockSkew:  "30s",
	}
}

func signedToken(t *testing.T, cfg TokenConfig, claims jwt.RegisteredClaims) string {
	t.Helper()
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(cfg.HMACSecret))
	if err != nil {
		t.Fatalf("SignedString() error = %v", err)
	}
	return token
}
