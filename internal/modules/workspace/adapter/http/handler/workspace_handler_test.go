package handler_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	workspacehandler "github.com/maguowei/gotobeta/internal/modules/workspace/adapter/http/handler"
	workspacerouter "github.com/maguowei/gotobeta/internal/modules/workspace/adapter/http/router"
	workspacecmd "github.com/maguowei/gotobeta/internal/modules/workspace/application/command"
	workspacequery "github.com/maguowei/gotobeta/internal/modules/workspace/application/query"
	workspaceresult "github.com/maguowei/gotobeta/internal/modules/workspace/application/result"
	"github.com/maguowei/gotobeta/internal/pkg/auth"
)

type fakeUC struct{}

func (fakeUC) CreateWorkspace(_ context.Context, _ workspacecmd.CreateWorkspaceCommand) (*workspaceresult.WorkspaceResult, error) {
	return &workspaceresult.WorkspaceResult{ID: 1, Slug: "acme", Name: "Acme", OwnerUserID: 9, Status: 1, CreatedAt: time.Now()}, nil
}
func (fakeUC) ListMyWorkspaces(_ context.Context, _ workspacequery.ListMyWorkspacesQuery) ([]*workspaceresult.WorkspaceResult, error) {
	return []*workspaceresult.WorkspaceResult{{ID: 1, CreatedAt: time.Now()}}, nil
}
func (fakeUC) InviteMember(_ context.Context, _ workspacecmd.InviteMemberCommand) (*workspaceresult.MemberResult, error) {
	return &workspaceresult.MemberResult{WorkspaceID: 1, UserID: 7, Status: 1, JoinedAt: time.Now()}, nil
}
func (fakeUC) AssignRole(_ context.Context, _ workspacecmd.AssignRoleCommand) error { return nil }
func (fakeUC) ListRoles(_ context.Context, _ workspacequery.ListRolesQuery) ([]*workspaceresult.RoleResult, error) {
	return []*workspaceresult.RoleResult{{ID: 2, Code: "admin", Name: "管理员"}}, nil
}

func newRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	h := workspacehandler.NewWorkspaceHandler(fakeUC{})
	e := gin.New()
	authMW := func(c *gin.Context) {
		c.Request = c.Request.WithContext(auth.WithClaims(c.Request.Context(), &auth.Claims{UserID: 9}))
		c.Next()
	}
	workspacerouter.RegisterRoutes(e.Group("/api/v1"), h, authMW)
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

func TestWorkspaceRoutes(t *testing.T) {
	e := newRouter()
	if code := do(t, e, "POST", "/api/v1/workspaces", `{"slug":"acme","name":"Acme"}`); code != http.StatusOK {
		t.Errorf("create 应 200, got %d", code)
	}
	if code := do(t, e, "GET", "/api/v1/workspaces", ``); code != http.StatusOK {
		t.Errorf("list 应 200, got %d", code)
	}
	if code := do(t, e, "POST", "/api/v1/workspaces/1/members", `{"userId":"7","roleCode":"member"}`); code != http.StatusOK {
		t.Errorf("invite 应 200, got %d", code)
	}
	if code := do(t, e, "POST", "/api/v1/workspaces/1/members/7/roles", `{"roleCode":"admin"}`); code != http.StatusOK {
		t.Errorf("assign 应 200, got %d", code)
	}
	if code := do(t, e, "GET", "/api/v1/workspaces/1/roles", ``); code != http.StatusOK {
		t.Errorf("roles 应 200, got %d", code)
	}
	if code := do(t, e, "POST", "/api/v1/workspaces/abc/members", `{"userId":"7","roleCode":"member"}`); code != http.StatusBadRequest {
		t.Errorf("非法 ws ID 应 400, got %d", code)
	}
}
