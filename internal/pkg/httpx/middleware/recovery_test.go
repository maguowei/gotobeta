package middleware

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	sentrysdk "github.com/getsentry/sentry-go"
	"github.com/gin-gonic/gin"
)

func TestSentryUsesRouteTemplateForRequestPath(t *testing.T) {
	gin.SetMode(gin.TestMode)

	transport := &sentrysdk.MockTransport{}
	client, err := sentrysdk.NewClient(sentrysdk.ClientOptions{
		Dsn:       "https://public@example.com/1",
		Transport: transport,
	})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	hub := sentrysdk.CurrentHub()
	previousClient := hub.Client()
	hub.BindClient(client)
	t.Cleanup(func() {
		hub.BindClient(previousClient)
	})

	router := gin.New()
	router.Use(Sentry())
	router.Use(Recovery(slog.New(slog.DiscardHandler)))
	router.GET("/todos/:id", func(*gin.Context) {
		panic("boom")
	})

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/todos/123?trace=abc", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	events := transport.Events()
	if len(events) != 1 {
		t.Fatalf("captured events = %d, want 1", len(events))
	}
	if got, want := events[0].Tags["request_path"], "/todos/:id"; got != want {
		t.Fatalf("request_path tag = %q, want %q", got, want)
	}
}
