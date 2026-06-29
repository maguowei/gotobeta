package handler

import (
	"context"
	"encoding/json"
	stderrors "errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	todocmd "github.com/maguowei/gotobeta/internal/modules/todo/application/command"
	todoquery "github.com/maguowei/gotobeta/internal/modules/todo/application/query"
	todoresult "github.com/maguowei/gotobeta/internal/modules/todo/application/result"
	todosvc "github.com/maguowei/gotobeta/internal/modules/todo/application/service"
	"github.com/maguowei/gotobeta/internal/pkg/apperr"
)

// 编译期断言：应用服务实现必须满足 handler 声明的用例接口，
// 方法签名漂移在最近的修改点直接编译失败。
var _ TodoUseCase = (*todosvc.TodoService)(nil)

type mockTodoUseCase struct {
	createCmd   todocmd.CreateTodoCommand
	getID       int64
	completeID  int64
	deleteID    int64
	created     *todoresult.TodoResult
	items       []*todoresult.TodoResult
	createErr   error
	getErr      error
	completeErr error
	deleteErr   error
	listErr     error
}

func (s *mockTodoUseCase) CreateTodo(_ context.Context, cmd todocmd.CreateTodoCommand) (*todoresult.TodoResult, error) {
	s.createCmd = cmd
	return s.created, s.createErr
}

func (s *mockTodoUseCase) GetTodo(_ context.Context, q todoquery.GetTodoQuery) (*todoresult.TodoResult, error) {
	s.getID = q.ID
	return s.created, s.getErr
}

func (s *mockTodoUseCase) CompleteTodo(_ context.Context, cmd todocmd.CompleteTodoCommand) (*todoresult.TodoResult, error) {
	s.completeID = cmd.ID
	return s.created, s.completeErr
}

func (s *mockTodoUseCase) DeleteTodo(_ context.Context, cmd todocmd.DeleteTodoCommand) error {
	s.deleteID = cmd.ID
	return s.deleteErr
}

func (s *mockTodoUseCase) ListTodos(context.Context, todoquery.ListTodosQuery) ([]*todoresult.TodoResult, error) {
	if s.items != nil {
		return s.items, s.listErr
	}
	return []*todoresult.TodoResult{s.created}, s.listErr
}

func TestTodoHandlerCreateTodoUsesUseCase(t *testing.T) {
	gin.SetMode(gin.TestMode)

	usecase := &mockTodoUseCase{
		created: &todoresult.TodoResult{
			ID:        12,
			Title:     "write tests",
			Status:    "pending",
			CreatedAt: time.Date(2026, 5, 17, 1, 2, 3, 0, time.UTC),
			UpdatedAt: time.Date(2026, 5, 17, 1, 2, 3, 0, time.UTC),
		},
	}
	handler := NewTodoHandler(usecase)
	router := gin.New()
	router.POST("/todos", handler.CreateTodo)

	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/todos", strings.NewReader(`{"title":"write tests"}`))
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", recorder.Code, recorder.Body.String())
	}
	if usecase.createCmd.Title != "write tests" {
		t.Fatalf("use case command title = %q, want write tests", usecase.createCmd.Title)
	}

	var body map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal body: %v", err)
	}
	if body["code"] != float64(0) {
		t.Fatalf("code = %v, want 0", body["code"])
	}
}

func TestTodoHandlerRejectsInvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewTodoHandler(&mockTodoUseCase{})
	router := gin.New()
	router.GET("/todos/:id", handler.GetTodo)

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/todos/not-a-number", nil)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", recorder.Code)
	}
}

func TestTodoHandlerRejectsNonPositiveID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewTodoHandler(&mockTodoUseCase{})
	router := gin.New()
	router.GET("/todos/:id", handler.GetTodo)

	for _, path := range []string{"/todos/0", "/todos/-1"} {
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, httptest.NewRequestWithContext(t.Context(), http.MethodGet, path, nil))

		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("%s status = %d, want 400", path, recorder.Code)
		}
	}
}

