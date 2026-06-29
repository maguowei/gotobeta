package handler

import (
	"context"

	"github.com/gin-gonic/gin"

	userreq "github.com/maguowei/gotobeta/internal/modules/user/adapter/http/request"
	userresp "github.com/maguowei/gotobeta/internal/modules/user/adapter/http/response"
	usercmd "github.com/maguowei/gotobeta/internal/modules/user/application/command"
	userquery "github.com/maguowei/gotobeta/internal/modules/user/application/query"
	userresult "github.com/maguowei/gotobeta/internal/modules/user/application/result"
	httpmiddleware "github.com/maguowei/gotobeta/internal/pkg/httpx/middleware"
	httpresponse "github.com/maguowei/gotobeta/internal/pkg/httpx/response"
)

// UserUseCase 定义用户资料 handler 依赖。
type UserUseCase interface {
	CurrentUser(ctx context.Context, query userquery.GetCurrentUserQuery) (*userresult.UserResult, error)
	UpdateProfile(ctx context.Context, cmd usercmd.UpdateProfileCommand) (*userresult.UserResult, error)
	ChangePassword(ctx context.Context, cmd usercmd.ChangePasswordCommand) error
	ListIdentities(ctx context.Context, query userquery.ListIdentitiesQuery) ([]*userresult.IdentityResult, error)
	UnlinkIdentity(ctx context.Context, cmd usercmd.UnlinkIdentityCommand) error
}

// UserHandler 处理用户资料 HTTP 请求。
type UserHandler struct {
	usecase UserUseCase
}

// NewUserHandler 创建 UserHandler。
func NewUserHandler(usecase UserUseCase) *UserHandler {
	return &UserHandler{usecase: usecase}
}

// Me 返回当前用户。
func (h *UserHandler) Me(c *gin.Context) {
	claims, ok := httpmiddleware.RequireClaims(c)
	if !ok {
		return
	}
	out, err := h.usecase.CurrentUser(c.Request.Context(), userquery.GetCurrentUserQuery{UserID: claims.UserID})
	if err != nil {
		httpresponse.Error(c, err)
		return
	}
	httpresponse.Success(c, userresp.ToUserResponse(out))
}

// UpdateMe 更新当前用户资料。
func (h *UserHandler) UpdateMe(c *gin.Context) {
	claims, ok := httpmiddleware.RequireClaims(c)
	if !ok {
		return
	}
	var req userreq.UpdateProfileRequest
	if !bindJSON(c, &req) {
		return
	}
	out, err := h.usecase.UpdateProfile(c.Request.Context(), req.ToCommand(claims.UserID))
	if err != nil {
		httpresponse.Error(c, err)
		return
	}
	httpresponse.Success(c, userresp.ToUserResponse(out))
}

// ChangePassword 修改当前用户密码。
func (h *UserHandler) ChangePassword(c *gin.Context) {
	claims, ok := httpmiddleware.RequireClaims(c)
	if !ok {
		return
	}
	var req userreq.ChangePasswordRequest
	if !bindJSON(c, &req) {
		return
	}
	if err := h.usecase.ChangePassword(c.Request.Context(), req.ToCommand(claims.UserID)); err != nil {
		httpresponse.Error(c, err)
		return
	}
	httpresponse.Success(c, nil)
}

// ListIdentities 列出三方身份。
func (h *UserHandler) ListIdentities(c *gin.Context) {
	claims, ok := httpmiddleware.RequireClaims(c)
	if !ok {
		return
	}
	items, err := h.usecase.ListIdentities(c.Request.Context(), userquery.ListIdentitiesQuery{UserID: claims.UserID})
	if err != nil {
		httpresponse.Error(c, err)
		return
	}
	httpresponse.Success(c, userresp.ToIdentityResponses(items))
}

// UnlinkIdentity 解绑三方身份。
func (h *UserHandler) UnlinkIdentity(c *gin.Context) {
	claims, ok := httpmiddleware.RequireClaims(c)
	if !ok {
		return
	}
	if err := h.usecase.UnlinkIdentity(c.Request.Context(), usercmd.UnlinkIdentityCommand{
		UserID:   claims.UserID,
		Provider: c.Param("provider"),
	}); err != nil {
		httpresponse.Error(c, err)
		return
	}
	httpresponse.Success(c, nil)
}
