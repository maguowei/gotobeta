package attachment

import "testing"

func TestNewValidation(t *testing.T) {
	t.Parallel()
	if _, err := New(1, 1, 9, "k", "", "image/png", 10); err == nil {
		t.Fatal("空文件名应报错")
	}
	if _, err := New(1, 1, 9, "k", "a.png", "", 10); err == nil {
		t.Fatal("空内容类型应报错")
	}
	if _, err := New(1, 1, 9, "k", "a.png", "image/png", -1); err == nil {
		t.Fatal("负大小应报错")
	}
	a, err := New(1, 1, 9, "k", "a.png", "image/png", 10)
	if err != nil {
		t.Fatalf("意外错误: %v", err)
	}
	if a.Status() != StatusPending {
		t.Fatalf("新建应为待提交, got %d", a.Status())
	}
}

func TestCommitIdempotent(t *testing.T) {
	t.Parallel()
	a, _ := New(1, 1, 9, "k", "a.png", "image/png", 10)
	if !a.Commit() {
		t.Fatal("首次提交应成功")
	}
	if a.Status() != StatusCommitted {
		t.Fatalf("提交后应为已提交, got %d", a.Status())
	}
	if a.Commit() {
		t.Fatal("重复提交应返回 false")
	}
}
