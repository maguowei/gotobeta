package request

import "testing"

func TestConversationRequestToCommand(t *testing.T) {
	c := CreateConversationRequest{Type: 1, PeerUserID: 20, Name: "g", Topic: "t", Visibility: 2}.ToCommand(1, 9)
	if c.WorkspaceID != 1 || c.OperatorUserID != 9 || c.Type != 1 || c.PeerUserID != 20 {
		t.Fatalf("CreateConversation 映射错误: %+v", c)
	}
	a := AddMemberRequest{MemberID: 200, Role: 3}.ToCommand(1, 9, 100)
	if a.MemberType != 1 || a.MemberID != 200 || a.ConversationID != 100 {
		t.Fatalf("AddMember 缺省 memberType 应为 1: %+v", a)
	}
	a2 := AddMemberRequest{MemberType: 2, MemberID: 5}.ToCommand(1, 9, 100)
	if a2.MemberType != 2 {
		t.Fatalf("AddMember memberType 应保留: %+v", a2)
	}
}

func TestMessageRequestToCommand(t *testing.T) {
	s := SendMessageRequest{ClientMsgID: "c1", ContentType: 1, Content: map[string]any{"text": "hi"}, ReplyToMsgID: 7}.ToCommand(1, 100, 9)
	if s.ClientMsgID != "c1" || s.ConversationID != 100 || s.SenderUserID != 9 || s.ReplyToMsgID != 7 {
		t.Fatalf("SendMessage 映射错误: %+v", s)
	}
	r := ReportReadRequest{ReadSeq: 5}.ToCommand(100, 9)
	if r.ConversationID != 100 || r.UserID != 9 || r.ReadSeq != 5 {
		t.Fatalf("ReportRead 映射错误: %+v", r)
	}
}

func TestAddReactionRequestToCommand(t *testing.T) {
	c := AddReactionRequest{Emoji: "👍"}.ToCommand(1, 100, 8001, 9)
	if c.WorkspaceID != 1 || c.ConversationID != 100 || c.MessageID != 8001 || c.OperatorUserID != 9 || c.Emoji != "👍" {
		t.Fatalf("AddReaction 映射错误: %+v", c)
	}
}
