package middleware

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/maguowei/gotobeta/internal/pkg/requestctx"
)

func TestLoggerWritesRequestFields(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var buf bytes.Buffer
	log := slog.New(slog.NewJSONHandler(&buf, nil))

	router := gin.New()
	router.Use(Logger(log))
	router.GET("/ping/:id", func(c *gin.Context) {
		c.Status(http.StatusAccepted)
	})

	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/ping/42", nil)
	request = request.WithContext(requestctx.WithRequestID(request.Context(), "req-123"))
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusAccepted {
		t.Fatalf("status code = %d, want %d", recorder.Code, http.StatusAccepted)
	}

	var entry map[string]any
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("unmarshal log entry: %v", err)
	}

	if entry["requestId"] != "req-123" {
		t.Fatalf("requestId = %v, want req-123", entry["requestId"])
	}
	if entry["method"] != http.MethodGet {
		t.Fatalf("method = %v, want GET", entry["method"])
	}
	if entry["path"] != "/ping/:id" {
		t.Fatalf("path = %v, want route template", entry["path"])
	}
	if entry["status"] != float64(http.StatusAccepted) {
		t.Fatalf("status = %v, want %d", entry["status"], http.StatusAccepted)
	}
}
