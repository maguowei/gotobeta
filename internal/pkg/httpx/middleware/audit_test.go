package middleware

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/maguowei/gotobeta/internal/pkg/requestctx"
)

func TestAuditLogsCallerBizCodeAndMaskedRequestBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var buf bytes.Buffer
	auditLogger := slog.New(slog.NewJSONHandler(&buf, nil))
	router := gin.New()
	router.Use(func(c *gin.Context) {
		ctx := requestctx.WithCaller(c.Request.Context(), "alice", "user")
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	})
	router.Use(Audit(auditLogger, AuditOptions{
		Enabled:             true,
		LogRequestBody:      true,
		MaskSensitiveFields: true,
		MaxBodyBytes:        64 * 1024,
	}))
	router.POST("/items", func(c *gin.Context) {
		c.PureJSON(http.StatusCreated, gin.H{"code": 0, "message": "success"})
	})

	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/items", strings.NewReader(`{"name":"demo","password":"secret"}`))
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	var entry map[string]any
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("unmarshal audit log: %v", err)
	}

	if entry["caller"] != "alice" || entry["callerType"] != "user" {
		t.Fatalf("caller fields = (%v, %v), want alice/user", entry["caller"], entry["callerType"])
	}
	if entry["bizCode"] != float64(0) {
		t.Fatalf("bizCode = %v, want 0", entry["bizCode"])
	}

	body, _ := entry["requestBody"].(string)
	if !strings.Contains(body, `"password":"***"`) {
		t.Fatalf("requestBody not masked: %q", body)
	}
}

func TestAuditMasksResponseBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var buf bytes.Buffer
	auditLogger := slog.New(slog.NewJSONHandler(&buf, nil))
	router := gin.New()
	router.Use(Audit(auditLogger, AuditOptions{
		Enabled:             true,
		LogResponseBody:     true,
		MaskSensitiveFields: true,
		MaxBodyBytes:        64 * 1024,
	}))
	router.GET("/token", func(c *gin.Context) {
		c.PureJSON(http.StatusOK, gin.H{"code": 0, "access_token": "secret-token"})
	})

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/token", nil))

	entry := decodeAuditEntry(t, buf.Bytes())
	body, _ := entry["responseBody"].(string)
	if strings.Contains(body, "secret-token") || !strings.Contains(body, `"access_token":"***"`) {
		t.Fatalf("responseBody not masked: %q", body)
	}
}

func TestAuditDoesNotLogRawMalformedJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var buf bytes.Buffer
	auditLogger := slog.New(slog.NewJSONHandler(&buf, nil))
	router := gin.New()
	router.Use(Audit(auditLogger, AuditOptions{
		Enabled:             true,
		LogRequestBody:      true,
		MaskSensitiveFields: true,
		MaxBodyBytes:        64 * 1024,
	}))
	router.POST("/items", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/items", strings.NewReader(`{"password":"secret"`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(httptest.NewRecorder(), req)

	entry := decodeAuditEntry(t, buf.Bytes())
	body, _ := entry["requestBody"].(string)
	if strings.Contains(body, "secret") || !strings.Contains(body, "invalid json") {
		t.Fatalf("malformed requestBody not safely summarized: %q", body)
	}
}

