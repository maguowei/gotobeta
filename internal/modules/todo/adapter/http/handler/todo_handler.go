package handler

import (
	"context"
	"strconv"

	"github.com/gin-gonic/gin"

	todoreq "github.com/maguowei/gotobeta/internal/modules/todo/adapter/http/request"
	todoresp "github.com/maguowei/gotobeta/internal/modules/todo/adapter/http/response"
	todocmd "github.com/maguowei/gotobeta/internal/modules/todo/application/command"
	todoquery "github.com/maguowei/gotobeta/internal/modules/todo/application/query"
	todoresult "github.com/maguowei/gotobeta/internal/modules/todo/application/result"
	httpresponse "github.com/maguowei/gotobeta/internal/pkg/httpx/response"
)

// TodoUseCase 定义 handler 层对应用用例的依赖。
type TodoUseCase interface {
	CreateTodo(ctx context.Context, cmd todocmd.CreateTodoCommand) (*todoresult.TodoResult, error)
	GetTodo(ctx context.Context, q todoquery.GetTodoQuery) (*todoresult.TodoResult, error)
	CompleteTodo(ctx context.Context, cmd todocmd.CompleteTodoCommand) (*todoresult.TodoResult, error)
	DeleteTodo(ctx context.Context, cmd todocmd.DeleteTodoCommand) error
	ListTodos(ctx context.Context, q todoquery.ListTodosQuery) ([]*todoresult.TodoResult, error)
}

// TodoHandler 处理 Todo HTTP 请求。
type TodoHandler struct {
	usecase TodoUseCase
}

// NewTodoHandler 创建 Handler。
func NewTodoHandler(uc TodoUseCase) *TodoHandler {
	return &TodoHandler{usecase: uc}
}

// ListTodos 列表查询。
func (h *TodoHandler) ListTodos(c *gin.Context) {
	items, err := h.usecase.ListTodos(c.Request.Context(), todoquery.ListTodosQuery{})
	if err != nil {
		httpresponse.Error(c, err)
		return
	}
	httpresponse.Success(c, todoresp.ToTodoListResponse(items))
}

// GetTodo 查询单个待办。
func (h *TodoHandler) GetTodo(c *gin.Context) {
	id, err := parsePositiveID(c.Param("id"))
	if err != nil {
		httpresponse.ErrorWithCode(c, httpresponse.CodeInvalidParam, "无效的待办 ID")
		return
	}
	item, err := h.usecase.GetTodo(c.Request.Context(), todoquery.GetTodoQuery{ID: id})
	if err != nil {
		httpresponse.Error(c, err)
		return
	}
	httpresponse.Success(c, todoresp.ToTodoResponse(item))
}

// CreateTodo 创建待办。
func (h *TodoHandler) CreateTodo(c *gin.Context) {
	var req todoreq.CreateTodoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresponse.ErrorWithCode(c, httpresponse.CodeInvalidParam, "请求参数格式错误")
		return
	}
	item, err := h.usecase.CreateTodo(c.Request.Context(), req.ToCommand())
	if err != nil {
		httpresponse.Error(c, err)
		return
	}
	httpresponse.Success(c, todoresp.ToTodoResponse(item))
}

// CompleteTodo 完成待办。
func (h *TodoHandler) CompleteTodo(c *gin.Context) {
	id, err := parsePositiveID(c.Param("id"))
	if err != nil {
		httpresponse.ErrorWithCode(c, httpresponse.CodeInvalidParam, "无效的待办 ID")
		return
	}
	item, err := h.usecase.CompleteTodo(c.Request.Context(), todocmd.CompleteTodoCommand{ID: id})
	if err != nil {
		httpresponse.Error(c, err)
		return
	}
	httpresponse.Success(c, todoresp.ToTodoResponse(item))
}

// DeleteTodo 删除待办。
func (h *TodoHandler) DeleteTodo(c *gin.Context) {
	id, err := parsePositiveID(c.Param("id"))
	if err != nil {
		httpresponse.ErrorWithCode(c, httpresponse.CodeInvalidParam, "无效的待办 ID")
		return
	}
	if err := h.usecase.DeleteTodo(c.Request.Context(), todocmd.DeleteTodoCommand{ID: id}); err != nil {
		httpresponse.Error(c, err)
		return
	}
	httpresponse.Success(c, nil)
}

func parsePositiveID(raw string) (int64, error) {
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, err
	}
	if id <= 0 {
		return 0, strconv.ErrSyntax
	}
	return id, nil
}
