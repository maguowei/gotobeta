package router

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/maguowei/gotobeta/internal/modules/todo/adapter/http/handler"
	todocmd "github.com/maguowei/gotobeta/internal/modules/todo/application/command"
	todoquery "github.com/maguowei/gotobeta/internal/modules/todo/application/query"
	todoresult "github.com/maguowei/gotobeta/internal/modules/todo/application/result"
)

type routeUseCase struct{}

func (routeUseCase) CreateTodo(context.Context, todocmd.CreateTodoCommand) (*todoresult.TodoResult, error) {
	return &todoresult.TodoResult{ID: 1, Title: "write tests", Status: "pending"}, nil
}

func (routeUseCase) GetTodo(context.Context, todoquery.GetTodoQuery) (*todoresult.TodoResult, error) {
	return &todoresult.TodoResult{ID: 1, Title: "write tests", Status: "pending"}, nil
}

func (routeUseCase) CompleteTodo(context.Context, todocmd.CompleteTodoCommand) (*todoresult.TodoResult, error) {
	return &todoresult.TodoResult{ID: 1, Title: "write tests", Status: "done"}, nil
}

func (routeUseCase) DeleteTodo(context.Context, todocmd.DeleteTodoCommand) error { return nil }

func (routeUseCase) ListTodos(context.Context, todoquery.ListTodosQuery) ([]*todoresult.TodoResult, error) {
	return []*todoresult.TodoResult{
		{ID: 1, Title: "write tests", Status: "pending"},
	}, nil
}

func TestRegisterRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)

	engine := gin.New()
	RegisterRoutes(engine.Group(""), handler.NewTodoHandler(routeUseCase{}))

	tests := []struct {
		method string
		path   string
		body   string
	}{
		{method: http.MethodGet, path: "/todos"},
		{method: http.MethodGet, path: "/todos/1"},
		{method: http.MethodPost, path: "/todos", body: `{"title":"write tests"}`},
		{method: http.MethodPost, path: "/todos/1/complete"},
		{method: http.MethodDelete, path: "/todos/1"},
	}

	for _, tt := range tests {
		req := httptest.NewRequestWithContext(t.Context(), tt.method, tt.path, http.NoBody)
		if tt.body != "" {
			req = httptest.NewRequestWithContext(t.Context(), tt.method, tt.path, strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
		}
		recorder := httptest.NewRecorder()
		engine.ServeHTTP(recorder, req)
		if recorder.Code != http.StatusOK {
			t.Fatalf("%s %s status = %d, want 200; body=%s", tt.method, tt.path, recorder.Code, recorder.Body.String())
		}
	}
}

func TestRegisterRoutesAppliesMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// 回归保护：传入的中间件必须作用于全部 Todo 路由（含写操作），
	// 否则在启用 user-auth 的服务里 todo 端点会变成公开可写。
	engine := gin.New()
	blocked := func(c *gin.Context) {
		c.AbortWithStatus(http.StatusUnauthorized)
	}
	RegisterRoutes(engine.Group(""), handler.NewTodoHandler(routeUseCase{}), blocked)

	for _, tt := range []struct {
		method string
		path   string
	}{
		{method: http.MethodGet, path: "/todos"},
		{method: http.MethodPost, path: "/todos"},
		{method: http.MethodDelete, path: "/todos/1"},
	} {
		req := httptest.NewRequestWithContext(t.Context(), tt.method, tt.path, http.NoBody)
		recorder := httptest.NewRecorder()
		engine.ServeHTTP(recorder, req)
		if recorder.Code != http.StatusUnauthorized {
			t.Fatalf("%s %s status = %d, want 401 (middleware must guard the route)", tt.method, tt.path, recorder.Code)
		}
	}
}
