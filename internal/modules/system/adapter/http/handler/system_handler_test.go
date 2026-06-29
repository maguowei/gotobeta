package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/maguowei/gotobeta/internal/pkg/health"
)

func TestHealthzReturnsOK(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	h := NewSystemHandler(health.NewRegistry())
	router := gin.New()
	router.GET("/healthz", h.Healthz)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/healthz", nil)
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status code = %d, want %d", recorder.Code, http.StatusOK)
	}

	var body map[string]string
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if body["status"] != "ok" {
		t.Fatalf("status = %q, want ok", body["status"])
	}
}

func TestReadyzAllHealthy(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	reg := health.NewRegistry()
	reg.Register("db", health.CheckerFunc(func(_ context.Context) error { return nil }))

	h := NewSystemHandler(reg)
	router := gin.New()
	router.GET("/readyz", h.Readyz)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/readyz", nil)
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status code = %d, want %d", recorder.Code, http.StatusOK)
	}
}

func TestReadyzDegraded(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	reg := health.NewRegistry()
	reg.Register("db", health.CheckerFunc(func(_ context.Context) error {
		return errors.New("connection refused")
	}))

	h := NewSystemHandler(reg)
	router := gin.New()
	router.GET("/readyz", h.Readyz)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/readyz", nil)
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("status code = %d, want %d", recorder.Code, http.StatusServiceUnavailable)
	}

	var body health.Result
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if body.Status != "degraded" {
		t.Fatalf("status = %q, want degraded", body.Status)
	}
	if body.Checks["db"] != "connection refused" {
		t.Fatalf("db check = %q, want connection refused", body.Checks["db"])
	}
}
