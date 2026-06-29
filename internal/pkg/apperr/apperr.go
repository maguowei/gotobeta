package apperr

import (
	"fmt"
	"log/slog"
)

// Kind 表示错误类别。
type Kind int

const (
	KindInvalidParam Kind = iota + 1
	KindNotFound
	KindConflict
	KindUnauthorized
	KindForbidden
	KindInternal
)

// DomainError 是统一业务错误。
type DomainError struct {
	Kind    Kind
	Code    string
	Message string
	Cause   error
}

func (e *DomainError) Error() string { return e.Message }
func (e *DomainError) Unwrap() error { return e.Cause }

// LogAttrs 返回错误自述的结构化字段。
// logger.WithError 调用方负责把这些字段平铺进日志记录。
func (e *DomainError) LogAttrs() []slog.Attr {
	attrs := []slog.Attr{
		slog.String("errKind", e.Kind.String()),
		slog.String("errMsg", e.Message),
	}
	if e.Code != "" {
		attrs = append(attrs, slog.String("errCode", e.Code))
	}
	if e.Cause != nil {
		attrs = append(attrs, slog.String("errCause", e.Cause.Error()))
	}
	return attrs
}

// WithCode 链式设置业务错误码。
func (e *DomainError) WithCode(code string) *DomainError {
	e.Code = code
	return e
}

func (k Kind) String() string {
	switch k {
	case KindInvalidParam:
		return "InvalidParam"
	case KindNotFound:
		return "NotFound"
	case KindConflict:
		return "Conflict"
	case KindUnauthorized:
		return "Unauthorized"
	case KindForbidden:
		return "Forbidden"
	case KindInternal:
		return "Internal"
	default:
		return fmt.Sprintf("Unknown(%d)", int(k))
	}
}

// InvalidParam 创建参数错误。
func InvalidParam(message string) *DomainError {
	return &DomainError{Kind: KindInvalidParam, Message: message}
}

// NotFound 创建不存在错误。
func NotFound(message string) *DomainError {
	return &DomainError{Kind: KindNotFound, Message: message}
}

// Conflict 创建冲突错误。
func Conflict(message string) *DomainError {
	return &DomainError{Kind: KindConflict, Message: message}
}

// Unauthorized 创建未认证错误。
func Unauthorized(message string) *DomainError {
	return &DomainError{Kind: KindUnauthorized, Message: message}
}

// Forbidden 创建无权限错误。
func Forbidden(message string) *DomainError {
	return &DomainError{Kind: KindForbidden, Message: message}
}

// Internal 创建内部错误。
func Internal(message string, cause error) *DomainError {
	return &DomainError{Kind: KindInternal, Message: message, Cause: cause}
}
