package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/maguowei/gotobeta/internal/pkg/requestctx"
)

func TestRequestIDCreatesResponseHeaderAndContextValue(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(RequestID())

	var gotRequestID string
	router.GET("/ping", func(c *gin.Context) {
		gotRequestID = requestctx.GetRequestID(c.Request.Context())
		c.Status(http.StatusNoContent)
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/ping", nil)
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status code = %d, want %d", recorder.Code, http.StatusNoContent)
	}

	responseRequestID := recorder.Header().Get(headerXRequestID)
	if responseRequestID == "" {
		t.Fatalf("response header %s is empty", headerXRequestID)
	}

	if gotRequestID != responseRequestID {
		t.Fatalf("context request id = %q, want %q", gotRequestID, responseRequestID)
	}
}

func TestRequestIDPreservesIncomingHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(RequestID())

	var gotRequestID string
	router.GET("/ping", func(c *gin.Context) {
		gotRequestID = requestctx.GetRequestID(c.Request.Context())
		c.Status(http.StatusNoContent)
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/ping", nil)
	request.Header.Set(headerXRequestID, "req-from-client")
	router.ServeHTTP(recorder, request)

	if gotRequestID != "req-from-client" {
		t.Fatalf("context request id = %q, want req-from-client", gotRequestID)
	}

	if recorder.Header().Get(headerXRequestID) != "req-from-client" {
		t.Fatalf("response request id = %q, want req-from-client", recorder.Header().Get(headerXRequestID))
	}
}

func TestSanitizeRequestIDRejectsUnsafeValues(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{name: "empty", raw: "", want: ""},
		{name: "newline", raw: "req\nnext", want: ""},
		{name: "delete", raw: "req" + string(rune(0x7f)), want: ""},
		{name: "too long", raw: string(make([]byte, maxRequestIDLength+1)), want: ""},
		{name: "valid", raw: "req-123", want: "req-123"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := sanitizeRequestID(tt.raw); got != tt.want {
				t.Fatalf("sanitizeRequestID() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRequestIDReplacesUnsafeIncomingHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(RequestID())

	var gotRequestID string
	router.GET("/ping", func(c *gin.Context) {
		gotRequestID = requestctx.GetRequestID(c.Request.Context())
		c.Status(http.StatusNoContent)
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/ping", nil)
	request.Header.Set(headerXRequestID, "req\nbad")
	router.ServeHTTP(recorder, request)

	if gotRequestID == "" || gotRequestID == "req\nbad" {
		t.Fatalf("context request id = %q, want generated safe id", gotRequestID)
	}
}
