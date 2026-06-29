package handler_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	realtimehandler "github.com/maguowei/gotobeta/internal/modules/realtime/adapter/http/handler"
	"github.com/maguowei/gotobeta/internal/pkg/auth"
)

type fakeTicketUC struct {
	token string
	err   error
}

func (f fakeTicketUC) IssueTicket(_ context.Context, _ int64) (string, error) {
	return f.token, f.err
}

func newEngine(uc realtimehandler.TicketUseCase, withClaims bool) *gin.Engine {
	gin.SetMode(gin.TestMode)
	h := realtimehandler.NewTicketHandler(uc)
	e := gin.New()
	e.POST("/ws/ticket", func(c *gin.Context) {
		if withClaims {
			c.Request = c.Request.WithContext(auth.WithClaims(c.Request.Context(), &auth.Claims{UserID: 9}))
		}
		h.IssueTicket(c)
	})
	return e
}

func TestIssueTicketOK(t *testing.T) {
	e := newEngine(fakeTicketUC{token: "tk-1"}, true)
	r := httptest.NewRequest("POST", "/ws/ticket", nil)
	w := httptest.NewRecorder()
	e.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("应 200, got %d", w.Code)
	}
}

func TestIssueTicketNoClaims(t *testing.T) {
	e := newEngine(fakeTicketUC{token: "tk-1"}, false)
	r := httptest.NewRequest("POST", "/ws/ticket", nil)
	w := httptest.NewRecorder()
	e.ServeHTTP(w, r)
	if w.Code == http.StatusOK {
		t.Fatalf("缺少 claims 不应 200, got %d", w.Code)
	}
}
