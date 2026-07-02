// Package handler 处理 media 模块 HTTP 请求。
package handler

import (
	"context"

	"github.com/gin-gonic/gin"

	mediareq "github.com/maguowei/gotobeta/internal/modules/media/adapter/http/request"
	mediaresp "github.com/maguowei/gotobeta/internal/modules/media/adapter/http/response"
	mediacmd "github.com/maguowei/gotobeta/internal/modules/media/application/command"
	mediaresult "github.com/maguowei/gotobeta/internal/modules/media/application/result"
	"github.com/maguowei/gotobeta/internal/pkg/httpx"
	httpmiddleware "github.com/maguowei/gotobeta/internal/pkg/httpx/middleware"
	httpresponse "github.com/maguowei/gotobeta/internal/pkg/httpx/response"
)

// AttachmentUseCase 定义 handler 对附件用例的依赖。
type AttachmentUseCase interface {
	Presign(ctx context.Context, cmd mediacmd.PresignAttachmentCommand) (*mediaresult.PresignResult, error)
	Commit(ctx context.Context, cmd mediacmd.CommitAttachmentCommand) (*mediaresult.AttachmentResult, error)
}

// AttachmentHandler 处理附件 HTTP 请求。
type AttachmentHandler struct {
	usecase AttachmentUseCase
}

// NewAttachmentHandler 创建 Handler。
func NewAttachmentHandler(uc AttachmentUseCase) *AttachmentHandler {
	return &AttachmentHandler{usecase: uc}
}

// Presign 申请预签名上传。
func (h *AttachmentHandler) Presign(c *gin.Context) {
	claims, ok := httpmiddleware.RequireClaims(c)
	if !ok {
		return
	}
	var req mediareq.PresignAttachmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresponse.ErrorWithCode(c, httpresponse.CodeInvalidParam, "请求参数格式错误")
		return
	}
	out, err := h.usecase.Presign(c.Request.Context(), req.ToCommand(claims.UserID))
	if err != nil {
		httpresponse.Error(c, err)
		return
	}
	httpresponse.Success(c, mediaresp.ToPresignResponse(out))
}

// Commit 确认附件上传完成。
func (h *AttachmentHandler) Commit(c *gin.Context) {
	claims, ok := httpmiddleware.RequireClaims(c)
	if !ok {
		return
	}
	id, err := httpx.ParsePositiveID(c.Param("id"))
	if err != nil {
		httpresponse.ErrorWithCode(c, httpresponse.CodeInvalidParam, "无效的附件 ID")
		return
	}
	var req mediareq.CommitAttachmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresponse.ErrorWithCode(c, httpresponse.CodeInvalidParam, "请求参数格式错误")
		return
	}
	out, err := h.usecase.Commit(c.Request.Context(), req.ToCommand(claims.UserID, id))
	if err != nil {
		httpresponse.Error(c, err)
		return
	}
	httpresponse.Success(c, mediaresp.ToAttachmentResponse(out))
}
