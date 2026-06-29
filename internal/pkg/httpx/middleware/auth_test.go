package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"github.com/maguowei/gotobeta/internal/pkg/auth"
)

func TestAuthJWTInjectsClaims(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := testAuthJWTConfig()
	token := signedAuthToken(t, cfg, jwt.RegisteredClaims{
		Issuer:    cfg.Issuer,
		Audience:  jwt.ClaimStrings{cfg.Audience},
		Subject:   "user-1",
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
	})
	router := gin.New()
	router.Use(AuthJWT(cfg))
	router.GET("/private", func(c *gin.Context) {
		claims, ok := auth.ClaimsFromContext(c.Request.Context())
		if !ok {
			c.Status(http.StatusInternalServerError)
			return
		}
		c.String(http.StatusOK, claims.Subject)
	})

	req := httptest.NewRequest(http.MethodGet, "/private", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK || rec.Body.String() != "user-1" {
		t.Fatalf("response = %d %q", rec.Code, rec.Body.String())
	}
}

func TestAuthJWTRejectsMissingBearerToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(AuthJWT(testAuthJWTConfig()))
	router.GET("/private", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/private", nil))

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

// stubRevoker 按预置集合判定 jti 是否吊销。
type stubRevoker struct{ revoked map[string]bool }

func (r stubRevoker) IsRevoked(_ context.Context, jti string) (bool, error) {
	return r.revoked[jti], nil
}

func TestAuthJWTRejectsRevokedToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := testAuthJWTConfig()
	cfg.Revoker = stubRevoker{revoked: map[string]bool{"jti-1": true}}
	token := signedAuthToken(t, cfg, jwt.RegisteredClaims{
		ID:        "jti-1",
		Issuer:    cfg.Issuer,
		Audience:  jwt.ClaimStrings{cfg.Audience},
		Subject:   "user-1",
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
	})
	router := gin.New()
	router.Use(AuthJWT(cfg))
	router.GET("/private", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	req := httptest.NewRequest(http.MethodGet, "/private", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("吊销 token 应被拒，status = %d, want 401", rec.Code)
	}
}

func TestAuthJWTAllowsNonRevokedToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := testAuthJWTConfig()
	cfg.Revoker = stubRevoker{revoked: map[string]bool{"other": true}}
	token := signedAuthToken(t, cfg, jwt.RegisteredClaims{
		ID:        "jti-live",
		Issuer:    cfg.Issuer,
		Audience:  jwt.ClaimStrings{cfg.Audience},
		Subject:   "user-1",
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
	})
	router := gin.New()
	router.Use(AuthJWT(cfg))
	router.GET("/private", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	req := httptest.NewRequest(http.MethodGet, "/private", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("未吊销 token 应放行，status = %d, want 204", rec.Code)
	}
}

func testAuthJWTConfig() AuthJWTOptions {
	return AuthJWTOptions{
		Enabled:    true,
		Issuer:     "gotobeta",
		Audience:   "gotobeta-api",
		HMACSecret: "test-secret",
		ClockSkew:  "30s",
	}
}

func signedAuthToken(t *testing.T, opts AuthJWTOptions, claims jwt.RegisteredClaims) string {
	t.Helper()
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(opts.HMACSecret))
	if err != nil {
		t.Fatalf("SignedString() error = %v", err)
	}
	return token
}
