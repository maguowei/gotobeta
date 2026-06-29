package handler_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	messaginghandler "github.com/maguowei/gotobeta/internal/modules/messaging/adapter/http/handler"
	messagingrouter "github.com/maguowei/gotobeta/internal/modules/messaging/adapter/http/router"
	messagingcmd "github.com/maguowei/gotobeta/internal/modules/messaging/application/command"
	messagingquery "github.com/maguowei/gotobeta/internal/modules/messaging/application/query"
	messagingresult "github.com/maguowei/gotobeta/internal/modules/messaging/application/result"
	"github.com/maguowei/gotobeta/internal/pkg/apperr"
	"github.com/maguowei/gotobeta/internal/pkg/auth"
)

// errUC 是可配置错误的 usecase fake：所有方法统一返回 err。
type errUC struct{ err error }

func (u errUC) CreateConversation(_ context.Context, _ messagingcmd.CreateConversationCommand) (*messagingresult.ConversationResult, error) {
	return nil, u.err
}
func (u errUC) ListConversations(_ context.Context, _ messagingquery.ListConversationsQuery) ([]*messagingresult.ConversationResult, error) {
	return nil, u.err
}
func (u errUC) AddMember(_ context.Context, _ messagingcmd.AddMemberCommand) (*messagingresult.ConversationMemberResult, error) {
	return nil, u.err
}
func (u errUC) RemoveMember(_ context.Context, _ messagingcmd.RemoveMemberCommand) error {
	return u.err
}
func (u errUC) ListMembers(_ context.Context, _ messagingquery.ListMembersQuery) ([]*messagingresult.ConversationMemberResult, error) {
	return nil, u.err
}
func (u errUC) SendMessage(_ context.Context, _ messagingcmd.SendMessageCommand) (*messagingresult.MessageResult, error) {
	return nil, u.err
}
func (u errUC) PullMessages(_ context.Context, _ messagingquery.PullMessagesQuery) ([]*messagingresult.MessageResult, error) {
	return nil, u.err
}
func (u errUC) RecallMessage(_ context.Context, _ messagingcmd.RecallMessageCommand) error {
	return u.err
}
func (u errUC) ReportRead(_ context.Context, _ messagingcmd.ReportReadCommand) error { return u.err }

// newErrRouter 构造一个注入 claims、但 usecase 固定返回 err 的路由。
func newErrRouter(err error) *gin.Engine {
	gin.SetMode(gin.TestMode)
	uc := errUC{err: err}
	convH := messaginghandler.NewConversationHandler(uc)
	msgH := messaginghandler.NewMessageHandler(uc)
	e := gin.New()
	authMW := func(c *gin.Context) {
		c.Request = c.Request.WithContext(auth.WithClaims(c.Request.Context(), &auth.Claims{UserID: 9}))
		c.Next()
	}
	messagingrouter.RegisterRoutes(e.Group("/api/v1"), convH, msgH, authMW)
	return e
}

// newNoAuthRouter 构造一个不注入 claims 的路由，用于验证缺失认证时的 401 分支。
func newNoAuthRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	uc := fakeUC{}
	convH := messaginghandler.NewConversationHandler(uc)
	msgH := messaginghandler.NewMessageHandler(uc)
	e := gin.New()
	passthrough := func(c *gin.Context) { c.Next() }
	messagingrouter.RegisterRoutes(e.Group("/api/v1"), convH, msgH, passthrough)
	return e
}

