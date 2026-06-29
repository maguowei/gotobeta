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
	"github.com/maguowei/gotobeta/internal/pkg/auth"
)

type fakeUC struct{}

func (fakeUC) CreateConversation(_ context.Context, _ messagingcmd.CreateConversationCommand) (*messagingresult.ConversationResult, error) {
	return &messagingresult.ConversationResult{ID: 100, WorkspaceID: 1, Type: 2, Name: "g"}, nil
}
func (fakeUC) ListConversations(_ context.Context, _ messagingquery.ListConversationsQuery) ([]*messagingresult.ConversationResult, error) {
	return []*messagingresult.ConversationResult{{ID: 100, WorkspaceID: 1, Unread: 2}}, nil
}
func (fakeUC) AddMember(_ context.Context, _ messagingcmd.AddMemberCommand) (*messagingresult.ConversationMemberResult, error) {
	return &messagingresult.ConversationMemberResult{ConversationID: 100, MemberID: 200, Role: 3}, nil
}
func (fakeUC) RemoveMember(_ context.Context, _ messagingcmd.RemoveMemberCommand) error { return nil }
func (fakeUC) ListMembers(_ context.Context, _ messagingquery.ListMembersQuery) ([]*messagingresult.ConversationMemberResult, error) {
	return []*messagingresult.ConversationMemberResult{{ConversationID: 100, MemberID: 9, Role: 1}}, nil
}
func (fakeUC) SendMessage(_ context.Context, _ messagingcmd.SendMessageCommand) (*messagingresult.MessageResult, error) {
	return &messagingresult.MessageResult{MessageID: 8001, ConversationID: 100, Seq: 1, Content: map[string]any{"text": "hi"}}, nil
}
func (fakeUC) PullMessages(_ context.Context, _ messagingquery.PullMessagesQuery) ([]*messagingresult.MessageResult, error) {
	return []*messagingresult.MessageResult{{MessageID: 8001, Seq: 1}}, nil
}
func (fakeUC) RecallMessage(_ context.Context, _ messagingcmd.RecallMessageCommand) error { return nil }
func (fakeUC) ReportRead(_ context.Context, _ messagingcmd.ReportReadCommand) error       { return nil }

func newRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	uc := fakeUC{}
	convH := messaginghandler.NewConversationHandler(uc)
	msgH := messaginghandler.NewMessageHandler(uc)
	e := gin.New()
	authMW := func(c *gin.Context) {
		c.Request = c.Request.WithContext(auth.WithClaims(c.Request.Context(), &auth.Claims{UserID: 9}))
		c.Next()
	}
	messagingrouter.RegisterRoutes(e.Group("/api/v1"), convH, msgH, nil, authMW)
	return e
}

func do(t *testing.T, e *gin.Engine, method, path, body string) int {
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

func TestMessagingRoutes(t *testing.T) {
	e := newRouter()
	cases := []struct {
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
	for _, tc := range cases {
		if code := do(t, e, tc.method, tc.path, tc.body); code != http.StatusOK {
			t.Errorf("%s %s 应返回 200, got %d", tc.method, tc.path, code)
		}
	}
}

func TestMessagingInvalidParams(t *testing.T) {
	e := newRouter()
	// 非法会话 ID。
	if code := do(t, e, "GET", "/api/v1/workspaces/1/conversations/abc/members", ""); code != http.StatusBadRequest {
		t.Errorf("非法会话 ID 应 400, got %d", code)
	}
	// 缺少必填字段。
	if code := do(t, e, "POST", "/api/v1/workspaces/1/conversations/100/messages", `{}`); code != http.StatusBadRequest {
		t.Errorf("缺字段应 400, got %d", code)
	}
}
