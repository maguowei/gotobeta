package handler

import (
	"context"
	"strconv"

	"github.com/gin-gonic/gin"

	messagingreq "github.com/maguowei/gotobeta/internal/modules/messaging/adapter/http/request"
	messagingresp "github.com/maguowei/gotobeta/internal/modules/messaging/adapter/http/response"
	messagingcmd "github.com/maguowei/gotobeta/internal/modules/messaging/application/command"
	messagingquery "github.com/maguowei/gotobeta/internal/modules/messaging/application/query"
	messagingresult "github.com/maguowei/gotobeta/internal/modules/messaging/application/result"
	httpmiddleware "github.com/maguowei/gotobeta/internal/pkg/httpx/middleware"
	httpresponse "github.com/maguowei/gotobeta/internal/pkg/httpx/response"
)

// MessageUseCase 定义 handler 对消息用例的依赖。
type MessageUseCase interface {
	SendMessage(ctx context.Context, cmd messagingcmd.SendMessageCommand) (*messagingresult.MessageResult, error)
	PullMessages(ctx context.Context, q messagingquery.PullMessagesQuery) ([]*messagingresult.MessageResult, error)
	RecallMessage(ctx context.Context, cmd messagingcmd.RecallMessageCommand) error
	ReportRead(ctx context.Context, cmd messagingcmd.ReportReadCommand) error
}

// MessageHandler 处理消息 HTTP 请求。
type MessageHandler struct {
	usecase MessageUseCase
}

// NewMessageHandler 创建 Handler。
func NewMessageHandler(uc MessageUseCase) *MessageHandler {
	return &MessageHandler{usecase: uc}
}

// SendMessage 发送消息。
func (h *MessageHandler) SendMessage(c *gin.Context) {
	claims, ok := httpmiddleware.RequireClaims(c)
	if !ok {
		return
	}
	wsID, cid, ok := parseWsConv(c)
	if !ok {
		return
	}
	var req messagingreq.SendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresponse.ErrorWithCode(c, httpresponse.CodeInvalidParam, "请求参数格式错误")
		return
	}
	out, err := h.usecase.SendMessage(c.Request.Context(), req.ToCommand(wsID, cid, claims.UserID))
	if err != nil {
		httpresponse.Error(c, err)
		return
	}
	httpresponse.Success(c, messagingresp.ToMessageResponse(out))
}

// PullMessages 增量拉取消息。
func (h *MessageHandler) PullMessages(c *gin.Context) {
	claims, ok := httpmiddleware.RequireClaims(c)
	if !ok {
		return
	}
	wsID, cid, ok := parseWsConv(c)
	if !ok {
		return
	}
	afterSeq, err := parseNonNegativeQuery(c.Query("afterSeq"))
	if err != nil {
		httpresponse.ErrorWithCode(c, httpresponse.CodeInvalidParam, "无效的 afterSeq")
		return
	}
	limit, err := parseNonNegativeQuery(c.Query("limit"))
	if err != nil {
		httpresponse.ErrorWithCode(c, httpresponse.CodeInvalidParam, "无效的 limit")
		return
	}
	items, err := h.usecase.PullMessages(c.Request.Context(), messagingquery.PullMessagesQuery{
		WorkspaceID: wsID, OperatorUserID: claims.UserID, ConversationID: cid,
		AfterSeq: afterSeq, Limit: int(limit),
	})
	if err != nil {
		httpresponse.Error(c, err)
		return
	}
	httpresponse.Success(c, messagingresp.ToMessageListResponse(items))
}

// RecallMessage 撤回消息。
func (h *MessageHandler) RecallMessage(c *gin.Context) {
	claims, ok := httpmiddleware.RequireClaims(c)
	if !ok {
		return
	}
	wsID, cid, ok := parseWsConv(c)
	if !ok {
		return
	}
	mid, err := parsePositiveID(c.Param("mid"))
	if err != nil {
		httpresponse.ErrorWithCode(c, httpresponse.CodeInvalidParam, "无效的消息 ID")
		return
	}
	if err := h.usecase.RecallMessage(c.Request.Context(), messagingcmd.RecallMessageCommand{
		WorkspaceID: wsID, ConversationID: cid, OperatorUserID: claims.UserID, MessageID: mid,
	}); err != nil {
		httpresponse.Error(c, err)
		return
	}
	httpresponse.Success(c, nil)
}

// ReportRead 上报已读水位。
func (h *MessageHandler) ReportRead(c *gin.Context) {
	claims, ok := httpmiddleware.RequireClaims(c)
	if !ok {
		return
	}
	_, cid, ok := parseWsConv(c)
	if !ok {
		return
	}
	var req messagingreq.ReportReadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresponse.ErrorWithCode(c, httpresponse.CodeInvalidParam, "请求参数格式错误")
		return
	}
	if err := h.usecase.ReportRead(c.Request.Context(), req.ToCommand(cid, claims.UserID)); err != nil {
		httpresponse.Error(c, err)
		return
	}
	httpresponse.Success(c, nil)
}

// parseWsConv 解析路径中的 ws 与 cid，失败时已写入响应。
func parseWsConv(c *gin.Context) (wsID, cid int64, ok bool) {
	wsID, err := parsePositiveID(c.Param("ws"))
	if err != nil {
		httpresponse.ErrorWithCode(c, httpresponse.CodeInvalidParam, "无效的工作区 ID")
		return 0, 0, false
	}
	cid, err = parsePositiveID(c.Param("cid"))
	if err != nil {
		httpresponse.ErrorWithCode(c, httpresponse.CodeInvalidParam, "无效的会话 ID")
		return 0, 0, false
	}
	return wsID, cid, true
}

func parseNonNegativeQuery(raw string) (int64, error) {
	if raw == "" {
		return 0, nil
	}
	v, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || v < 0 {
		return 0, strconv.ErrSyntax
	}
	return v, nil
}
