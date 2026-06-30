package response

import (
	"testing"
	"time"

	messagingresult "github.com/maguowei/gotobeta/internal/modules/messaging/application/result"
)

func TestConversationResponses(t *testing.T) {
	at := time.Now()
	out := &messagingresult.ConversationResult{ID: 100, WorkspaceID: 1, Type: 2, Name: "g", LastMsgAt: &at, Unread: 2, CreatedAt: at}
	r := ToConversationResponse(out)
	if r.ConversationID != 100 || r.Unread != 2 || r.LastMsgAt == "" {
		t.Fatalf("会话响应映射错误: %+v", r)
	}
	list := ToConversationListResponse([]*messagingresult.ConversationResult{out})
	if len(list) != 1 {
		t.Fatalf("列表长度错误: %d", len(list))
	}
	// LastMsgAt 为 nil 时不格式化。
	r2 := ToConversationResponse(&messagingresult.ConversationResult{ID: 1, CreatedAt: at})
	if r2.LastMsgAt != "" {
		t.Fatalf("无末条消息时 LastMsgAt 应为空: %q", r2.LastMsgAt)
	}
}

func TestMemberAndMessageResponses(t *testing.T) {
	at := time.Now()
	m := ToConversationMemberResponse(&messagingresult.ConversationMemberResult{ConversationID: 100, MemberID: 9, Role: 1, JoinedAt: at})
	if m.MemberID != 9 || m.Role != 1 {
		t.Fatalf("成员响应映射错误: %+v", m)
	}
	ml := ToConversationMemberListResponse([]*messagingresult.ConversationMemberResult{{ConversationID: 100, MemberID: 9, JoinedAt: at}})
	if len(ml) != 1 {
		t.Fatalf("成员列表错误: %d", len(ml))
	}
	msg := ToMessageResponse(&messagingresult.MessageResult{MessageID: 8001, ConversationID: 100, Seq: 1, Content: map[string]any{"text": "hi"}, ServerTime: at})
	if msg.MessageID != 8001 || msg.Seq != 1 {
		t.Fatalf("消息响应映射错误: %+v", msg)
	}
	msl := ToMessageListResponse([]*messagingresult.MessageResult{{MessageID: 8001, ServerTime: at}})
	if len(msl) != 1 {
		t.Fatalf("消息列表错误: %d", len(msl))
	}
}

func TestReactionListResponse(t *testing.T) {
	list := ToReactionListResponse([]*messagingresult.ReactionResult{{MessageID: 8001, UserID: 9, Emoji: "👍"}})
	if len(list) != 1 {
		t.Fatalf("表情回应列表错误: %d", len(list))
	}
	if list[0].MessageID != 8001 || list[0].UserID != 9 || list[0].Emoji != "👍" {
		t.Fatalf("表情回应响应映射错误: %+v", list[0])
	}
}
