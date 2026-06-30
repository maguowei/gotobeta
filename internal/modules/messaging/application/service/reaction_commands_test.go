package service

import (
	"context"
	"fmt"
	"log/slog"
	"testing"
	"time"

	messagingcmd "github.com/maguowei/gotobeta/internal/modules/messaging/application/command"
	messagingquery "github.com/maguowei/gotobeta/internal/modules/messaging/application/query"
	"github.com/maguowei/gotobeta/internal/modules/messaging/domain/reaction"
	"github.com/maguowei/gotobeta/internal/pkg/apperr"
	"github.com/maguowei/gotobeta/internal/pkg/authz"
)

// memReactionRepo 是内存版表情回应仓储。
type memReactionRepo struct {
	items map[string]*reaction.Reaction
}

func newMemReactionRepo() *memReactionRepo {
	return &memReactionRepo{items: map[string]*reaction.Reaction{}}
}

func reactionKey(messageID, userID int64, emoji string) string {
	return fmt.Sprintf("%d|%d|%s", messageID, userID, emoji)
}

func (r *memReactionRepo) Add(_ context.Context, rc *reaction.Reaction) error {
	k := reactionKey(rc.MessageID(), rc.UserID(), rc.Emoji())
	if _, ok := r.items[k]; ok {
		return reaction.ErrAlreadyExists
	}
	r.items[k] = rc
	return nil
}

func (r *memReactionRepo) Remove(_ context.Context, messageID, userID int64, emoji string) (bool, error) {
	k := reactionKey(messageID, userID, emoji)
	if _, ok := r.items[k]; !ok {
		return false, nil
	}
	delete(r.items, k)
	return true, nil
}

func (r *memReactionRepo) ListByMessageID(_ context.Context, messageID int64) ([]*reaction.Reaction, error) {
	out := make([]*reaction.Reaction, 0)
	for _, rc := range r.items {
		if rc.MessageID() == messageID {
			out = append(out, rc)
		}
	}
	return out, nil
}

// denyChecker 始终拒绝权限。
type denyChecker struct{}

func (denyChecker) Check(context.Context, authz.Request) error { return apperr.Forbidden("denied") }

// newReactionService 构造可注入自定义 reaction repo 与 checker 的服务，并预置一条消息。
func newReactionSvcWithMessage(t *testing.T, checker authz.Checker) (*MessageService, *memReactionRepo, *capturePublisher, int64) {
	t.Helper()
	convRepo := newMemConvRepo()
	msgRepo := newMemMsgRepo()
	reactionRepo := newMemReactionRepo()
	pub := &capturePublisher{}
	seedActiveMember(convRepo, 100, 9)
	svc := NewMessageService(msgRepo, convRepo, reactionRepo, &memSeqAlloc{}, checker, pub, &fakeIDGen{}, directTxRunner{}, 2*time.Minute, 50, slog.Default(), nil)
	sent, err := svc.SendMessage(context.Background(), textCmd(100, 9, "c1", "hi"))
	if err != nil {
		t.Fatalf("预置消息失败: %v", err)
	}
	return svc, reactionRepo, pub, sent.MessageID
}

func addCmd(convID, msgID, user int64, emoji string) messagingcmd.AddReactionCommand {
	return messagingcmd.AddReactionCommand{
		WorkspaceID: 1, ConversationID: convID, MessageID: msgID, OperatorUserID: user, Emoji: emoji,
	}
}

