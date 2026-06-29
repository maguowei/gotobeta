package service

import (
	"context"
	stderrors "errors"
	"log/slog"

	todoresult "github.com/maguowei/gotobeta/internal/modules/todo/application/result"
	"github.com/maguowei/gotobeta/internal/modules/todo/domain/todo"
	"github.com/maguowei/gotobeta/internal/pkg/apperr"
	"github.com/maguowei/gotobeta/internal/pkg/idgen"
	"github.com/maguowei/gotobeta/internal/pkg/persistence"
)

// TodoService 编排 Todo 用例。
type TodoService struct {
	repository  todo.Repository
	idGenerator idgen.Generator
	txRunner    persistence.TxRunner
	logger      *slog.Logger
}

// NewTodoService 创建服务。
func NewTodoService(
	repository todo.Repository,
	idGenerator idgen.Generator,
	txRunner persistence.TxRunner,
	logger *slog.Logger,
) *TodoService {
	return &TodoService{
		repository:  repository,
		idGenerator: idGenerator,
		txRunner:    txRunner,
		logger:      logger,
	}
}

func toResult(t *todo.Todo) *todoresult.TodoResult {
	return &todoresult.TodoResult{
		ID:        t.ID(),
		Title:     t.Title().String(),
		Status:    string(t.Status()),
		CreatedAt: t.CreatedAt(),
		UpdatedAt: t.UpdatedAt(),
	}
}

func wrapInfrastructureError(message string, err error) error {
	if stderrors.Is(err, context.Canceled) || stderrors.Is(err, context.DeadlineExceeded) {
		return err
	}
	return apperr.Internal(message, err)
}
