package persistence

import (
	"log/slog"
	"testing"
	"time"

	"github.com/maguowei/gotobeta/internal/ent"
	"github.com/maguowei/gotobeta/internal/modules/todo/domain/todo"
)

func TestNewTodoRepositoryStoresDependencies(t *testing.T) {
	t.Parallel()

	logger := slog.Default()
	repo := NewTodoRepository(nil, logger)
	if repo == nil {
		t.Fatal("NewTodoRepository() = nil")
	}
	if repo.client != nil {
		t.Fatalf("client = %v, want nil", repo.client)
	}
	if repo.logger != logger {
		t.Fatalf("logger = %v, want injected logger", repo.logger)
	}
}

func TestToEntityMapsEntTodo(t *testing.T) {
	t.Parallel()

	createdAt := time.Date(2026, 5, 29, 10, 11, 12, 0, time.UTC)
	updatedAt := time.Date(2026, 5, 29, 13, 14, 15, 0, time.UTC)

	got := toEntity(&ent.Todo{
		BizID:     42,
		Title:     "write tests",
		Status:    string(todo.StatusDone),
		Version:   3,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	})

	if got.ID() != 42 {
		t.Fatalf("ID = %d, want 42", got.ID())
	}
	if got.Version() != 3 {
		t.Fatalf("Version = %d, want 3", got.Version())
	}
	if got.Title().String() != "write tests" {
		t.Fatalf("Title = %q, want write tests", got.Title().String())
	}
	if got.Status() != todo.StatusDone {
		t.Fatalf("Status = %q, want done", got.Status())
	}
	if !got.CreatedAt().Equal(createdAt) {
		t.Fatalf("CreatedAt = %s, want %s", got.CreatedAt(), createdAt)
	}
	if !got.UpdatedAt().Equal(updatedAt) {
		t.Fatalf("UpdatedAt = %s, want %s", got.UpdatedAt(), updatedAt)
	}
}
