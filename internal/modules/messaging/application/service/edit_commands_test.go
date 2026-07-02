package service

import (
	"context"
	"log/slog"
	"testing"
	"time"

	messagingcmd "github.com/maguowei/gotobeta/internal/modules/messaging/application/command"
	"github.com/maguowei/gotobeta/internal/modules/messaging/domain/conversation"
	"github.com/maguowei/gotobeta/internal/modules/messaging/domain/messagechange"
	"github.com/maguowei/gotobeta/internal/pkg/imevent"
)

// editCmd 构造编辑命令。
func editCmd(convID, operator, msgID int64, text string) messagingcmd.EditMessageCommand {
	return messagingcmd.EditMessageCommand{
		WorkspaceID: 1, ConversationID: convID, OperatorUserID: operator, MessageID: msgID,
		Content: map[string]any{"text": text},
	}
}

func TestEditMessageSelfSuccess(t *testing.T) {
	convRepo := newMemConvRepo()
	msgRepo := newMemMsgRepo()
	pub := &capturePublisher{}
	seedActiveMember(convRepo, 100, 9)
	svc := newMsgService(convRepo, msgRepo, pub)

	sent, _ := svc.SendMessage(context.Background(), textCmd(100, 9, "c1", "old"))
	out, err := svc.EditMessage(context.Background(), editCmd(100, 9, sent.MessageID, "new"))
	if err != nil {
		t.Fatalf("本人编辑应成功: %v", err)
	}
	if out.Content["text"] != "new" {
		t.Fatalf("返回内容应为 new, got %v", out.Content["text"])
	}
	if out.EditedAt == nil {
		t.Fatal("返回结果应带 editedAt")
	}
	stored, _ := msgRepo.FindByID(context.Background(), sent.MessageID)
	if stored.Content()["text"] != "new" || stored.EditedAt() == nil {
		t.Fatalf("存储应已更新内容与 editedAt: %+v", stored)
	}
	// 发送 + 编辑各 1 事件；编辑事件类型为 MessageEdited。
	if len(pub.events) != 2 {
		t.Fatalf("应共 2 个事件, got %d", len(pub.events))
	}
	if _, ok := pub.events[1].(imevent.MessageEditedEvent); !ok {
		t.Fatalf("第二个事件应为 MessageEditedEvent, got %T", pub.events[1])
	}
}

func TestEditMessageNonSelfForbidden(t *testing.T) {
	convRepo := newMemConvRepo()
	msgRepo := newMemMsgRepo()
	pub := &capturePublisher{}
	seedActiveMember(convRepo, 100, 9)
	// 另加一个活跃成员 8。
	_ = convRepo.AddMember(context.Background(), conversation.NewMember(8000, 100, conversation.MemberUser, 8, conversation.RoleMember))
	svc := newMsgService(convRepo, msgRepo, pub)

	sent, _ := svc.SendMessage(context.Background(), textCmd(100, 9, "c1", "old"))
	if _, err := svc.EditMessage(context.Background(), editCmd(100, 8, sent.MessageID, "new")); err == nil {
		t.Fatal("非本人编辑应被拒绝")
	}
	// 不应发出编辑事件（仍只有发送事件）。
	if len(pub.events) != 1 {
		t.Fatalf("非本人编辑不应发事件, got %d", len(pub.events))
	}
}

func TestEditMessageNonMemberForbidden(t *testing.T) {
	convRepo := newMemConvRepo()
	msgRepo := newMemMsgRepo()
	seedActiveMember(convRepo, 100, 9)
	svc := newMsgService(convRepo, msgRepo, &capturePublisher{})

	sent, _ := svc.SendMessage(context.Background(), textCmd(100, 9, "c1", "old"))
	if _, err := svc.EditMessage(context.Background(), editCmd(100, 999, sent.MessageID, "new")); err == nil {
		t.Fatal("非成员编辑应被拒绝")
	}
}

func TestEditMessageNotFound(t *testing.T) {
	convRepo := newMemConvRepo()
	msgRepo := newMemMsgRepo()
	seedActiveMember(convRepo, 100, 9)
	svc := newMsgService(convRepo, msgRepo, &capturePublisher{})

	if _, err := svc.EditMessage(context.Background(), editCmd(100, 9, 999999, "new")); err == nil {
		t.Fatal("编辑不存在消息应被拒绝")
	}
}

func TestEditMessageCrossConversation(t *testing.T) {
	convRepo := newMemConvRepo()
	msgRepo := newMemMsgRepo()
	seedActiveMember(convRepo, 100, 9)
	seedActiveMember(convRepo, 200, 9)
	svc := newMsgService(convRepo, msgRepo, &capturePublisher{})

	sent, _ := svc.SendMessage(context.Background(), textCmd(100, 9, "c1", "old"))
	// 用会话 200 编辑会话 100 的消息 → 不属于该会话。
	if _, err := svc.EditMessage(context.Background(), editCmd(200, 9, sent.MessageID, "new")); err == nil {
		t.Fatal("跨会话编辑应被拒绝")
	}
}

func TestEditMessageExpiredWindow(t *testing.T) {
	convRepo := newMemConvRepo()
	msgRepo := newMemMsgRepo()
	seedActiveMember(convRepo, 100, 9)
	// 编辑窗口复用 recallWindow=0，必定超窗。
	svc := newMsgServiceWithWindow(convRepo, msgRepo, &capturePublisher{}, 0)

	sent, _ := svc.SendMessage(context.Background(), textCmd(100, 9, "c1", "old"))
	if _, err := svc.EditMessage(context.Background(), editCmd(100, 9, sent.MessageID, "new")); err == nil {
		t.Fatal("超窗编辑应被拒绝")
	}
}

func TestEditMessageWritesChange(t *testing.T) {
	convRepo := newMemConvRepo()
	msgRepo := newMemMsgRepo()
	changeRepo := newMemChangeRepo()
	pub := &capturePublisher{}
	seedActiveMember(convRepo, 100, 9)
	svc := NewMessageService(msgRepo, convRepo, newMemReactionRepo(), changeRepo, &memSeqAlloc{}, allowChecker{}, pub, &fakeIDGen{}, directTxRunner{}, 2*time.Minute, 50, slog.Default(), nil)

	sent, _ := svc.SendMessage(context.Background(), textCmd(100, 9, "c1", "old"))
	if _, err := svc.EditMessage(context.Background(), editCmd(100, 9, sent.MessageID, "new")); err != nil {
		t.Fatalf("编辑失败: %v", err)
	}
	// 变更流应含 1 条 created + 1 条 edited。
	var edited *messagechange.Change
	for _, c := range changeRepo.items {
		if c.Type() == messagechange.ChangeEdited {
			edited = c
		}
	}
	if edited == nil {
		t.Fatal("应写入 edited 变更")
	}
	if edited.MessageID() != sent.MessageID || edited.Payload()["content"] == nil {
		t.Fatalf("edited 变更字段错误: %+v", edited)
	}
	// changeSeq 应严格大于发送消息占用的 seq（编辑另占一个后继 seq）。
	if edited.ChangeSeq() <= sent.Seq {
		t.Fatalf("edited changeSeq 应大于发送 seq %d, got %d", sent.Seq, edited.ChangeSeq())
	}
	// actorID 应为编辑操作者。
	if edited.ActorID() != 9 {
		t.Fatalf("edited actorID 应为 9, got %d", edited.ActorID())
	}
}
