package acl_test

import (
	"testing"
	"time"

	"github.com/maguowei/gotobeta/internal/modules/workspace/domain/acl"
)

// TestNewEntryGetters 覆盖工厂成功路径与全部 getter，默认 sourceType=1。
func TestNewEntryGetters(t *testing.T) {
	exp := time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)
	e, err := acl.NewEntry(42, 7, acl.SubjectRole, 99, "channel", "c-1", "channel.archive", acl.EffectDeny, "违规", 5, &exp)
	if err != nil {
		t.Fatalf("NewEntry error: %v", err)
	}
	if e.ID() != 42 {
		t.Fatalf("ID = %d, want 42", e.ID())
	}
	if e.WorkspaceID() != 7 {
		t.Fatalf("WorkspaceID = %d, want 7", e.WorkspaceID())
	}
	if e.SubjectType() != acl.SubjectRole {
		t.Fatalf("SubjectType = %d, want role", e.SubjectType())
	}
	if e.SubjectID() != 99 {
		t.Fatalf("SubjectID = %d, want 99", e.SubjectID())
	}
	if e.ResourceType() != "channel" {
		t.Fatalf("ResourceType = %q", e.ResourceType())
	}
	if e.ResourceID() != "c-1" {
		t.Fatalf("ResourceID = %q", e.ResourceID())
	}
	if e.ActionCode() != "channel.archive" {
		t.Fatalf("ActionCode = %q", e.ActionCode())
	}
	if e.Effect() != acl.EffectDeny {
		t.Fatalf("Effect = %d, want deny", e.Effect())
	}
	if e.Reason() != "违规" {
		t.Fatalf("Reason = %q", e.Reason())
	}
	if e.SourceType() != 1 {
		t.Fatalf("SourceType = %d, want 1", e.SourceType())
	}
	if e.ExpiresAt() == nil || !e.ExpiresAt().Equal(exp) {
		t.Fatalf("ExpiresAt = %v, want %v", e.ExpiresAt(), exp)
	}
	if e.CreatedBy() != 5 {
		t.Fatalf("CreatedBy = %d, want 5", e.CreatedBy())
	}
}

// TestNewEntryInvalidEffect 校验非法 effect 被拒绝。
func TestNewEntryInvalidEffect(t *testing.T) {
	cases := []struct {
		name   string
		effect acl.Effect
		ok     bool
	}{
		{"allow", acl.EffectAllow, true},
		{"deny", acl.EffectDeny, true},
		{"zero", acl.Effect(0), false},
		{"unknown", acl.Effect(3), false},
		{"negative", acl.Effect(-1), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := acl.NewEntry(1, 1, acl.SubjectUser, 1, "r", "1", "a", tc.effect, "原因", 1, nil)
			if tc.ok && err != nil {
				t.Fatalf("expected ok, got %v", err)
			}
			if !tc.ok && err == nil {
				t.Fatal("expected error for invalid effect")
			}
		})
	}
}

// TestUnmarshalEntry 校验从 DB 重建：原样保留所有字段，含自定义 sourceType。
func TestUnmarshalEntry(t *testing.T) {
	exp := time.Date(2031, 6, 1, 0, 0, 0, 0, time.UTC)
	created := time.Date(2024, 3, 3, 3, 3, 3, 0, time.UTC)
	e := acl.UnmarshalEntry(8, 2, acl.SubjectUser, 50, "conversation", "100", "message.recall",
		acl.EffectAllow, "特批", 3, &exp, 6, created)

	if e.ID() != 8 {
		t.Fatalf("ID = %d, want 8", e.ID())
	}
	if e.WorkspaceID() != 2 {
		t.Fatalf("WorkspaceID = %d, want 2", e.WorkspaceID())
	}
	if e.SubjectType() != acl.SubjectUser {
		t.Fatalf("SubjectType = %d, want user", e.SubjectType())
	}
	if e.SubjectID() != 50 {
		t.Fatalf("SubjectID = %d, want 50", e.SubjectID())
	}
	if e.ResourceType() != "conversation" {
		t.Fatalf("ResourceType = %q", e.ResourceType())
	}
	if e.ResourceID() != "100" {
		t.Fatalf("ResourceID = %q", e.ResourceID())
	}
	if e.ActionCode() != "message.recall" {
		t.Fatalf("ActionCode = %q", e.ActionCode())
	}
	if e.Effect() != acl.EffectAllow {
		t.Fatalf("Effect = %d, want allow", e.Effect())
	}
	if e.Reason() != "特批" {
		t.Fatalf("Reason = %q", e.Reason())
	}
	if e.SourceType() != 3 {
		t.Fatalf("SourceType = %d, want 3", e.SourceType())
	}
	if e.ExpiresAt() == nil || !e.ExpiresAt().Equal(exp) {
		t.Fatalf("ExpiresAt = %v, want %v", e.ExpiresAt(), exp)
	}
	if e.CreatedBy() != 6 {
		t.Fatalf("CreatedBy = %d, want 6", e.CreatedBy())
	}
}

// TestIsActiveNoExpiry 无过期时间的条目永久有效。
func TestIsActiveNoExpiry(t *testing.T) {
	e, _ := acl.NewEntry(1, 1, acl.SubjectUser, 1, "r", "1", "a", acl.EffectDeny, "永久", 1, nil)
	if !e.IsActive(time.Now()) {
		t.Fatal("entry without expiry should always be active")
	}
}
