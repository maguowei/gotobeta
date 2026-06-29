package authz_test

import (
	"context"
	"testing"

	"github.com/maguowei/gotobeta/internal/pkg/authz"
)

// fakeChecker 验证端口可被实现与注入。
type fakeChecker struct{ allow bool }

func (f fakeChecker) Check(_ context.Context, _ authz.Request) error {
	if f.allow {
		return nil
	}
	return context.Canceled // 占位错误，仅验证非 nil 路径
}

func TestCheckerContract(t *testing.T) {
	var c authz.Checker = fakeChecker{allow: true}
	req := authz.Request{
		WorkspaceID:  1,
		Subject:      authz.Subject{UserID: 42},
		Action:       "message.send",
		ResourceType: "conversation",
		ResourceID:   "100",
	}
	if err := c.Check(context.Background(), req); err != nil {
		t.Fatalf("allow checker should return nil, got %v", err)
	}

	c = fakeChecker{allow: false}
	if err := c.Check(context.Background(), req); err == nil {
		t.Fatal("deny checker should return error")
	}
}
