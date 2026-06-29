package handler_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	mediahandler "github.com/maguowei/gotobeta/internal/modules/media/adapter/http/handler"
	mediarouter "github.com/maguowei/gotobeta/internal/modules/media/adapter/http/router"
	mediacmd "github.com/maguowei/gotobeta/internal/modules/media/application/command"
	mediaresult "github.com/maguowei/gotobeta/internal/modules/media/application/result"
	"github.com/maguowei/gotobeta/internal/pkg/auth"
)

type fakeUC struct{}

func (fakeUC) Presign(_ context.Context, _ mediacmd.PresignAttachmentCommand) (*mediaresult.PresignResult, error) {
	return &mediaresult.PresignResult{AttachmentID: 5001, ObjectKey: "k", UploadURL: "https://x"}, nil
}
func (fakeUC) Commit(_ context.Context, _ mediacmd.CommitAttachmentCommand) (*mediaresult.AttachmentResult, error) {
	return &mediaresult.AttachmentResult{ID: 5001, Status: 2, PublicURL: "https://cdn/x"}, nil
}

func newRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	h := mediahandler.NewAttachmentHandler(fakeUC{})
	e := gin.New()
	authMW := func(c *gin.Context) {
		c.Request = c.Request.WithContext(auth.WithClaims(c.Request.Context(), &auth.Claims{UserID: 9}))
		c.Next()
	}
	mediarouter.RegisterRoutes(e.Group("/api/v1"), h, authMW)
	return e
}

func do(t *testing.T, e *gin.Engine, method, path, body string) int {
	t.Helper()
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.ServeHTTP(w, r)
	return w.Code
}

func TestMediaRoutes(t *testing.T) {
	e := newRouter()
	if code := do(t, e, "POST", "/api/v1/attachments/presign", `{"workspaceId":"1","fileName":"a.png","contentType":"image/png","sizeBytes":10}`); code != http.StatusOK {
		t.Errorf("presign 应 200, got %d", code)
	}
	if code := do(t, e, "POST", "/api/v1/attachments/5001/commit", `{"workspaceId":"1"}`); code != http.StatusOK {
		t.Errorf("commit 应 200, got %d", code)
	}
	if code := do(t, e, "POST", "/api/v1/attachments/abc/commit", `{"workspaceId":"1"}`); code != http.StatusBadRequest {
		t.Errorf("非法 ID 应 400, got %d", code)
	}
}
