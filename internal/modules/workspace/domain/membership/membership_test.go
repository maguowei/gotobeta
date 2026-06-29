package membership_test

import (
	"testing"
	"time"

	"github.com/maguowei/gotobeta/internal/modules/workspace/domain/membership"
)

// TestNew 校验工厂成功路径：默认 StatusActive，时间字段被填充且一致。
func TestNew(t *testing.T) {
	m, err := membership.New(1, 100, 200)
	if err != nil {
		t.Fatalf("New error: %v", err)
	}
	if m.ID() != 1 {
		t.Fatalf("ID = %d, want 1", m.ID())
	}
	if m.WorkspaceID() != 100 {
		t.Fatalf("WorkspaceID = %d, want 100", m.WorkspaceID())
	}
	if m.UserID() != 200 {
		t.Fatalf("UserID = %d, want 200", m.UserID())
	}
	if m.Status() != membership.StatusActive {
		t.Fatalf("Status = %d, want active", m.Status())
	}
	if m.JoinedAt().IsZero() || m.CreatedAt().IsZero() || m.UpdatedAt().IsZero() {
		t.Fatal("time fields should be populated")
	}
	if !m.JoinedAt().Equal(m.CreatedAt()) || !m.CreatedAt().Equal(m.UpdatedAt()) {
		t.Fatal("joinedAt/createdAt/updatedAt should be identical on creation")
	}
}

// TestNewValidatesInputs 表驱动覆盖工厂的非法入参分支。
func TestNewValidatesInputs(t *testing.T) {
	cases := []struct {
		name        string
		workspaceID int64
		userID      int64
		ok          bool
	}{
		{"valid", 1, 1, true},
		{"zero-workspace", 0, 1, false},
		{"negative-workspace", -1, 1, false},
		{"zero-user", 1, 0, false},
		{"negative-user", 1, -1, false},
		{"both-invalid", 0, 0, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := membership.New(9, tc.workspaceID, tc.userID)
			if tc.ok && err != nil {
				t.Fatalf("expected ok, got %v", err)
			}
			if !tc.ok && err == nil {
				t.Fatal("expected error for invalid input")
			}
		})
	}
}

// TestUnmarshalFromDB 校验从 DB 重建：跳过业务校验，原样保留状态与时间。
func TestUnmarshalFromDB(t *testing.T) {
	joined := time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)
	created := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	updated := time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC)

	m := membership.UnmarshalFromDB(5, 100, 200, membership.StatusDisabled, joined, created, updated)
	if m.ID() != 5 {
		t.Fatalf("ID = %d, want 5", m.ID())
	}
	if m.WorkspaceID() != 100 {
		t.Fatalf("WorkspaceID = %d, want 100", m.WorkspaceID())
	}
	if m.UserID() != 200 {
		t.Fatalf("UserID = %d, want 200", m.UserID())
	}
	if m.Status() != membership.StatusDisabled {
		t.Fatalf("Status = %d, want disabled", m.Status())
	}
	if !m.JoinedAt().Equal(joined) {
		t.Fatalf("JoinedAt = %v, want %v", m.JoinedAt(), joined)
	}
	if !m.CreatedAt().Equal(created) {
		t.Fatalf("CreatedAt = %v, want %v", m.CreatedAt(), created)
	}
	if !m.UpdatedAt().Equal(updated) {
		t.Fatalf("UpdatedAt = %v, want %v", m.UpdatedAt(), updated)
	}
}
