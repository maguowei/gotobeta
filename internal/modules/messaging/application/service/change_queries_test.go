package service

import (
	"context"
	"testing"

	messagingquery "github.com/maguowei/gotobeta/internal/modules/messaging/application/query"
)

func TestListChangesNonMemberForbidden(t *testing.T) {
	convRepo := newMemConvRepo()
	msgRepo := newMemMsgRepo()
	seedActiveMember(convRepo, 100, 9)
	svc := newMsgService(convRepo, msgRepo, &capturePublisher{})

	if _, err := svc.ListChanges(context.Background(), messagingquery.ListChangesQuery{
		WorkspaceID: 1, OperatorUserID: 999, ConversationID: 100, AfterChangeSeq: 0, Limit: 50,
	}); err == nil {
		t.Fatal("非成员拉取变更应被拒绝")
	}
}

func TestListChangesReturnsAfterSeqAndHasMore(t *testing.T) {
	convRepo := newMemConvRepo()
	msgRepo := newMemMsgRepo()
	pub := &capturePublisher{}
	seedActiveMember(convRepo, 100, 9)
	svc := newMsgService(convRepo, msgRepo, pub)

	// 发 3 条消息 → 变更流应有 3 条 created。
	if _, err := svc.SendMessage(context.Background(), textCmd(100, 9, "c1", "a")); err != nil {
		t.Fatalf("发送失败: %v", err)
	}
	if _, err := svc.SendMessage(context.Background(), textCmd(100, 9, "c2", "b")); err != nil {
		t.Fatalf("发送失败: %v", err)
	}
	if _, err := svc.SendMessage(context.Background(), textCmd(100, 9, "c3", "c")); err != nil {
		t.Fatalf("发送失败: %v", err)
	}

	// limit=2 → 取前 2，hasMore=true。
	page, err := svc.ListChanges(context.Background(), messagingquery.ListChangesQuery{
		WorkspaceID: 1, OperatorUserID: 9, ConversationID: 100, AfterChangeSeq: 0, Limit: 2,
	})
	if err != nil {
		t.Fatalf("拉取失败: %v", err)
	}
	if len(page.Changes) != 2 {
		t.Fatalf("应返回 2 条, got %d", len(page.Changes))
	}
	if !page.HasMore {
		t.Fatal("取满 limit 应 hasMore=true")
	}
	if page.NextCursor != page.Changes[1].ChangeSeq {
		t.Fatalf("nextCursor 应为末条 changeSeq: %d vs %d", page.NextCursor, page.Changes[1].ChangeSeq)
	}

	// 带 nextCursor 续拉 → 剩 1 条，hasMore=false。
	page2, _ := svc.ListChanges(context.Background(), messagingquery.ListChangesQuery{
		WorkspaceID: 1, OperatorUserID: 9, ConversationID: 100, AfterChangeSeq: page.NextCursor, Limit: 2,
	})
	if len(page2.Changes) != 1 || page2.HasMore {
		t.Fatalf("续拉应剩 1 条且 hasMore=false: len=%d hasMore=%v", len(page2.Changes), page2.HasMore)
	}
}
