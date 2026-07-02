// Package handler 处理 workspace 模块 HTTP 请求。
package handler

import (
	"context"

	"github.com/gin-gonic/gin"

	workspacereq "github.com/maguowei/gotobeta/internal/modules/workspace/adapter/http/request"
	workspaceresp "github.com/maguowei/gotobeta/internal/modules/workspace/adapter/http/response"
	workspacecmd "github.com/maguowei/gotobeta/internal/modules/workspace/application/command"
	workspacequery "github.com/maguowei/gotobeta/internal/modules/workspace/application/query"
	workspaceresult "github.com/maguowei/gotobeta/internal/modules/workspace/application/result"
	"github.com/maguowei/gotobeta/internal/pkg/httpx"
	httpmiddleware "github.com/maguowei/gotobeta/internal/pkg/httpx/middleware"
	httpresponse "github.com/maguowei/gotobeta/internal/pkg/httpx/response"
)

// WorkspaceUseCase 定义 handler 对应用用例的依赖。
type WorkspaceUseCase interface {
	CreateWorkspace(ctx context.Context, cmd workspacecmd.CreateWorkspaceCommand) (*workspaceresult.WorkspaceResult, error)
	ListMyWorkspaces(ctx context.Context, q workspacequery.ListMyWorkspacesQuery) ([]*workspaceresult.WorkspaceResult, error)
	InviteMember(ctx context.Context, cmd workspacecmd.InviteMemberCommand) (*workspaceresult.MemberResult, error)
	AssignRole(ctx context.Context, cmd workspacecmd.AssignRoleCommand) error
	ListRoles(ctx context.Context, q workspacequery.ListRolesQuery) ([]*workspaceresult.RoleResult, error)
}

// WorkspaceHandler 处理工作区 HTTP 请求。
type WorkspaceHandler struct {
	usecase WorkspaceUseCase
}

// NewWorkspaceHandler 创建 Handler。
func NewWorkspaceHandler(uc WorkspaceUseCase) *WorkspaceHandler {
	return &WorkspaceHandler{usecase: uc}
}

// CreateWorkspace 创建工作区。
func (h *WorkspaceHandler) CreateWorkspace(c *gin.Context) {
	claims, ok := httpmiddleware.RequireClaims(c)
	if !ok {
		return
	}
	var req workspacereq.CreateWorkspaceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresponse.ErrorWithCode(c, httpresponse.CodeInvalidParam, "请求参数格式错误")
		return
	}
	out, err := h.usecase.CreateWorkspace(c.Request.Context(), req.ToCommand(claims.UserID))
	if err != nil {
		httpresponse.Error(c, err)
		return
	}
	httpresponse.Success(c, workspaceresp.ToWorkspaceResponse(out))
}

// ListMyWorkspaces 列出我的工作区。
func (h *WorkspaceHandler) ListMyWorkspaces(c *gin.Context) {
	claims, ok := httpmiddleware.RequireClaims(c)
	if !ok {
		return
	}
	items, err := h.usecase.ListMyWorkspaces(c.Request.Context(), workspacequery.ListMyWorkspacesQuery{UserID: claims.UserID})
	if err != nil {
		httpresponse.Error(c, err)
		return
	}
	httpresponse.Success(c, workspaceresp.ToWorkspaceListResponse(items))
}

// InviteMember 邀请成员。
func (h *WorkspaceHandler) InviteMember(c *gin.Context) {
	claims, ok := httpmiddleware.RequireClaims(c)
	if !ok {
		return
	}
	wsID, err := httpx.ParsePositiveID(c.Param("ws"))
	if err != nil {
		httpresponse.ErrorWithCode(c, httpresponse.CodeInvalidParam, "无效的工作区 ID")
		return
	}
	var req workspacereq.InviteMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresponse.ErrorWithCode(c, httpresponse.CodeInvalidParam, "请求参数格式错误")
		return
	}
	out, err := h.usecase.InviteMember(c.Request.Context(), req.ToCommand(wsID, claims.UserID))
	if err != nil {
		httpresponse.Error(c, err)
		return
	}
	httpresponse.Success(c, workspaceresp.ToMemberResponse(out))
}

// AssignRole 给成员分配角色。
func (h *WorkspaceHandler) AssignRole(c *gin.Context) {
	claims, ok := httpmiddleware.RequireClaims(c)
	if !ok {
		return
	}
	wsID, err := httpx.ParsePositiveID(c.Param("ws"))
	if err != nil {
		httpresponse.ErrorWithCode(c, httpresponse.CodeInvalidParam, "无效的工作区 ID")
		return
	}
	uid, err := httpx.ParsePositiveID(c.Param("uid"))
	if err != nil {
		httpresponse.ErrorWithCode(c, httpresponse.CodeInvalidParam, "无效的用户 ID")
		return
	}
	var req workspacereq.AssignRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresponse.ErrorWithCode(c, httpresponse.CodeInvalidParam, "请求参数格式错误")
		return
	}
	if err := h.usecase.AssignRole(c.Request.Context(), req.ToCommand(wsID, claims.UserID, uid)); err != nil {
		httpresponse.Error(c, err)
		return
	}
	httpresponse.Success(c, nil)
}

// ListRoles 列出工作区角色。
func (h *WorkspaceHandler) ListRoles(c *gin.Context) {
	claims, ok := httpmiddleware.RequireClaims(c)
	if !ok {
		return
	}
	wsID, err := httpx.ParsePositiveID(c.Param("ws"))
	if err != nil {
		httpresponse.ErrorWithCode(c, httpresponse.CodeInvalidParam, "无效的工作区 ID")
		return
	}
	items, err := h.usecase.ListRoles(c.Request.Context(), workspacequery.ListRolesQuery{WorkspaceID: wsID, OperatorUserID: claims.UserID})
	if err != nil {
		httpresponse.Error(c, err)
		return
	}
	httpresponse.Success(c, workspaceresp.ToRoleListResponse(items))
}
