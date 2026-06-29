package service

import (
	"context"
	"testing"

	messagingcmd "github.com/maguowei/gotobeta/internal/modules/messaging/application/command"
	messagingquery "github.com/maguowei/gotobeta/internal/modules/messaging/application/query"
	"github.com/maguowei/gotobeta/internal/modules/messaging/domain/conversation"
)

func TestListConversationsWithUnread(t *testing.T) {
	convRepo := newMemConvRepo()
	svc := newConvService(convRepo, allowChecker{})
	created, _ := svc.CreateConversation(context.Background(), messagingcmd.CreateConversationCommand{
		WorkspaceID: 1, OperatorUserID: 100, Type: 2, Name: "g",
	})
	// 推进会话 last_seq，制造未读。
	conv, _ := convRepo.FindByID(context.Background(), created.ID)
	conv.ApplyMessage(5, 999, "d", conv.CreatedAt())
	_ = convRepo.Save(context.Background(), conv)

	items, err := svc.ListConversations(context.Background(), messagingquery.ListConversationsQuery{
		WorkspaceID: 1, UserID: 100,
	})
	if err != nil {
		t.Fatalf("列表失败: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("应有 1 个会话, got %d", len(items))
	}
	if items[0].Unread != 5 {
		t.Fatalf("未读应为 5, got %d", items[0].Unread)
	}
	// 其他工作区过滤。
	other, err := svc.ListConversations(context.Background(), messagingquery.ListConversationsQuery{
		WorkspaceID: 2, UserID: 100,
	})
	if err != nil || len(other) != 0 {
		t.Fatalf("跨工作区应过滤: %v %d", err, len(other))
	}
}

func TestConversationUserIDsAndPeers(t *testing.T) {
	convRepo := newMemConvRepo()
	svc := newConvService(convRepo, allowChecker{})
	// 群里 owner 100 + 成员 200、300。
	created, _ := svc.CreateConversation(context.Background(), messagingcmd.CreateConversationCommand{
		WorkspaceID: 1, OperatorUserID: 100, Type: 2, Name: "g",
	})
	for _, uid := range []int64{200, 300} {
		if _, err := svc.AddMember(context.Background(), messagingcmd.AddMemberCommand{
			WorkspaceID: 1, OperatorUserID: 100, ConversationID: created.ID, MemberType: 1, MemberID: uid, Role: 3,
		}); err != nil {
			t.Fatalf("加人失败: %v", err)
		}
	}
	ids, err := svc.ConversationUserIDs(context.Background(), created.ID)
	if err != nil || len(ids) != 3 {
		t.Fatalf("应有 3 个用户成员: %v %d", err, len(ids))
	}
	peers, err := svc.UserConversationPeers(context.Background(), 100)
	if err != nil || len(peers) != 2 {
		t.Fatalf("100 的同会话伙伴应有 2 个: %v %d", err, len(peers))
	}
}

func TestListMembersRequiresMembership(t *testing.T) {
	convRepo := newMemConvRepo()
	svc := newConvService(convRepo, allowChecker{})
	created, _ := svc.CreateConversation(context.Background(), messagingcmd.CreateConversationCommand{
		WorkspaceID: 1, OperatorUserID: 100, Type: 2, Name: "g",
	})
	if _, err := svc.ListMembers(context.Background(), messagingquery.ListMembersQuery{
		ConversationID: created.ID, OperatorUserID: 999,
	}); err == nil {
		t.Fatal("非成员不应能查成员列表")
	}
	items, err := svc.ListMembers(context.Background(), messagingquery.ListMembersQuery{
		ConversationID: created.ID, OperatorUserID: 100,
	})
	if err != nil || len(items) != 1 {
		t.Fatalf("owner 应能查列表: %v %d", err, len(items))
	}
}

func TestCreateChannel(t *testing.T) {
	convRepo := newMemConvRepo()
	svc := newConvService(convRepo, allowChecker{})
	out, err := svc.CreateConversation(context.Background(), messagingcmd.CreateConversationCommand{
		WorkspaceID: 1, OperatorUserID: 100, Type: 3, Name: "general", Visibility: 1,
	})
	if err != nil {
		t.Fatalf("建频道失败: %v", err)
	}
	if conversation.Type(out.Type) != conversation.TypeChannel {
		t.Fatalf("类型应为频道, got %d", out.Type)
	}
}

func TestAddMemberDMRejected(t *testing.T) {
	convRepo := newMemConvRepo()
	svc := newConvService(convRepo, allowChecker{})
	dm, _ := svc.CreateConversation(context.Background(), messagingcmd.CreateConversationCommand{
		WorkspaceID: 1, OperatorUserID: 10, Type: 1, PeerUserID: 20,
	})
	if _, err := svc.AddMember(context.Background(), messagingcmd.AddMemberCommand{
		WorkspaceID: 1, OperatorUserID: 10, ConversationID: dm.ID, MemberType: 1, MemberID: 30, Role: 3,
	}); err == nil {
		t.Fatal("单聊不应支持加人")
	}
}

func TestRemoveMemberByManager(t *testing.T) {
	convRepo := newMemConvRepo()
	svc := newConvService(convRepo, allowChecker{})
	created, _ := svc.CreateConversation(context.Background(), messagingcmd.CreateConversationCommand{
		WorkspaceID: 1, OperatorUserID: 100, Type: 2, Name: "g",
	})
	if _, err := svc.AddMember(context.Background(), messagingcmd.AddMemberCommand{
		WorkspaceID: 1, OperatorUserID: 100, ConversationID: created.ID, MemberType: 1, MemberID: 200, Role: 3,
	}); err != nil {
		t.Fatalf("加人失败: %v", err)
	}
	// owner 移除成员 200。
	if err := svc.RemoveMember(context.Background(), messagingcmd.RemoveMemberCommand{
		WorkspaceID: 1, OperatorUserID: 100, ConversationID: created.ID, MemberType: 1, MemberID: 200,
	}); err != nil {
		t.Fatalf("owner 移除成员失败: %v", err)
	}
}

func TestPullMessages(t *testing.T) {
	convRepo := newMemConvRepo()
	msgRepo := newMemMsgRepo()
	pub := &capturePublisher{}
	seedActiveMember(convRepo, 100, 9)
	svc := newMsgService(convRepo, msgRepo, pub)
	_, _ = svc.SendMessage(context.Background(), textCmd(100, 9, "c1", "a"))
	_, _ = svc.SendMessage(context.Background(), textCmd(100, 9, "c2", "b"))

	items, err := svc.PullMessages(context.Background(), messagingquery.PullMessagesQuery{
		ConversationID: 100, OperatorUserID: 9, AfterSeq: 0, Limit: 10,
	})
	if err != nil || len(items) != 2 {
		t.Fatalf("应拉到 2 条: %v %d", err, len(items))
	}
	// 非成员拉取被拒。
	if _, err := svc.PullMessages(context.Background(), messagingquery.PullMessagesQuery{
		ConversationID: 100, OperatorUserID: 999, AfterSeq: 0,
	}); err == nil {
		t.Fatal("非成员拉取应被拒绝")
	}
}

func TestSendMessageConversationNotFound(t *testing.T) {
	svc := newMsgService(newMemConvRepo(), newMemMsgRepo(), &capturePublisher{})
	if _, err := svc.SendMessage(context.Background(), textCmd(404, 9, "c1", "hi")); err == nil {
		t.Fatal("会话不存在应报错")
	}
}

func TestSendMessageEmptyClientMsgID(t *testing.T) {
	svc := newMsgService(newMemConvRepo(), newMemMsgRepo(), &capturePublisher{})
	cmd := textCmd(100, 9, "", "hi")
	if _, err := svc.SendMessage(context.Background(), cmd); err == nil {
		t.Fatal("空 client_msg_id 应报错")
	}
}

func TestRecallMessageNotFound(t *testing.T) {
	convRepo := newMemConvRepo()
	seedActiveMember(convRepo, 100, 9)
	svc := newMsgService(convRepo, newMemMsgRepo(), &capturePublisher{})
	if err := svc.RecallMessage(context.Background(), messagingcmd.RecallMessageCommand{
		WorkspaceID: 1, ConversationID: 100, OperatorUserID: 9, MessageID: 404,
	}); err == nil {
		t.Fatal("撤回不存在的消息应报错")
	}
}
