package response

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/maguowei/gotobeta/internal/pkg/apperr"
)

func TestErrorMapsDomainErrorKinds(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   int
	}{
		{name: "invalid param", err: apperr.InvalidParam("参数错误"), wantStatus: http.StatusBadRequest, wantCode: CodeInvalidParam},
		{name: "unauthorized", err: apperr.Unauthorized("未认证"), wantStatus: http.StatusUnauthorized, wantCode: CodeUnauthorized},
		{name: "forbidden", err: apperr.Forbidden("无权限"), wantStatus: http.StatusForbidden, wantCode: CodeForbidden},
		{name: "not found", err: apperr.NotFound("不存在"), wantStatus: http.StatusNotFound, wantCode: CodeNotFound},
		{name: "conflict", err: apperr.Conflict("冲突"), wantStatus: http.StatusUnprocessableEntity, wantCode: CodeConflict},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)

			Error(ctx, tt.err)

			if recorder.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", recorder.Code, tt.wantStatus)
			}

			var body APIResponse
			if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
				t.Fatalf("unmarshal response: %v", err)
			}
			if body.Code != tt.wantCode {
				t.Fatalf("code = %d, want %d", body.Code, tt.wantCode)
			}
		})
	}
}

func TestSuccessAndErrorWithCode(t *testing.T) {
	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)

	Success(ctx, gin.H{"id": 1})

	if recorder.Code != http.StatusOK {
		t.Fatalf("success status = %d, want %d", recorder.Code, http.StatusOK)
	}
	var body APIResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal success response: %v", err)
	}
	if body.Code != CodeSuccess || body.Message != "success" {
		t.Fatalf("success body = %#v", body)
	}

	recorder = httptest.NewRecorder()
	ctx, _ = gin.CreateTestContext(recorder)

	ErrorWithCode(ctx, CodeNotFound, "missing")

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("error status = %d, want %d", recorder.Code, http.StatusNotFound)
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal error response: %v", err)
	}
	if body.Code != CodeNotFound || body.Message != "missing" {
		t.Fatalf("error body = %#v", body)
	}
}

func TestErrorMapsContextErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   int
	}{
		{name: "request canceled", err: context.Canceled, wantStatus: StatusClientClosedRequest, wantCode: CodeClientClosedRequest},
		{name: "request timeout", err: context.DeadlineExceeded, wantStatus: http.StatusGatewayTimeout, wantCode: CodeTimeout},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)

			Error(ctx, tt.err)

			if recorder.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", recorder.Code, tt.wantStatus)
			}

			var body APIResponse
			if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
				t.Fatalf("unmarshal response: %v", err)
			}
			if body.Code != tt.wantCode {
				t.Fatalf("code = %d, want %d", body.Code, tt.wantCode)
			}
		})
	}
}

func TestErrorMapsUnknownErrorsToInternal(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name string
		err  error
	}{
		{name: "plain error", err: errors.New("disk failed")},
		{name: "unknown domain kind", err: &apperr.DomainError{Kind: apperr.Kind(99), Message: "unknown"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)

			Error(ctx, tt.err)

			if recorder.Code != http.StatusInternalServerError {
				t.Fatalf("status = %d, want %d", recorder.Code, http.StatusInternalServerError)
			}

			var body APIResponse
			if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
				t.Fatalf("unmarshal response: %v", err)
			}
			if body.Code != CodeInternal {
				t.Fatalf("code = %d, want %d", body.Code, CodeInternal)
			}
		})
	}
}
