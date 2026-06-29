package service

import (
	"context"
	"log/slog"
	"testing"
	"time"

	messagingcmd "github.com/maguowei/gotobeta/internal/modules/messaging/application/command"
	"github.com/maguowei/gotobeta/internal/modules/messaging/domain/conversation"
	"github.com/maguowei/gotobeta/internal/modules/messaging/domain/message"
	"github.com/maguowei/gotobeta/internal/pkg/event"
)

// memMsgRepo 是内存版消息仓储。
type memMsgRepo struct {
	byID     map[int64]*message.Message
	byClient map[string]*message.Message
	bySeq    []*message.Message
}

func newMemMsgRepo() *memMsgRepo {
	return &memMsgRepo{byID: map[int64]*message.Message{}, byClient: map[string]*message.Message{}}
}

func clientKey(convID int64, cid string) string { return string(rune(convID)) + "|" + cid }

func (r *memMsgRepo) Create(_ context.Context, m *message.Message) error {
	r.byID[m.ID()] = m
	if m.ClientMsgID() != nil {
		r.byClient[clientKey(m.ConversationID(), *m.ClientMsgID())] = m
	}
	r.bySeq = append(r.bySeq, m)
	return nil
}

func (r *memMsgRepo) FindByID(_ context.Context, id int64) (*message.Message, error) {
	if m, ok := r.byID[id]; ok {
		return m, nil
	}
	return nil, message.ErrNotFound
}

func (r *memMsgRepo) FindByClientMsgID(_ context.Context, convID int64, cid string) (*message.Message, error) {
	if m, ok := r.byClient[clientKey(convID, cid)]; ok {
		return m, nil
	}
	return nil, message.ErrNotFound
}

func (r *memMsgRepo) Save(_ context.Context, m *message.Message) error {
	r.byID[m.ID()] = m
	return nil
}

func (r *memMsgRepo) ListAfterSeq(_ context.Context, convID, afterSeq int64, limit int) ([]*message.Message, error) {
	var out []*message.Message
	for _, m := range r.bySeq {
		if m.ConversationID() == convID && m.Seq() > afterSeq {
			out = append(out, m)
			if len(out) >= limit {
				break
			}
		}
	}
	return out, nil
}

// memSeqAlloc 内存 seq 分配器。
type memSeqAlloc struct{ n map[int64]int64 }

func (a *memSeqAlloc) Next(_ context.Context, convID int64) (int64, error) {
	if a.n == nil {
		a.n = map[int64]int64{}
	}
	a.n[convID]++
	return a.n[convID], nil
}

// capturePublisher 记录发布的事件。
type capturePublisher struct{ events []event.Event }

func (p *capturePublisher) Publish(_ context.Context, evts ...event.Event) error {
	p.events = append(p.events, evts...)
	return nil
}

func seedActiveMember(repo *memConvRepo, convID, userID int64) {
	conv, _ := conversation.NewGroup(convID, 1, "g", userID)
	repo.Create(context.Background(), conv)
	repo.AddMember(context.Background(), conversation.NewMember(userID*1000, convID, conversation.MemberUser, userID, conversation.RoleOwner))
}

func newMsgService(convRepo *memConvRepo, msgRepo *memMsgRepo, pub *capturePublisher) *MessageService {
	return NewMessageService(msgRepo, convRepo, &memSeqAlloc{}, allowChecker{}, pub, &fakeIDGen{}, directTxRunner{}, 2*time.Minute, 50, slog.Default())
}

func textCmd(convID, sender int64, cid, text string) messagingcmd.SendMessageCommand {
	return messagingcmd.SendMessageCommand{
		WorkspaceID: 1, ConversationID: convID, SenderUserID: sender,
		ClientMsgID: cid, ContentType: 1, Content: map[string]any{"text": text},
	}
}

func TestSendMessageAssignsSeqAndPublishes(t *testing.T) {
	convRepo := newMemConvRepo()
	msgRepo := newMemMsgRepo()
	pub := &capturePublisher{}
	seedActiveMember(convRepo, 100, 9)
	svc := newMsgService(convRepo, msgRepo, pub)

	first, err := svc.SendMessage(context.Background(), textCmd(100, 9, "c1", "hi"))
	if err != nil {
		t.Fatalf("发送失败: %v", err)
	}
	if first.Seq != 1 {
		t.Fatalf("首条 seq 应为 1, got %d", first.Seq)
	}
	second, err := svc.SendMessage(context.Background(), textCmd(100, 9, "c2", "yo"))
	if err != nil {
		t.Fatalf("发送失败: %v", err)
	}
	if second.Seq != 2 {
		t.Fatalf("第二条 seq 应为 2, got %d", second.Seq)
	}
	if len(pub.events) != 2 {
		t.Fatalf("应发布 2 个事件, got %d", len(pub.events))
	}
	// 会话游标推进。
	conv, _ := convRepo.FindByID(context.Background(), 100)
	if conv.LastSeq() != 2 {
		t.Fatalf("会话 last_seq 应为 2, got %d", conv.LastSeq())
	}
}

