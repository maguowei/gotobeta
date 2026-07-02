package apperr

import (
	"context"
	stderrors "errors"
	"testing"
)

func TestDomainError_LogAttrs(t *testing.T) {
	t.Parallel()

	e := Internal("boom", stderrors.New("root"))
	e.Code = "X1"

	attrs := e.LogAttrs()
	if len(attrs) != 4 {
		t.Fatalf("expected 4 attrs, got %d", len(attrs))
	}

	got := map[string]string{}
	for _, a := range attrs {
		got[a.Key] = a.Value.String()
	}
	if got["errKind"] != "Internal" {
		t.Errorf("errKind = %q", got["errKind"])
	}
	if got["errMsg"] != "boom" {
		t.Errorf("errMsg = %q", got["errMsg"])
	}
	if got["errCode"] != "X1" {
		t.Errorf("errCode = %q", got["errCode"])
	}
	if got["errCause"] != "root" {
		t.Errorf("errCause = %q", got["errCause"])
	}
}

func TestDomainError_LogAttrs_NoCauseNoCode(t *testing.T) {
	t.Parallel()

	attrs := Conflict("dup").LogAttrs()
	if len(attrs) != 2 {
		t.Fatalf("expected 2 attrs, got %d", len(attrs))
	}
}

func TestDomainError_WithCode(t *testing.T) {
	t.Parallel()

	e := Internal("boom", nil).WithCode("E42")
	if e.Code != "E42" {
		t.Fatalf("Code = %q", e.Code)
	}
}

func TestDomainErrorErrorAndUnwrap(t *testing.T) {
	t.Parallel()

	cause := stderrors.New("root")
	err := Internal("boom", cause)

	if err.Error() != "boom" {
		t.Fatalf("Error() = %q, want boom", err.Error())
	}
	if !stderrors.Is(err, cause) {
		t.Fatalf("DomainError should unwrap root cause")
	}
}

// TestWrapInternal 验证 context 取消/超时原样透传，其余错误包装为 Internal 并保留 cause。
func TestWrapInternal(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		err  error
	}{
		{name: "canceled", err: context.Canceled},
		{name: "deadline", err: context.DeadlineExceeded},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := WrapInternal("msg", tc.err); !stderrors.Is(got, tc.err) {
				t.Fatalf("WrapInternal() = %v, want passthrough of %v", got, tc.err)
			}
		})
	}

	cause := stderrors.New("infra down")
	wrapped := WrapInternal("查询失败", cause)
	if wrapped == cause { //nolint:errorlint // 有意做指针相等判断而非 errors.Is
		t.Fatalf("WrapInternal() returned raw infra error, want wrapped DomainError")
	}
	if !stderrors.Is(wrapped, cause) {
		t.Fatalf("WrapInternal() lost original cause in error chain")
	}
	if wrapped.Error() != "查询失败" {
		t.Fatalf("WrapInternal() message = %q, want 查询失败", wrapped.Error())
	}
}

func TestKindStringAndConstructors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  *DomainError
		kind Kind
	}{
		{name: "invalid param", err: InvalidParam("bad"), kind: KindInvalidParam},
		{name: "not found", err: NotFound("missing"), kind: KindNotFound},
		{name: "conflict", err: Conflict("dup"), kind: KindConflict},
		{name: "unauthorized", err: Unauthorized("login"), kind: KindUnauthorized},
		{name: "forbidden", err: Forbidden("deny"), kind: KindForbidden},
		{name: "internal", err: Internal("boom", nil), kind: KindInternal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if tt.err.Kind != tt.kind {
				t.Fatalf("Kind = %v, want %v", tt.err.Kind, tt.kind)
			}
			if tt.err.Kind.String() == "" {
				t.Fatalf("Kind.String() is empty")
			}
		})
	}

	if got := Kind(99).String(); got != "Unknown(99)" {
		t.Fatalf("unknown kind = %q", got)
	}
}