func TestAuditRecordsRequestBodyReadErrorSafely(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var buf bytes.Buffer
	auditLogger := slog.New(slog.NewJSONHandler(&buf, nil))
	router := gin.New()
	router.Use(Audit(auditLogger, AuditOptions{
		Enabled:        true,
		LogRequestBody: true,
		MaxBodyBytes:   64 * 1024,
	}))
	router.POST("/items", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/items", nil)
	req.Body = failingReadCloser{}
	router.ServeHTTP(httptest.NewRecorder(), req)

	entry := decodeAuditEntry(t, buf.Bytes())
	body, _ := entry["requestBody"].(string)
	if !strings.Contains(body, "body read error") || strings.Contains(body, "read failed") {
		t.Fatalf("requestBody read error is not safe: %q", body)
	}
}

func TestAuditDisabledSkipsLogging(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var buf bytes.Buffer
	router := gin.New()
	router.Use(Audit(slog.New(slog.NewJSONHandler(&buf, nil)), AuditOptions{Enabled: false}))
	router.GET("/ping", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/ping", nil))

	if buf.Len() != 0 {
		t.Fatalf("audit log should be empty when disabled: %s", buf.String())
	}
}

func TestAuditSummarizesLargeRequestBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var buf bytes.Buffer
	router := gin.New()
	router.Use(Audit(slog.New(slog.NewJSONHandler(&buf, nil)), AuditOptions{
		Enabled:             true,
		LogRequestBody:      true,
		MaskSensitiveFields: false,
		MaxBodyBytes:        8,
	}))
	router.POST("/items", func(c *gin.Context) {
		c.Status(http.StatusCreated)
	})

	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/items", strings.NewReader(`{"password":"secret"}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(httptest.NewRecorder(), req)

	entry := decodeAuditEntry(t, buf.Bytes())
	body, _ := entry["requestBody"].(string)
	if body != "[body too large, size=9]" {
		t.Fatalf("requestBody = %q, want safe large body summary", body)
	}
}

func TestAuditKeepsBodyWhenMaskingDisabledAndSummarizesLargeResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var buf bytes.Buffer
	router := gin.New()
	router.Use(Audit(slog.New(slog.NewJSONHandler(&buf, nil)), AuditOptions{
		Enabled:             true,
		LogRequestBody:      true,
		LogResponseBody:     true,
		MaskSensitiveFields: false,
		MaxBodyBytes:        64 * 1024,
	}))
	router.POST("/items", func(c *gin.Context) {
		c.String(http.StatusCreated, "ok")
	})

	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/items", strings.NewReader(`{"password":"secret"}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(httptest.NewRecorder(), req)

	entry := decodeAuditEntry(t, buf.Bytes())
	requestBody, _ := entry["requestBody"].(string)
	if requestBody != `{"password":"secret"}` {
		t.Fatalf("requestBody = %q, want raw body when masking disabled", requestBody)
	}
	responseBody, _ := entry["responseBody"].(string)
	if responseBody != "ok" {
		t.Fatalf("responseBody = %q, want raw response body", responseBody)
	}

	buf.Reset()
	router = gin.New()
	router.Use(Audit(slog.New(slog.NewJSONHandler(&buf, nil)), AuditOptions{
		Enabled:         true,
		LogResponseBody: true,
		MaxBodyBytes:    8,
	}))
	router.GET("/large", func(c *gin.Context) {
		c.String(http.StatusOK, "response body longer than limit")
	})
	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/large", nil))

	entry = decodeAuditEntry(t, buf.Bytes())
	responseBody, _ = entry["responseBody"].(string)
	if !strings.Contains(responseBody, "body too large") {
		t.Fatalf("responseBody = %q, want too large summary", responseBody)
	}
}

func TestAuditExtractsStatusStringBizCodeAndIgnoresInvalidStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var buf bytes.Buffer
	router := gin.New()
	router.Use(Audit(slog.New(slog.NewJSONHandler(&buf, nil)), AuditOptions{
		Enabled:         true,
		LogResponseBody: true,
		MaxBodyBytes:    64 * 1024,
	}))
	router.GET("/items", func(c *gin.Context) {
		c.PureJSON(http.StatusAccepted, gin.H{"Status": "7"})
	})

	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/items", nil))

	entry := decodeAuditEntry(t, buf.Bytes())
	if entry["bizCode"] != float64(7) {
		t.Fatalf("bizCode = %#v, want 7", entry["bizCode"])
	}

	buf.Reset()
	router = gin.New()
	router.Use(Audit(slog.New(slog.NewJSONHandler(&buf, nil)), AuditOptions{
		Enabled:         true,
		LogResponseBody: true,
		MaxBodyBytes:    64 * 1024,
	}))
	router.GET("/bad-status", func(c *gin.Context) {
		c.PureJSON(http.StatusOK, gin.H{"Status": "not-a-number"})
	})
	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/bad-status", nil))

	entry = decodeAuditEntry(t, buf.Bytes())
	if entry["bizCode"] != float64(-1) {
		t.Fatalf("invalid status bizCode = %#v, want -1", entry["bizCode"])
	}
}

func decodeAuditEntry(t *testing.T, data []byte) map[string]any {
	t.Helper()

	var entry map[string]any
	if err := json.Unmarshal(data, &entry); err != nil {
		t.Fatalf("unmarshal audit log: %v", err)
	}
	return entry
}

type failingReadCloser struct{}

func (failingReadCloser) Read([]byte) (int, error) {
	return 0, errors.New("read failed")
}

func (failingReadCloser) Close() error { return nil }

var _ io.ReadCloser = failingReadCloser{}
