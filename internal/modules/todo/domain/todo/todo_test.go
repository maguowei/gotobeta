package todo

import "testing"

func TestNewRejectsEmptyTitle(t *testing.T) {
	if _, err := New(1, "   "); err == nil {
		t.Fatal("New() error = nil, want validation error")
	}
}

func TestNewTrimsTitleAndSetsDefaultFields(t *testing.T) {
	todo, err := New(42, "  buy milk  ")
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if todo.ID() != 42 {
		t.Fatalf("todo.ID() = %d, want 42", todo.ID())
	}

	if todo.Title().String() != "buy milk" {
		t.Fatalf("todo.Title().String() = %q, want buy milk", todo.Title().String())
	}

	if todo.Status() != StatusPending {
		t.Fatalf("todo.Status() = %q, want pending", todo.Status())
	}

	if todo.Version() != 1 {
		t.Fatalf("todo.Version() = %d, want 1", todo.Version())
	}

	if todo.CreatedAt().IsZero() {
		t.Fatal("todo.CreatedAt() is zero")
	}

	if todo.UpdatedAt().IsZero() {
		t.Fatal("todo.UpdatedAt() is zero")
	}
}

func TestTodo_Complete(t *testing.T) {
	todo, _ := New(1, "test")

	if err := todo.Complete(); err != nil {
		t.Fatalf("Complete() error = %v", err)
	}

	if !todo.IsDone() {
		t.Fatal("IsDone() = false after Complete()")
	}

	if err := todo.Complete(); err == nil {
		t.Fatal("Complete() on done todo should return error")
	}
}

func TestTodo_UpdateTitle(t *testing.T) {
	todo, _ := New(1, "old title")

	if err := todo.UpdateTitle("new title"); err != nil {
		t.Fatalf("UpdateTitle() error = %v", err)
	}

	if todo.Title().String() != "new title" {
		t.Fatalf("Title() = %q, want new title", todo.Title().String())
	}

	if err := todo.UpdateTitle("   "); err == nil {
		t.Fatal("UpdateTitle() with empty title should return error")
	}
}
