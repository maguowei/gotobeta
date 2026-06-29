// Package handler 处理 realtime 模块 HTTP 请求。
package handler

import (
	"context"

	"github.com/gin-gonic/gin"

	realtimeresp "github.com/maguowei/gotobeta/internal/modules/realtime/adapter/http/response"
	httpmiddleware "github.com/maguowei/gotobeta/internal/pkg/httpx/middleware"
	httpresponse "github.com/maguowei/gotobeta/internal/pkg/httpx/response"
)

// TicketUseCase 定义 handler 对 ticket 用例的依赖。
type TicketUseCase interface {
	IssueTicket(ctx context.Context, userID int64) (string, error)
}

// TicketHandler 处理 WS ticket 请求。
type TicketHandler struct {
	usecase TicketUseCase
}

// NewTicketHandler 创建 Handler。
func NewTicketHandler(uc TicketUseCase) *TicketHandler {
	return &TicketHandler{usecase: uc}
}

// IssueTicket 换取 WS 鉴权 ticket。
func (h *TicketHandler) IssueTicket(c *gin.Context) {
	claims, ok := httpmiddleware.RequireClaims(c)
	if !ok {
		return
	}
	token, err := h.usecase.IssueTicket(c.Request.Context(), claims.UserID)
	if err != nil {
		httpresponse.Error(c, err)
		return
	}
	httpresponse.Success(c, realtimeresp.ToTicketResponse(token))
}