func TestTodoHandlerListGetCompleteAndDeleteSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)

	output := &todoresult.TodoResult{
		ID:        12,
		Title:     "write tests",
		Status:    "pending",
		CreatedAt: time.Date(2026, 5, 17, 1, 2, 3, 0, time.UTC),
		UpdatedAt: time.Date(2026, 5, 17, 1, 2, 3, 0, time.UTC),
	}
	usecase := &mockTodoUseCase{created: output, items: []*todoresult.TodoResult{output}}
	h := NewTodoHandler(usecase)
	router := gin.New()
	router.GET("/todos", h.ListTodos)
	router.GET("/todos/:id", h.GetTodo)
	router.POST("/todos/:id/complete", h.CompleteTodo)
	router.DELETE("/todos/:id", h.DeleteTodo)

	tests := []struct {
		method string
		path   string
	}{
		{method: http.MethodGet, path: "/todos"},
		{method: http.MethodGet, path: "/todos/12"},
		{method: http.MethodPost, path: "/todos/12/complete"},
		{method: http.MethodDelete, path: "/todos/12"},
	}

	for _, tt := range tests {
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, httptest.NewRequestWithContext(t.Context(), tt.method, tt.path, nil))
		if recorder.Code != http.StatusOK {
			t.Fatalf("%s %s status = %d, want 200; body=%s", tt.method, tt.path, recorder.Code, recorder.Body.String())
		}
	}

	if usecase.getID != 12 || usecase.completeID != 12 || usecase.deleteID != 12 {
		t.Fatalf("captured ids = get:%d complete:%d delete:%d, want all 12", usecase.getID, usecase.completeID, usecase.deleteID)
	}
}

func TestTodoHandlerCreateTodoErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name string
		body string
		err  error
		want int
	}{
		{name: "invalid json", body: `{`, want: http.StatusBadRequest},
		{name: "service error", body: `{"title":"write tests"}`, err: apperr.Internal("create failed", stderrors.New("insert failed")), want: http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewTodoHandler(&mockTodoUseCase{createErr: tt.err})
			router := gin.New()
			router.POST("/todos", h.CreateTodo)

			req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/todos", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, req)

			if recorder.Code != tt.want {
				t.Fatalf("status = %d, want %d; body=%s", recorder.Code, tt.want, recorder.Body.String())
			}
		})
	}
}

func TestTodoHandlerUseCaseErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name   string
		method string
		path   string
		setup  func(*mockTodoUseCase)
		route  func(*gin.Engine, *TodoHandler)
		want   int
	}{
		{
			name:   "list error",
			method: http.MethodGet,
			path:   "/todos",
			setup:  func(s *mockTodoUseCase) { s.listErr = apperr.Internal("list failed", nil) },
			route:  func(r *gin.Engine, h *TodoHandler) { r.GET("/todos", h.ListTodos) },
			want:   http.StatusInternalServerError,
		},
		{
			name:   "get not found",
			method: http.MethodGet,
			path:   "/todos/12",
			setup:  func(s *mockTodoUseCase) { s.getErr = apperr.NotFound("missing") },
			route:  func(r *gin.Engine, h *TodoHandler) { r.GET("/todos/:id", h.GetTodo) },
			want:   http.StatusNotFound,
		},
		{
			name:   "complete error",
			method: http.MethodPost,
			path:   "/todos/12/complete",
			setup:  func(s *mockTodoUseCase) { s.completeErr = apperr.Internal("complete failed", nil) },
			route:  func(r *gin.Engine, h *TodoHandler) { r.POST("/todos/:id/complete", h.CompleteTodo) },
			want:   http.StatusInternalServerError,
		},
		{
			name:   "delete error",
			method: http.MethodDelete,
			path:   "/todos/12",
			setup:  func(s *mockTodoUseCase) { s.deleteErr = apperr.Internal("delete failed", nil) },
			route:  func(r *gin.Engine, h *TodoHandler) { r.DELETE("/todos/:id", h.DeleteTodo) },
			want:   http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			usecase := &mockTodoUseCase{}
			tt.setup(usecase)
			h := NewTodoHandler(usecase)
			router := gin.New()
			tt.route(router, h)

			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, httptest.NewRequestWithContext(t.Context(), tt.method, tt.path, nil))
			if recorder.Code != tt.want {
				t.Fatalf("status = %d, want %d; body=%s", recorder.Code, tt.want, recorder.Body.String())
			}
		})
	}
}
