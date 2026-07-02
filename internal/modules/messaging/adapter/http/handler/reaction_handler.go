package handler

import (
	"context"

	"github.com/gin-gonic/gin"

	messagingreq "github.com/maguowei/gotobeta/internal/modules/messaging/adapter/http/request"
	messagingresp "github.com/maguowei/gotobeta/internal/modules/messaging/adapter/http/response"
	messagingcmd "github.com/maguowei/gotobeta/internal/modules/messaging/application/command"
	messagingquery "github.com/maguowei/gotobeta/internal/modules/messaging/application/query"
	messagingresult "github.com/maguowei/gotobeta/internal/modules/messaging/application/result"
	"github.com/maguowei/gotobeta/internal/pkg/httpx"
	httpmiddleware "github.com/maguowei/gotobeta/internal/pkg/httpx/middleware"
	httpresponse "github.com/maguowei/gotobeta/internal/pkg/httpx/response"
)

// ReactionUseCase 定义 handler 对表情回应用例的依赖。
type ReactionUseCase interface {
	AddReaction(ctx context.Context, cmd messagingcmd.AddReactionCommand) error
	RemoveReaction(ctx context.Context, cmd messagingcmd.RemoveReactionCommand) error
	ListReactions(ctx context.Context, q messagingquery.ListReactionsQuery) ([]*messagingresult.ReactionResult, error)
}

// ReactionHandler 处理表情回应 HTTP 请求。
type ReactionHandler struct {
	usecase ReactionUseCase
}

// NewReactionHandler 创建 Handler。
func NewReactionHandler(uc ReactionUseCase) *ReactionHandler {
	return &ReactionHandler{usecase: uc}
}

// AddReaction 添加表情回应。
func (h *ReactionHandler) AddReaction(c *gin.Context) {
	claims, ok := httpmiddleware.RequireClaims(c)
	if !ok {
		return
	}
	wsID, cid, mid, ok := parseWsConvMsg(c)
	if !ok {
		return
	}
	var req messagingreq.AddReactionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresponse.ErrorWithCode(c, httpresponse.CodeInvalidParam, "请求参数格式错误")
		return
	}
	if err := h.usecase.AddReaction(c.Request.Context(), req.ToCommand(wsID, cid, mid, claims.UserID)); err != nil {
		httpresponse.Error(c, err)
		return
	}
	httpresponse.Success(c, nil)
}

// RemoveReaction 取消表情回应（emoji 经 query 传入，避免 DELETE 携带 body）。
func (h *ReactionHandler) RemoveReaction(c *gin.Context) {
	claims, ok := httpmiddleware.RequireClaims(c)
	if !ok {
		return
	}
	wsID, cid, mid, ok := parseWsConvMsg(c)
	if !ok {
		return
	}
	emoji := c.Query("emoji")
	if emoji == "" {
		httpresponse.ErrorWithCode(c, httpresponse.CodeInvalidParam, "emoji 不能为空")
		return
	}
	if err := h.usecase.RemoveReaction(c.Request.Context(), messagingcmd.RemoveReactionCommand{
		WorkspaceID: wsID, ConversationID: cid, MessageID: mid, OperatorUserID: claims.UserID, Emoji: emoji,
	}); err != nil {
		httpresponse.Error(c, err)
		return
	}
	httpresponse.Success(c, nil)
}

// ListReactions 列举消息的表情回应。
func (h *ReactionHandler) ListReactions(c *gin.Context) {
	claims, ok := httpmiddleware.RequireClaims(c)
	if !ok {
		return
	}
	wsID, cid, mid, ok := parseWsConvMsg(c)
	if !ok {
		return
	}
	out, err := h.usecase.ListReactions(c.Request.Context(), messagingquery.ListReactionsQuery{
		WorkspaceID: wsID, ConversationID: cid, MessageID: mid, OperatorUserID: claims.UserID,
	})
	if err != nil {
		httpresponse.Error(c, err)
		return
	}
	httpresponse.Success(c, messagingresp.ToReactionListResponse(out))
}

// parseWsConvMsg 解析路径中的 ws、cid 与 mid，失败时已写入响应。
func parseWsConvMsg(c *gin.Context) (wsID, cid, mid int64, ok bool) {
	wsID, cid, ok = parseWsConv(c)
	if !ok {
		return 0, 0, 0, false
	}
	mid, err := httpx.ParsePositiveID(c.Param("mid"))
	if err != nil {
		httpresponse.ErrorWithCode(c, httpresponse.CodeInvalidParam, "无效的消息 ID")
		return 0, 0, 0, false
	}
	return wsID, cid, mid, true
}
