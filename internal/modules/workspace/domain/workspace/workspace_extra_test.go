package workspace_test

import (
	"testing"
	"time"

	"github.com/maguowei/gotobeta/internal/modules/workspace/domain/workspace"
)

// TestNewGetters 覆盖工厂成功路径剩余 getter 与默认 settings/时间字段。
func TestNewGetters(t *testing.T) {
	w, err := workspace.New(7, "acme", "Acme Inc", 42)
	if err != nil {
		t.Fatalf("New error: %v", err)
	}
	if w.ID() != 7 {
		t.Fatalf("ID = %d, want 7", w.ID())
	}
	if w.Slug() != "acme" {
		t.Fatalf("Slug = %q, want acme", w.Slug())
	}
	if w.Name() != "Acme Inc" {
		t.Fatalf("Name = %q", w.Name())
	}
	if w.Settings() == nil || len(w.Settings()) != 0 {
		t.Fatalf("Settings = %v, want empty non-nil map", w.Settings())
	}
	if w.CreatedAt().IsZero() || w.UpdatedAt().IsZero() {
		t.Fatal("time fields should be populated")
	}
}

// TestUnmarshalFromDB 校验从 DB 重建：跳过校验，原样保留字段；nil settings 归一为空 map。
func TestUnmarshalFromDB(t *testing.T) {
	created := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	updated := time.Date(2024, 2, 2, 0, 0, 0, 0, time.UTC)
	settings := map[string]any{"theme": "dark"}

	w := workspace.UnmarshalFromDB(3, "Invalid_Slug", "Name", 9, workspace.StatusDisabled, settings, created, updated)
	if w.ID() != 3 {
		t.Fatalf("ID = %d, want 3", w.ID())
	}
	if w.Slug() != "Invalid_Slug" {
		t.Fatalf("Slug = %q, want Invalid_Slug (validation skipped)", w.Slug())
	}
	if w.OwnerUserID() != 9 {
		t.Fatalf("OwnerUserID = %d, want 9", w.OwnerUserID())
	}
	if w.Status() != workspace.StatusDisabled {
		t.Fatalf("Status = %d, want disabled", w.Status())
	}
	if w.Settings()["theme"] != "dark" {
		t.Fatalf("Settings = %v, want theme=dark", w.Settings())
	}
	if !w.CreatedAt().Equal(created) || !w.UpdatedAt().Equal(updated) {
		t.Fatal("time fields should be preserved")
	}
}

// TestUnmarshalFromDBNilSettings 校验 nil settings 被归一为空 map，避免下游 nil 解引用。
func TestUnmarshalFromDBNilSettings(t *testing.T) {
	w := workspace.UnmarshalFromDB(1, "acme", "Acme", 1, workspace.StatusActive, nil, time.Now(), time.Now())
	if w.Settings() == nil {
		t.Fatal("nil settings should be normalized to empty map")
	}
	if len(w.Settings()) != 0 {
		t.Fatalf("Settings = %v, want empty", w.Settings())
	}
}

// TestRename 覆盖改名成功与空名校验失败两个分支。
func TestRename(t *testing.T) {
	w, _ := workspace.New(1, "acme", "Acme", 10)
	before := w.UpdatedAt()
	time.Sleep(time.Millisecond)

	if err := w.Rename("New Name"); err != nil {
		t.Fatalf("Rename error: %v", err)
	}
	if w.Name() != "New Name" {
		t.Fatalf("Name = %q, want New Name", w.Name())
	}
	if !w.UpdatedAt().After(before) {
		t.Fatal("UpdatedAt should advance after rename")
	}

	if err := w.Rename(""); err == nil {
		t.Fatal("empty name should fail")
	}
	if w.Name() != "New Name" {
		t.Fatal("name should be unchanged after failed rename")
	}
}

// TestStatusIsValid 表驱动覆盖状态值合法性判断。
func TestStatusIsValid(t *testing.T) {
	cases := []struct {
		name   string
		status workspace.Status
		valid  bool
	}{
		{"active", workspace.StatusActive, true},
		{"disabled", workspace.StatusDisabled, true},
		{"zero", workspace.Status(0), false},
		{"unknown", workspace.Status(99), false},
		{"negative", workspace.Status(-1), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.status.IsValid(); got != tc.valid {
				t.Fatalf("IsValid() = %v, want %v", got, tc.valid)
			}
		})
	}
}