func TestAddReactionPublishesAndIdempotent(t *testing.T) {
	svc, repo, pub, msgID := newReactionSvcWithMessage(t, allowChecker{})
	base := len(pub.events) // 预置消息已发 1 个事件

	if err := svc.AddReaction(context.Background(), addCmd(100, msgID, 9, "👍")); err != nil {
		t.Fatalf("添加回应失败: %v", err)
	}
	if len(pub.events) != base+1 {
		t.Fatalf("添加回应应发 1 事件, got %d", len(pub.events)-base)
	}
	if _, ok := repo.items[reactionKey(msgID, 9, "👍")]; !ok {
		t.Fatal("回应应已落库")
	}
	// 重复添加同 emoji：幂等，不发事件。
	if err := svc.AddReaction(context.Background(), addCmd(100, msgID, 9, "👍")); err != nil {
		t.Fatalf("幂等添加失败: %v", err)
	}
	if len(pub.events) != base+1 {
		t.Fatalf("重复添加不应再发事件, got %d", len(pub.events)-base)
	}
}

func TestRemoveReaction(t *testing.T) {
	svc, _, pub, msgID := newReactionSvcWithMessage(t, allowChecker{})
	_ = svc.AddReaction(context.Background(), addCmd(100, msgID, 9, "👍"))
	base := len(pub.events)

	if err := svc.RemoveReaction(context.Background(), messagingcmd.RemoveReactionCommand{
		WorkspaceID: 1, ConversationID: 100, MessageID: msgID, OperatorUserID: 9, Emoji: "👍",
	}); err != nil {
		t.Fatalf("取消回应失败: %v", err)
	}
	if len(pub.events) != base+1 {
		t.Fatalf("取消回应应发 1 事件, got %d", len(pub.events)-base)
	}
	// 再次取消：不存在，幂等不发事件。
	if err := svc.RemoveReaction(context.Background(), messagingcmd.RemoveReactionCommand{
		WorkspaceID: 1, ConversationID: 100, MessageID: msgID, OperatorUserID: 9, Emoji: "👍",
	}); err != nil {
		t.Fatalf("重复取消应幂等: %v", err)
	}
	if len(pub.events) != base+1 {
		t.Fatalf("重复取消不应发事件, got %d", len(pub.events)-base)
	}
}

func TestAddReactionPermissionDenied(t *testing.T) {
	svc, _, _, msgID := newReactionSvcWithMessage(t, denyChecker{})
	if err := svc.AddReaction(context.Background(), addCmd(100, msgID, 9, "👍")); err == nil {
		t.Fatal("无 message.react 权限应被拒绝")
	}
}

func TestAddReactionNonMemberForbidden(t *testing.T) {
	svc, _, _, msgID := newReactionSvcWithMessage(t, allowChecker{})
	if err := svc.AddReaction(context.Background(), addCmd(100, msgID, 999, "👍")); err == nil {
		t.Fatal("非成员添加回应应被拒绝")
	}
}

func TestAddReactionMessageNotFound(t *testing.T) {
	svc, _, _, _ := newReactionSvcWithMessage(t, allowChecker{})
	if err := svc.AddReaction(context.Background(), addCmd(100, 999999, 9, "👍")); err == nil {
		t.Fatal("对不存在消息添加回应应被拒绝")
	}
}

func TestAddReactionMessageNotInConversation(t *testing.T) {
	svc, _, _, msgID := newReactionSvcWithMessage(t, allowChecker{})
	// 消息属于会话 100，却声称属于会话 200。
	if err := svc.AddReaction(context.Background(), addCmd(200, msgID, 9, "👍")); err == nil {
		t.Fatal("消息不属于该会话应被拒绝")
	}
}

func TestListReactions(t *testing.T) {
	svc, _, _, msgID := newReactionSvcWithMessage(t, allowChecker{})
	_ = svc.AddReaction(context.Background(), addCmd(100, msgID, 9, "👍"))

	list, err := svc.ListReactions(context.Background(), messagingquery.ListReactionsQuery{
		WorkspaceID: 1, ConversationID: 100, MessageID: msgID, OperatorUserID: 9,
	})
	if err != nil {
		t.Fatalf("列举回应失败: %v", err)
	}
	if len(list) != 1 || list[0].Emoji != "👍" || list[0].UserID != 9 {
		t.Fatalf("列举结果不符: %+v", list)
	}
}