// sendReq 发送请求并返回状态码。
func sendReq(t *testing.T, e *gin.Engine, method, path, body string) int {
	t.Helper()
	var r *http.Request
	if body != "" {
		r = httptest.NewRequest(method, path, strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
	} else {
		r = httptest.NewRequest(method, path, nil)
	}
	w := httptest.NewRecorder()
	e.ServeHTTP(w, r)
	return w.Code
}

// 覆盖每个 handler 的 usecase 错误分支：domain error 应映射到对应 HTTP status。
func TestMessagingUseCaseErrors(t *testing.T) {
	endpoints := []struct {
		name, method, path, body string
	}{
		{"create", "POST", "/api/v1/workspaces/1/conversations", `{"type":2,"name":"g"}`},
		{"list", "GET", "/api/v1/workspaces/1/conversations", ""},
		{"addMember", "POST", "/api/v1/workspaces/1/conversations/100/members", `{"memberId":"200","role":3}`},
		{"removeMember", "DELETE", "/api/v1/workspaces/1/conversations/100/members/200", ""},
		{"listMembers", "GET", "/api/v1/workspaces/1/conversations/100/members", ""},
		{"send", "POST", "/api/v1/workspaces/1/conversations/100/messages", `{"clientMsgId":"c1","contentType":1,"content":{"text":"hi"}}`},
		{"pull", "GET", "/api/v1/workspaces/1/conversations/100/messages?afterSeq=0&limit=10", ""},
		{"recall", "POST", "/api/v1/workspaces/1/conversations/100/messages/8001/recall", ""},
		{"reportRead", "POST", "/api/v1/workspaces/1/conversations/100/read", `{"readSeq":5}`},
	}
	kinds := []struct {
		name string
		err  error
		want int
	}{
		{"notFound", apperr.NotFound("不存在"), http.StatusNotFound},
		{"forbidden", apperr.Forbidden("无权限"), http.StatusForbidden},
		{"conflict", apperr.Conflict("冲突"), http.StatusUnprocessableEntity},
		{"internal", apperr.Internal("内部错误", nil), http.StatusInternalServerError},
	}
	for _, k := range kinds {
		e := newErrRouter(k.err)
		for _, ep := range endpoints {
			if code := sendReq(t, e, ep.method, ep.path, ep.body); code != k.want {
				t.Errorf("%s/%s 应返回 %d, got %d", k.name, ep.name, k.want, code)
			}
		}
	}
}

// 缺失 claims 时所有 handler 都应在 RequireClaims 处返回 401。
func TestMessagingMissingClaims(t *testing.T) {
	e := newNoAuthRouter()
	endpoints := []struct {
		method, path, body string
	}{
		{"POST", "/api/v1/workspaces/1/conversations", `{"type":2,"name":"g"}`},
		{"GET", "/api/v1/workspaces/1/conversations", ""},
		{"POST", "/api/v1/workspaces/1/conversations/100/members", `{"memberId":"200","role":3}`},
		{"DELETE", "/api/v1/workspaces/1/conversations/100/members/200", ""},
		{"GET", "/api/v1/workspaces/1/conversations/100/members", ""},
		{"POST", "/api/v1/workspaces/1/conversations/100/messages", `{"clientMsgId":"c1","contentType":1,"content":{"text":"hi"}}`},
		{"GET", "/api/v1/workspaces/1/conversations/100/messages?afterSeq=0&limit=10", ""},
		{"POST", "/api/v1/workspaces/1/conversations/100/messages/8001/recall", ""},
		{"POST", "/api/v1/workspaces/1/conversations/100/read", `{"readSeq":5}`},
	}
	for _, ep := range endpoints {
		if code := sendReq(t, e, ep.method, ep.path, ep.body); code != http.StatusUnauthorized {
			t.Errorf("%s %s 缺 claims 应 401, got %d", ep.method, ep.path, code)
		}
	}
}

// 覆盖 message handler 中独有的参数解析错误分支。
func TestMessagingExtraInvalidParams(t *testing.T) {
	e := newRouter()
	cases := []struct {
		name, method, path, body string
	}{
		{"非法 ws ID", "GET", "/api/v1/workspaces/abc/conversations", ""},
		{"非法消息 ID", "POST", "/api/v1/workspaces/1/conversations/100/messages/abc/recall", ""},
		{"非法 afterSeq", "GET", "/api/v1/workspaces/1/conversations/100/messages?afterSeq=-1", ""},
		{"非法 limit", "GET", "/api/v1/workspaces/1/conversations/100/messages?limit=abc", ""},
		{"非法 memberType", "DELETE", "/api/v1/workspaces/1/conversations/100/members/200?memberType=abc", ""},
		{"create body 错误", "POST", "/api/v1/workspaces/1/conversations", `{`},
		{"addMember body 错误", "POST", "/api/v1/workspaces/1/conversations/100/members", `{`},
		{"reportRead body 错误", "POST", "/api/v1/workspaces/1/conversations/100/read", `{`},
	}
	for _, tc := range cases {
		if code := sendReq(t, e, tc.method, tc.path, tc.body); code != http.StatusBadRequest {
			t.Errorf("%s 应 400, got %d", tc.name, code)
		}
	}
}
