package service

import (
	"context"
	stderrors "errors"
	"log/slog"

	todocmd "github.com/maguowei/gotobeta/internal/modules/todo/application/command"
	todoresult "github.com/maguowei/gotobeta/internal/modules/todo/application/result"
	"github.com/maguowei/gotobeta/internal/modules/todo/domain/todo"
	"github.com/maguowei/gotobeta/internal/pkg/apperr"
	loggerx "github.com/maguowei/gotobeta/internal/pkg/logger"
)

// CreateTodo 创建待办。
func (s *TodoService) CreateTodo(ctx context.Context, cmd todocmd.CreateTodoCommand) (*todoresult.TodoResult, error) {
	todoID, err := s.idGenerator.NextID(ctx)
	if err != nil {
		return nil, wrapInfrastructureError("生成待办 ID 失败", err)
	}

	item, err := todo.New(todoID, cmd.Title)
	if err != nil {
		return nil, err
	}

	err = s.txRunner.RunInTx(ctx, func(txCtx context.Context) error {
		if err := s.repository.Create(txCtx, item); err != nil {
			return wrapInfrastructureError("保存待办失败", err)
		}

		return nil
	})
	if err != nil {
		// 在 service 边界记一次结构化错误日志：DomainError 自动展开 errKind/errCode/errCause，
		// 上层 handler 仅做 HTTP 转换，不再重复 ErrorContext。
		loggerx.WithError(ctx, s.logger, "create todo failed", err, slog.Int64("todoId", todoID), slog.String("title", cmd.Title))
		return nil, err
	}

	s.logger.InfoContext(ctx, "todo created", slog.Int64("todoId", item.ID()))
	return toResult(item), nil
}

// CompleteTodo 完成待办。
func (s *TodoService) CompleteTodo(ctx context.Context, cmd todocmd.CompleteTodoCommand) (*todoresult.TodoResult, error) {
	var completed *todo.Todo
	err := s.txRunner.RunInTx(ctx, func(txCtx context.Context) error {
		item, err := s.repository.FindByID(txCtx, cmd.ID)
		if err != nil {
			if stderrors.Is(err, todo.ErrNotFound) {
				return apperr.NotFound("待办不存在")
			}
			return wrapInfrastructureError("查询待办失败", err)
		}

		if err := item.Complete(); err != nil {
			return err
		}

		if err := s.repository.Save(txCtx, item); err != nil {
			if stderrors.Is(err, todo.ErrNotFound) {
				return apperr.NotFound("待办不存在")
			}
			if stderrors.Is(err, todo.ErrConflict) {
				return apperr.Conflict("待办已被并发修改，请重试")
			}
			return wrapInfrastructureError("保存待办失败", err)
		}

		completed = item
		return nil
	})
	if err != nil {
		loggerx.WithError(ctx, s.logger, "complete todo failed", err, slog.Int64("todoId", cmd.ID))
		return nil, err
	}

	s.logger.InfoContext(ctx, "todo completed", slog.Int64("todoId", cmd.ID))
	return toResult(completed), nil
}

// DeleteTodo 删除待办。
func (s *TodoService) DeleteTodo(ctx context.Context, cmd todocmd.DeleteTodoCommand) error {
	if err := s.txRunner.RunInTx(ctx, func(txCtx context.Context) error {
		if err := s.repository.Delete(txCtx, cmd.ID); err != nil {
			if stderrors.Is(err, todo.ErrNotFound) {
				return apperr.NotFound("待办不存在")
			}
			return wrapInfrastructureError("删除待办失败", err)
		}
		return nil
	}); err != nil {
		loggerx.WithError(ctx, s.logger, "delete todo failed", err, slog.Int64("todoId", cmd.ID))
		return err
	}
	s.logger.InfoContext(ctx, "todo deleted", slog.Int64("todoId", cmd.ID))
	return nil
}
