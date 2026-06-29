package response

import (
	"testing"
	"time"

	todoresult "github.com/maguowei/gotobeta/internal/modules/todo/application/result"
)

func TestToTodoResponse(t *testing.T) {
	createdAt := time.Date(2026, 5, 29, 10, 11, 12, 0, time.UTC)
	updatedAt := time.Date(2026, 5, 29, 13, 14, 15, 0, time.UTC)

	got := ToTodoResponse(&todoresult.TodoResult{
		ID:        42,
		Title:     "write tests",
		Status:    "pending",
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	})

	if got.TodoID != 42 {
		t.Fatalf("TodoID = %d, want 42", got.TodoID)
	}
	if got.Title != "write tests" {
		t.Fatalf("Title = %q, want write tests", got.Title)
	}
	if got.Status != "pending" {
		t.Fatalf("Status = %q, want pending", got.Status)
	}
	if got.CreatedAt != "2026-05-29 10:11:12" {
		t.Fatalf("CreatedAt = %q, want formatted time", got.CreatedAt)
	}
	if got.UpdatedAt != "2026-05-29 13:14:15" {
		t.Fatalf("UpdatedAt = %q, want formatted time", got.UpdatedAt)
	}
}

func TestToTodoListResponsePreservesOrder(t *testing.T) {
	now := time.Date(2026, 5, 29, 10, 0, 0, 0, time.UTC)
	items := []*todoresult.TodoResult{
		{ID: 1, Title: "first", Status: "pending", CreatedAt: now, UpdatedAt: now},
		{ID: 2, Title: "second", Status: "done", CreatedAt: now, UpdatedAt: now},
	}

	got := ToTodoListResponse(items)
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	if got[0].TodoID != 1 || got[1].TodoID != 2 {
		t.Fatalf("ids = %d/%d, want 1/2", got[0].TodoID, got[1].TodoID)
	}
}
