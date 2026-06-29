// Package handler 处理 messaging 模块 HTTP 请求。
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

// ConversationUseCase 定义 handler 对应用用例的依赖。
type ConversationUseCase interface {
	CreateConversation(ctx context.Context, cmd messagingcmd.CreateConversationCommand) (*messagingresult.ConversationResult, error)
	ListConversations(ctx context.Context, q messagingquery.ListConversationsQuery) ([]*messagingresult.ConversationResult, error)
	AddMember(ctx context.Context, cmd messagingcmd.AddMemberCommand) (*messagingresult.ConversationMemberResult, error)
	RemoveMember(ctx context.Context, cmd messagingcmd.RemoveMemberCommand) error
	ListMembers(ctx context.Context, q messagingquery.ListMembersQuery) ([]*messagingresult.ConversationMemberResult, error)
}

// ConversationHandler 处理会话 HTTP 请求。
type ConversationHandler struct {
	usecase ConversationUseCase
}

// NewConversationHandler 创建 Handler。
func NewConversationHandler(uc ConversationUseCase) *ConversationHandler {
	return &ConversationHandler{usecase: uc}
}

// CreateConversation 创建会话/频道。
func (h *ConversationHandler) CreateConversation(c *gin.Context) {
	claims, ok := httpmiddleware.RequireClaims(c)
	if !ok {
		return
	}
	wsID, err := parsePositiveID(c.Param("ws"))
	if err != nil {
		httpresponse.ErrorWithCode(c, httpresponse.CodeInvalidParam, "无效的工作区 ID")
		return
	}
	var req messagingreq.CreateConversationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresponse.ErrorWithCode(c, httpresponse.CodeInvalidParam, "请求参数格式错误")
		return
	}
	out, err := h.usecase.CreateConversation(c.Request.Context(), req.ToCommand(wsID, claims.UserID))
	if err != nil {
		httpresponse.Error(c, err)
		return
	}
	httpresponse.Success(c, messagingresp.ToConversationResponse(out))
}

// ListConversations 列出我的会话。
func (h *ConversationHandler) ListConversations(c *gin.Context) {
	claims, ok := httpmiddleware.RequireClaims(c)
	if !ok {
		return
	}
	wsID, err := parsePositiveID(c.Param("ws"))
	if err != nil {
		httpresponse.ErrorWithCode(c, httpresponse.CodeInvalidParam, "无效的工作区 ID")
		return
	}
	items, err := h.usecase.ListConversations(c.Request.Context(), messagingquery.ListConversationsQuery{
		WorkspaceID: wsID, UserID: claims.UserID,
	})
	if err != nil {
		httpresponse.Error(c, err)
		return
	}
	httpresponse.Success(c, messagingresp.ToConversationListResponse(items))
}

// AddMember 向会话加入成员。
func (h *ConversationHandler) AddMember(c *gin.Context) {
	claims, ok := httpmiddleware.RequireClaims(c)
	if !ok {
		return
	}
	wsID, cid, ok := parseWsConv(c)
	if !ok {
		return
	}
	var req messagingreq.AddMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresponse.ErrorWithCode(c, httpresponse.CodeInvalidParam, "请求参数格式错误")
		return
	}
	out, err := h.usecase.AddMember(c.Request.Context(), req.ToCommand(wsID, claims.UserID, cid))
	if err != nil {
		httpresponse.Error(c, err)
		return
	}
	httpresponse.Success(c, messagingresp.ToConversationMemberResponse(out))
}

// RemoveMember 从会话移除成员。
func (h *ConversationHandler) RemoveMember(c *gin.Context) {
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
		httpresponse.ErrorWithCode(c, httpresponse.CodeInvalidParam, "无效的成员 ID")
		return
	}
	memberType := int8(1)
	if raw := c.Query("memberType"); raw != "" {
		v, err := strconv.ParseInt(raw, 10, 8)
		if err != nil {
			httpresponse.ErrorWithCode(c, httpresponse.CodeInvalidParam, "无效的成员类型")
			return
		}
		memberType = int8(v)
	}
	if err := h.usecase.RemoveMember(c.Request.Context(), messagingcmd.RemoveMemberCommand{
		WorkspaceID: wsID, OperatorUserID: claims.UserID, ConversationID: cid, MemberType: memberType, MemberID: mid,
	}); err != nil {
		httpresponse.Error(c, err)
		return
	}
	httpresponse.Success(c, nil)
}

// ListMembers 列出会话成员。
func (h *ConversationHandler) ListMembers(c *gin.Context) {
	claims, ok := httpmiddleware.RequireClaims(c)
	if !ok {
		return
	}
	wsID, cid, ok := parseWsConv(c)
	if !ok {
		return
	}
	items, err := h.usecase.ListMembers(c.Request.Context(), messagingquery.ListMembersQuery{
		WorkspaceID: wsID, OperatorUserID: claims.UserID, ConversationID: cid,
	})
	if err != nil {
		httpresponse.Error(c, err)
		return
	}
	httpresponse.Success(c, messagingresp.ToConversationMemberListResponse(items))
}

func parsePositiveID(raw string) (int64, error) {
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, err
	}
	if id <= 0 {
		return 0, strconv.ErrSyntax
	}
	return id, nil
}
