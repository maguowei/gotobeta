package service

import (
	"context"
	"errors"
	"log/slog"
	"sync/atomic"
	"testing"

	messagingcmd "github.com/maguowei/gotobeta/internal/modules/messaging/application/command"
	"github.com/maguowei/gotobeta/internal/modules/messaging/domain/conversation"
	"github.com/maguowei/gotobeta/internal/pkg/apperr"
	"github.com/maguowei/gotobeta/internal/pkg/authz"
)

type allowChecker struct{}

func (allowChecker) Check(context.Context, authz.Request) error { return nil }

type fakeIDGen struct{ n int64 }

func (g *fakeIDGen) NextID(context.Context) (int64, error) { return atomic.AddInt64(&g.n, 1), nil }

type directTxRunner struct{}

func (directTxRunner) RunInTx(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

// memConvRepo 是内存版会话仓储，仅供应用层用例测试。
type memConvRepo struct {
	convs   map[int64]*conversation.Conversation
	byDMKey map[string]*conversation.Conversation
	members map[int64][]*conversation.Member
}

func newMemConvRepo() *memConvRepo {
	return &memConvRepo{
		convs:   map[int64]*conversation.Conversation{},
		byDMKey: map[string]*conversation.Conversation{},
		members: map[int64][]*conversation.Member{},
	}
}

func (r *memConvRepo) Create(_ context.Context, c *conversation.Conversation) error {
	r.convs[c.ID()] = c
	if c.DMKey() != nil {
		if _, ok := r.byDMKey[*c.DMKey()]; ok {
			return conversation.ErrDMExists
		}
		r.byDMKey[*c.DMKey()] = c
	}
	return nil
}

func (r *memConvRepo) FindByID(_ context.Context, id int64) (*conversation.Conversation, error) {
	if c, ok := r.convs[id]; ok {
		return c, nil
	}
	return nil, conversation.ErrNotFound
}

func (r *memConvRepo) FindByDMKey(_ context.Context, dmKey string) (*conversation.Conversation, error) {
	if c, ok := r.byDMKey[dmKey]; ok {
		return c, nil
	}
	return nil, conversation.ErrNotFound
}

func (r *memConvRepo) Save(_ context.Context, c *conversation.Conversation) error {
	r.convs[c.ID()] = c
	return nil
}

func (r *memConvRepo) AddMember(_ context.Context, m *conversation.Member) error {
	r.members[m.ConversationID()] = append(r.members[m.ConversationID()], m)
	return nil
}

func (r *memConvRepo) FindMember(_ context.Context, convID int64, mt conversation.MemberType, mid int64) (*conversation.Member, error) {
	for _, m := range r.members[convID] {
		if m.MemberType() == mt && m.MemberID() == mid {
			return m, nil
		}
	}
	return nil, conversation.ErrMemberNotFound
}

func (r *memConvRepo) SaveMember(context.Context, *conversation.Member) error { return nil }

func (r *memConvRepo) ListMembers(_ context.Context, convID int64) ([]*conversation.Member, error) {
	return r.members[convID], nil
}

func (r *memConvRepo) ListByMember(_ context.Context, mt conversation.MemberType, mid int64) ([]*conversation.Conversation, error) {
	var out []*conversation.Conversation
	for convID, members := range r.members {
		for _, m := range members {
			if m.MemberType() == mt && m.MemberID() == mid {
				out = append(out, r.convs[convID])
				break
			}
		}
	}
	return out, nil
}

func newConvService(repo conversation.Repository, checker authz.Checker) *ConversationService {
	return NewConversationService(repo, checker, &fakeIDGen{}, directTxRunner{}, slog.Default())
}

func TestCreateDMDedup(t *testing.T) {
	repo := newMemConvRepo()
	svc := newConvService(repo, allowChecker{})
	cmd := messagingcmd.CreateConversationCommand{WorkspaceID: 1, OperatorUserID: 10, Type: 1, PeerUserID: 20}

	first, err := svc.CreateConversation(context.Background(), cmd)
	if err != nil {
		t.Fatalf("首次创建单聊失败: %v", err)
	}
	// 对端发起方向相反，应命中同一 dm_key。
	second, err := svc.CreateConversation(context.Background(), messagingcmd.CreateConversationCommand{
		WorkspaceID: 1, OperatorUserID: 20, Type: 1, PeerUserID: 10,
	})
	if err != nil {
		t.Fatalf("二次创建单聊失败: %v", err)
	}
	if first.ID != second.ID {
		t.Fatalf("单聊应去重为同一会话: %d != %d", first.ID, second.ID)
	}
	if len(repo.convs) != 1 {
		t.Fatalf("应只创建 1 个会话，实际 %d", len(repo.convs))
	}
}

func TestCreateConversationInvalidType(t *testing.T) {
	svc := newConvService(newMemConvRepo(), allowChecker{})
	_, err := svc.CreateConversation(context.Background(), messagingcmd.CreateConversationCommand{
		WorkspaceID: 1, OperatorUserID: 10, Type: 9,
	})
	if err == nil {
		t.Fatal("非法会话类型应报错")
	}
}

func TestAddMemberRequiresManager(t *testing.T) {
	repo := newMemConvRepo()
	svc := newConvService(repo, allowChecker{})
	// 建一个群，operator 100 为 owner。
	created, err := svc.CreateConversation(context.Background(), messagingcmd.CreateConversationCommand{
		WorkspaceID: 1, OperatorUserID: 100, Type: 2, Name: "g",
	})
	if err != nil {
		t.Fatalf("建群失败: %v", err)
	}
	// 加一个普通成员 200。
	if _, err := svc.AddMember(context.Background(), messagingcmd.AddMemberCommand{
		WorkspaceID: 1, OperatorUserID: 100, ConversationID: created.ID, MemberType: 1, MemberID: 200, Role: 3,
	}); err != nil {
		t.Fatalf("owner 加人失败: %v", err)
	}
	// 普通成员 200 试图加人 300 → Forbidden。
	_, err = svc.AddMember(context.Background(), messagingcmd.AddMemberCommand{
		WorkspaceID: 1, OperatorUserID: 200, ConversationID: created.ID, MemberType: 1, MemberID: 300, Role: 3,
	})
	if err == nil {
		t.Fatal("非管理员加人应被拒绝")
	}
	var de *apperr.DomainError
	if !errors.As(err, &de) {
		t.Fatalf("应为 DomainError，得 %v", err)
	}
}

func TestRemoveMemberSelf(t *testing.T) {
	repo := newMemConvRepo()
	svc := newConvService(repo, allowChecker{})
	created, _ := svc.CreateConversation(context.Background(), messagingcmd.CreateConversationCommand{
		WorkspaceID: 1, OperatorUserID: 100, Type: 2, Name: "g",
	})
	if _, err := svc.AddMember(context.Background(), messagingcmd.AddMemberCommand{
		WorkspaceID: 1, OperatorUserID: 100, ConversationID: created.ID, MemberType: 1, MemberID: 200, Role: 3,
	}); err != nil {
		t.Fatalf("加人失败: %v", err)
	}
	// 成员 200 移除自己应允许。
	if err := svc.RemoveMember(context.Background(), messagingcmd.RemoveMemberCommand{
		WorkspaceID: 1, OperatorUserID: 200, ConversationID: created.ID, MemberType: 1, MemberID: 200,
	}); err != nil {
		t.Fatalf("移除自己应成功: %v", err)
	}
}
