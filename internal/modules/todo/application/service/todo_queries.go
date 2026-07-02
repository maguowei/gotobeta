package service

import (
	"context"
	stderrors "errors"

	todoquery "github.com/maguowei/gotobeta/internal/modules/todo/application/query"
	todoresult "github.com/maguowei/gotobeta/internal/modules/todo/application/result"
	"github.com/maguowei/gotobeta/internal/modules/todo/domain/todo"
	"github.com/maguowei/gotobeta/internal/pkg/apperr"
)

// GetTodo 查询单个待办。
func (s *TodoService) GetTodo(ctx context.Context, q todoquery.GetTodoQuery) (*todoresult.TodoResult, error) {
	item, err := s.repository.FindByID(ctx, q.ID)
	if err != nil {
		if stderrors.Is(err, todo.ErrNotFound) {
			return nil, apperr.NotFound("待办不存在")
		}
		return nil, apperr.WrapInternal("查询待办失败", err)
	}
	return toResult(item), nil
}

// ListTodos 列出待办；ListTodosQuery 预留分页、过滤等扩展字段位。
func (s *TodoService) ListTodos(ctx context.Context, q todoquery.ListTodosQuery) ([]*todoresult.TodoResult, error) {
	_ = q // ListTodosQuery 目前为空结构体，字段位预留给分页与过滤参数。
	items, err := s.repository.List(ctx)
	if err != nil {
		return nil, apperr.WrapInternal("查询待办失败", err)
	}

	results := make([]*todoresult.TodoResult, 0, len(items))
	for _, item := range items {
		results = append(results, toResult(item))
	}

	return results, nil
}
