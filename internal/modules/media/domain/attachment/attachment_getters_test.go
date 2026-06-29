package attachment

import (
	"testing"
	"time"
)

// TestNewGetters 校验 New 构造后所有 getter 返回构造入参。
func TestNewGetters(t *testing.T) {
	t.Parallel()
	a, err := New(7, 3, 9, "workspace/3/7/a.png", "a.png", "image/png", 100)
	if err != nil {
		t.Fatalf("意外错误: %v", err)
	}
	if a.ID() != 7 {
		t.Errorf("ID() = %d, want 7", a.ID())
	}
	if a.WorkspaceID() != 3 {
		t.Errorf("WorkspaceID() = %d, want 3", a.WorkspaceID())
	}
	if a.UploaderID() != 9 {
		t.Errorf("UploaderID() = %d, want 9", a.UploaderID())
	}
	if a.ObjectKey() != "workspace/3/7/a.png" {
		t.Errorf("ObjectKey() = %q", a.ObjectKey())
	}
	if a.FileName() != "a.png" {
		t.Errorf("FileName() = %q", a.FileName())
	}
	if a.ContentType() != "image/png" {
		t.Errorf("ContentType() = %q", a.ContentType())
	}
	if a.SizeBytes() != 100 {
		t.Errorf("SizeBytes() = %d, want 100", a.SizeBytes())
	}
	if a.Metadata() == nil {
		t.Error("Metadata() 不应为 nil")
	}
	if a.CreatedAt().IsZero() || a.UpdatedAt().IsZero() {
		t.Error("CreatedAt/UpdatedAt 应被初始化")
	}
}

// TestUnmarshalFromDB 校验从 DB 重建聚合（含 metadata 为 nil 时补默认值）。
func TestUnmarshalFromDB(t *testing.T) {
	t.Parallel()
	created := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	updated := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)

	t.Run("携带 metadata", func(t *testing.T) {
		t.Parallel()
		meta := map[string]any{"k": "v"}
		a := UnmarshalFromDB(1, 2, 3, "key", "f.txt", "text/plain", 8, StatusCommitted, meta, created, updated)
		if a.Status() != StatusCommitted {
			t.Errorf("Status() = %d, want Committed", a.Status())
		}
		if a.Metadata()["k"] != "v" {
			t.Errorf("metadata 未保留: %+v", a.Metadata())
		}
		if !a.CreatedAt().Equal(created) || !a.UpdatedAt().Equal(updated) {
			t.Error("时间字段未保留")
		}
	})

	t.Run("nil metadata 补默认空 map", func(t *testing.T) {
		t.Parallel()
		a := UnmarshalFromDB(1, 2, 3, "key", "f.txt", "text/plain", 8, StatusPending, nil, created, updated)
		if a.Metadata() == nil {
			t.Error("nil metadata 应补为空 map")
		}
	})
}