func TestSendMessageIdempotent(t *testing.T) {
	convRepo := newMemConvRepo()
	msgRepo := newMemMsgRepo()
	pub := &capturePublisher{}
	seedActiveMember(convRepo, 100, 9)
	svc := newMsgService(convRepo, msgRepo, pub)

	first, _ := svc.SendMessage(context.Background(), textCmd(100, 9, "dup", "hi"))
	second, err := svc.SendMessage(context.Background(), textCmd(100, 9, "dup", "hi-again"))
	if err != nil {
		t.Fatalf("幂等重发失败: %v", err)
	}
	if first.MessageID != second.MessageID || first.Seq != second.Seq {
		t.Fatalf("重复 client_msg_id 应返回原消息: %+v vs %+v", first, second)
	}
	if len(pub.events) != 1 {
		t.Fatalf("幂等命中不应重复发事件, got %d", len(pub.events))
	}
}

func TestSendMessageNonMemberForbidden(t *testing.T) {
	convRepo := newMemConvRepo()
	msgRepo := newMemMsgRepo()
	seedActiveMember(convRepo, 100, 9)
	svc := newMsgService(convRepo, msgRepo, &capturePublisher{})

	_, err := svc.SendMessage(context.Background(), textCmd(100, 999, "c1", "hi"))
	if err == nil {
		t.Fatal("非成员发送应被拒绝")
	}
}

func TestRecallSelfWritesSystemEntry(t *testing.T) {
	convRepo := newMemConvRepo()
	msgRepo := newMemMsgRepo()
	pub := &capturePublisher{}
	seedActiveMember(convRepo, 100, 9)
	svc := newMsgService(convRepo, msgRepo, pub)

	sent, _ := svc.SendMessage(context.Background(), textCmd(100, 9, "c1", "hi"))
	if err := svc.RecallMessage(context.Background(), messagingcmd.RecallMessageCommand{
		WorkspaceID: 1, ConversationID: 100, OperatorUserID: 9, MessageID: sent.MessageID,
	}); err != nil {
		t.Fatalf("撤回失败: %v", err)
	}
	recalled, _ := msgRepo.FindByID(context.Background(), sent.MessageID)
	if recalled.Status() != message.StatusRecalled {
		t.Fatalf("原消息应为已撤回, got %d", recalled.Status())
	}
	// 应写入一条系统撤回条目（seq=2）。
	conv, _ := convRepo.FindByID(context.Background(), 100)
	if conv.LastSeq() != 2 {
		t.Fatalf("撤回应占用新 seq, last_seq 应为 2, got %d", conv.LastSeq())
	}
	if len(pub.events) != 2 {
		t.Fatalf("发送 + 撤回应共 2 个事件, got %d", len(pub.events))
	}
}

func TestReportReadMonotonicAndEvent(t *testing.T) {
	convRepo := newMemConvRepo()
	msgRepo := newMemMsgRepo()
	pub := &capturePublisher{}
	seedActiveMember(convRepo, 100, 9)
	svc := newMsgService(convRepo, msgRepo, pub)

	// 推进到 5。
	if err := svc.ReportRead(context.Background(), messagingcmd.ReportReadCommand{
		WorkspaceID: 1, ConversationID: 100, UserID: 9, ReadSeq: 5,
	}); err != nil {
		t.Fatalf("上报失败: %v", err)
	}
	mem, _ := convRepo.FindMember(context.Background(), 100, conversation.MemberUser, 9)
	if mem.ReadSeq() != 5 {
		t.Fatalf("read_seq 应为 5, got %d", mem.ReadSeq())
	}
	if len(pub.events) != 1 {
		t.Fatalf("推进应发 1 事件, got %d", len(pub.events))
	}
	// 回退到 3 应幂等无变更、不发事件。
	if err := svc.ReportRead(context.Background(), messagingcmd.ReportReadCommand{
		WorkspaceID: 1, ConversationID: 100, UserID: 9, ReadSeq: 3,
	}); err != nil {
		t.Fatalf("回退上报应幂等: %v", err)
	}
	mem, _ = convRepo.FindMember(context.Background(), 100, conversation.MemberUser, 9)
	if mem.ReadSeq() != 5 {
		t.Fatalf("read_seq 不应回退, got %d", mem.ReadSeq())
	}
	if len(pub.events) != 1 {
		t.Fatalf("回退不应发事件, got %d", len(pub.events))
	}
}

func TestRecallExpiredWindow(t *testing.T) {
	convRepo := newMemConvRepo()
	msgRepo := newMemMsgRepo()
	seedActiveMember(convRepo, 100, 9)
	// 撤回窗口 0，必定超窗。
	svc := NewMessageService(msgRepo, convRepo, &memSeqAlloc{}, allowChecker{}, &capturePublisher{}, &fakeIDGen{}, directTxRunner{}, 0, 50, slog.Default())

	sent, _ := svc.SendMessage(context.Background(), textCmd(100, 9, "c1", "hi"))
	err := svc.RecallMessage(context.Background(), messagingcmd.RecallMessageCommand{
		WorkspaceID: 1, ConversationID: 100, OperatorUserID: 9, MessageID: sent.MessageID,
	})
	if err == nil {
		t.Fatal("超窗撤回应失败")
	}
}
