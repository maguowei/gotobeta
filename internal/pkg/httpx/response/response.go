package response

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/maguowei/gotobeta/internal/pkg/apperr"
)

// APIResponse 是统一响应体。
type APIResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data"`
}

const (
	// CodeSuccess 表示成功。
	CodeSuccess = 0
	// CodeInvalidParam 表示参数错误。
	CodeInvalidParam = 40001
	// CodeUnauthorized 表示未认证。
	CodeUnauthorized = 40101
	// CodeForbidden 表示无权限。
	CodeForbidden = 40301
	// CodeNotFound 表示资源不存在。
	CodeNotFound = 40401
	// CodeConflict 表示业务冲突。
	CodeConflict = 42201
	// CodeInternal 表示内部错误。
	CodeInternal = 50001
	// CodeClientClosedRequest 表示客户端取消请求。
	CodeClientClosedRequest = 49901
	// CodeTimeout 表示请求超时。
	CodeTimeout = 50401
	// StatusClientClosedRequest 是 Nginx 常用的客户端关闭请求状态码。
	StatusClientClosedRequest = 499
)

// Success 返回成功响应。
func Success(c *gin.Context, data any) {
	c.PureJSON(http.StatusOK, APIResponse{
		Code:    CodeSuccess,
		Message: "success",
		Data:    data,
	})
}

// Error 返回错误响应。
func Error(c *gin.Context, err error) {
	code := mapErrorCode(err)
	c.PureJSON(mapHTTPStatus(code), APIResponse{
		Code:    code,
		Message: mapErrorMessage(err),
		Data:    nil,
	})
}

// ErrorWithCode 返回指定错误码响应。
func ErrorWithCode(c *gin.Context, code int, message string) {
	c.PureJSON(mapHTTPStatus(code), APIResponse{
		Code:    code,
		Message: message,
		Data:    nil,
	})
}

func mapErrorCode(err error) int {
	if errors.Is(err, context.Canceled) {
		return CodeClientClosedRequest
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return CodeTimeout
	}

	var domainErr *apperr.DomainError
	if errors.As(err, &domainErr) {
		switch domainErr.Kind {
		case apperr.KindInvalidParam:
			return CodeInvalidParam
		case apperr.KindUnauthorized:
			return CodeUnauthorized
		case apperr.KindForbidden:
			return CodeForbidden
		case apperr.KindNotFound:
			return CodeNotFound
		case apperr.KindConflict:
			return CodeConflict
		default:
			return CodeInternal
		}
	}

	return CodeInternal
}

func mapHTTPStatus(code int) int {
	switch code {
	case CodeSuccess:
		return http.StatusOK
	case CodeInvalidParam:
		return http.StatusBadRequest
	case CodeUnauthorized:
		return http.StatusUnauthorized
	case CodeForbidden:
		return http.StatusForbidden
	case CodeNotFound:
		return http.StatusNotFound
	case CodeConflict:
		return http.StatusUnprocessableEntity
	case CodeClientClosedRequest:
		return StatusClientClosedRequest
	case CodeTimeout:
		return http.StatusGatewayTimeout
	default:
		return http.StatusInternalServerError
	}
}

func mapErrorMessage(err error) string {
	if errors.Is(err, context.Canceled) {
		return "请求已取消"
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return "请求超时"
	}

	var domainErr *apperr.DomainError
	if errors.As(err, &domainErr) {
		return domainErr.Message
	}

	return "服务器内部错误"
}
