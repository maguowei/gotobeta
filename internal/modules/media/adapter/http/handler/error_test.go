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
	"github.com/maguowei/gotobeta/internal/pkg/apperr"
	"github.com/maguowei/gotobeta/internal/pkg/auth"
)

// errUC 是可配置错误的 usecase fake：所有方法统一返回 err。
type errUC struct{ err error }

func (u errUC) Presign(_ context.Context, _ mediacmd.PresignAttachmentCommand) (*mediaresult.PresignResult, error) {
	return nil, u.err
}
func (u errUC) Commit(_ context.Context, _ mediacmd.CommitAttachmentCommand) (*mediaresult.AttachmentResult, error) {
	return nil, u.err
}

// newErrRouter 构造一个注入 claims、但 usecase 固定返回 err 的路由。
func newErrRouter(err error) *gin.Engine {
	gin.SetMode(gin.TestMode)
	h := mediahandler.NewAttachmentHandler(errUC{err: err})
	e := gin.New()
	authMW := func(c *gin.Context) {
		c.Request = c.Request.WithContext(auth.WithClaims(c.Request.Context(), &auth.Claims{UserID: 9}))
		c.Next()
	}
	mediarouter.RegisterRoutes(e.Group("/api/v1"), h, authMW)
	return e
}

// newNoAuthRouter 构造一个不注入 claims 的路由，用于验证缺失认证时的 401 分支。
func newNoAuthRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	h := mediahandler.NewAttachmentHandler(fakeUC{})
	e := gin.New()
	passthrough := func(c *gin.Context) { c.Next() }
	mediarouter.RegisterRoutes(e.Group("/api/v1"), h, passthrough)
	return e
}

// sendReq 发送请求并返回状态码。
func sendReq(t *testing.T, e *gin.Engine, method, path, body string) int {
	t.Helper()
	var r *http.Request
	if body != "" {
		r = httptest.NewRequest(method, path, strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
	} else {
		r = httptest.NewRequest(method, path, nil)
	}
	w := httptest.NewRecorder()
	e.ServeHTTP(w, r)
	return w.Code
}

// 覆盖 Presign / Commit 的 usecase 错误分支：domain error 应映射到对应 HTTP status。
func TestMediaUseCaseErrors(t *testing.T) {
	endpoints := []struct {
		name, method, path, body string
	}{
		{"presign", "POST", "/api/v1/attachments/presign", `{"workspaceId":"1","fileName":"a.png","contentType":"image/png","sizeBytes":10}`},
		{"commit", "POST", "/api/v1/attachments/5001/commit", `{"workspaceId":"1"}`},
	}
	kinds := []struct {
		name string
		err  error
		want int
	}{
		{"notFound", apperr.NotFound("不存在"), http.StatusNotFound},
		{"forbidden", apperr.Forbidden("无权限"), http.StatusForbidden},
		{"conflict", apperr.Conflict("冲突"), http.StatusUnprocessableEntity},
		{"internal", apperr.Internal("内部错误", nil), http.StatusInternalServerError},
	}
	for _, k := range kinds {
		e := newErrRouter(k.err)
		for _, ep := range endpoints {
			if code := sendReq(t, e, ep.method, ep.path, ep.body); code != k.want {
				t.Errorf("%s/%s 应返回 %d, got %d", k.name, ep.name, k.want, code)
			}
		}
	}
}

// 缺失 claims 时 handler 应在 RequireClaims 处返回 401。
func TestMediaMissingClaims(t *testing.T) {
	e := newNoAuthRouter()
	endpoints := []struct {
		method, path, body string
	}{
		{"POST", "/api/v1/attachments/presign", `{"workspaceId":"1","fileName":"a.png","contentType":"image/png","sizeBytes":10}`},
		{"POST", "/api/v1/attachments/5001/commit", `{"workspaceId":"1"}`},
	}
	for _, ep := range endpoints {
		if code := sendReq(t, e, ep.method, ep.path, ep.body); code != http.StatusUnauthorized {
			t.Errorf("%s %s 缺 claims 应 401, got %d", ep.method, ep.path, code)
		}
	}
}

// 覆盖 body 解析错误分支。
func TestMediaInvalidBody(t *testing.T) {
	e := newRouter()
	cases := []struct {
		name, method, path, body string
	}{
		{"presign body 错误", "POST", "/api/v1/attachments/presign", `{`},
		{"commit body 错误", "POST", "/api/v1/attachments/5001/commit", `{`},
	}
	for _, tc := range cases {
		if code := sendReq(t, e, tc.method, tc.path, tc.body); code != http.StatusBadRequest {
			t.Errorf("%s 应 400, got %d", tc.name, code)
		}
	}
}
