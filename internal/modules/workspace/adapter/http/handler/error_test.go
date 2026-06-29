package handler_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	workspacehandler "github.com/maguowei/gotobeta/internal/modules/workspace/adapter/http/handler"
	workspacerouter "github.com/maguowei/gotobeta/internal/modules/workspace/adapter/http/router"
	workspacecmd "github.com/maguowei/gotobeta/internal/modules/workspace/application/command"
	workspacequery "github.com/maguowei/gotobeta/internal/modules/workspace/application/query"
	workspaceresult "github.com/maguowei/gotobeta/internal/modules/workspace/application/result"
	"github.com/maguowei/gotobeta/internal/pkg/apperr"
	"github.com/maguowei/gotobeta/internal/pkg/auth"
)

// errUC 是可配置错误的 usecase fake：所有方法统一返回 err。
type errUC struct{ err error }

func (u errUC) CreateWorkspace(_ context.Context, _ workspacecmd.CreateWorkspaceCommand) (*workspaceresult.WorkspaceResult, error) {
	return nil, u.err
}
func (u errUC) ListMyWorkspaces(_ context.Context, _ workspacequery.ListMyWorkspacesQuery) ([]*workspaceresult.WorkspaceResult, error) {
	return nil, u.err
}
func (u errUC) InviteMember(_ context.Context, _ workspacecmd.InviteMemberCommand) (*workspaceresult.MemberResult, error) {
	return nil, u.err
}
func (u errUC) AssignRole(_ context.Context, _ workspacecmd.AssignRoleCommand) error { return u.err }
func (u errUC) ListRoles(_ context.Context, _ workspacequery.ListRolesQuery) ([]*workspaceresult.RoleResult, error) {
	return nil, u.err
}

// newErrRouter 构造一个注入 claims、但 usecase 固定返回 err 的路由。
func newErrRouter(err error) *gin.Engine {
	gin.SetMode(gin.TestMode)
	h := workspacehandler.NewWorkspaceHandler(errUC{err: err})
	e := gin.New()
	authMW := func(c *gin.Context) {
		c.Request = c.Request.WithContext(auth.WithClaims(c.Request.Context(), &auth.Claims{UserID: 9}))
		c.Next()
	}
	workspacerouter.RegisterRoutes(e.Group("/api/v1"), h, authMW)
	return e
}

// newNoAuthRouter 构造一个不注入 claims 的路由，用于验证缺失认证时的 401 分支。
func newNoAuthRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	h := workspacehandler.NewWorkspaceHandler(fakeUC{})
	e := gin.New()
	passthrough := func(c *gin.Context) { c.Next() }
	workspacerouter.RegisterRoutes(e.Group("/api/v1"), h, passthrough)
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

// 覆盖每个 handler 的 usecase 错误分支：domain error 应映射到对应 HTTP status。
func TestWorkspaceUseCaseErrors(t *testing.T) {
	endpoints := []struct {
		name, method, path, body string
	}{
		{"create", "POST", "/api/v1/workspaces", `{"slug":"acme","name":"Acme"}`},
		{"list", "GET", "/api/v1/workspaces", ""},
		{"invite", "POST", "/api/v1/workspaces/1/members", `{"userId":"7","roleCode":"member"}`},
		{"assign", "POST", "/api/v1/workspaces/1/members/7/roles", `{"roleCode":"admin"}`},
		{"roles", "GET", "/api/v1/workspaces/1/roles", ""},
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

// 缺失 claims 时所有 handler 都应在 RequireClaims 处返回 401。
func TestWorkspaceMissingClaims(t *testing.T) {
	e := newNoAuthRouter()
	endpoints := []struct {
		method, path, body string
	}{
		{"POST", "/api/v1/workspaces", `{"slug":"acme","name":"Acme"}`},
		{"GET", "/api/v1/workspaces", ""},
		{"POST", "/api/v1/workspaces/1/members", `{"userId":"7","roleCode":"member"}`},
		{"POST", "/api/v1/workspaces/1/members/7/roles", `{"roleCode":"admin"}`},
		{"GET", "/api/v1/workspaces/1/roles", ""},
	}
	for _, ep := range endpoints {
		if code := sendReq(t, e, ep.method, ep.path, ep.body); code != http.StatusUnauthorized {
			t.Errorf("%s %s 缺 claims 应 401, got %d", ep.method, ep.path, code)
		}
	}
}

// 覆盖参数与 body 解析错误分支。
func TestWorkspaceExtraInvalidParams(t *testing.T) {
	e := newRouter()
	cases := []struct {
		name, method, path, body string
	}{
		{"create body 错误", "POST", "/api/v1/workspaces", `{`},
		{"invite body 错误", "POST", "/api/v1/workspaces/1/members", `{`},
		{"非法 uid", "POST", "/api/v1/workspaces/1/members/abc/roles", `{"roleCode":"admin"}`},
		{"assign body 错误", "POST", "/api/v1/workspaces/1/members/7/roles", `{`},
		{"非法 ws ID(assign)", "POST", "/api/v1/workspaces/abc/members/7/roles", `{"roleCode":"admin"}`},
		{"非法 ws ID(roles)", "GET", "/api/v1/workspaces/abc/roles", ""},
	}
	for _, tc := range cases {
		if code := sendReq(t, e, tc.method, tc.path, tc.body); code != http.StatusBadRequest {
			t.Errorf("%s 应 400, got %d", tc.name, code)
		}
	}
}
