package service

import (
	"context"
	stderrors "errors"

	messagingquery "github.com/maguowei/gotobeta/internal/modules/messaging/application/query"
	messagingresult "github.com/maguowei/gotobeta/internal/modules/messaging/application/result"
	"github.com/maguowei/gotobeta/internal/modules/messaging/domain/conversation"
)

// ListConversations 返回我加入的会话列表，含未读数，按最近消息时间倒序。
func (s *ConversationService) ListConversations(ctx context.Context, q messagingquery.ListConversationsQuery) ([]*messagingresult.ConversationResult, error) {
	convs, err := s.conversations.ListByMember(ctx, conversation.MemberUser, q.UserID)
	if err != nil {
		return nil, wrapInfrastructureError("查询会话列表失败", err)
	}
	items := make([]*messagingresult.ConversationResult, 0, len(convs))
	for _, conv := range convs {
		if conv.WorkspaceID() != q.WorkspaceID {
			continue
		}
		res := toConversationResult(conv)
		mem, err := s.conversations.FindMember(ctx, conv.ID(), conversation.MemberUser, q.UserID)
		if err == nil {
			res.ReadSeq = mem.ReadSeq()
			res.Unread = mem.Unread(conv.LastSeq())
		} else if !stderrors.Is(err, conversation.ErrMemberNotFound) {
			return nil, wrapInfrastructureError("查询会话成员失败", err)
		}
		items = append(items, res)
	}
	return items, nil
}

// ConversationUserIDs 返回会话内全部活跃用户成员的 userID，实现 imrt.MemberLookup。
func (s *ConversationService) ConversationUserIDs(ctx context.Context, conversationID int64) ([]int64, error) {
	members, err := s.conversations.ListMembers(ctx, conversationID)
	if err != nil {
		return nil, wrapInfrastructureError("查询会话成员失败", err)
	}
	ids := make([]int64, 0, len(members))
	for _, m := range members {
		if m.MemberType() == conversation.MemberUser && m.Status() == conversation.MemberActive {
			ids = append(ids, m.MemberID())
		}
	}
	return ids, nil
}

// UserConversationPeers 返回与该用户共享任一会话的其他活跃用户集合，实现 imrt.MemberLookup。
func (s *ConversationService) UserConversationPeers(ctx context.Context, userID int64) ([]int64, error) {
	convs, err := s.conversations.ListByMember(ctx, conversation.MemberUser, userID)
	if err != nil {
		return nil, wrapInfrastructureError("查询会话列表失败", err)
	}
	peers := make(map[int64]struct{})
	for _, conv := range convs {
		members, err := s.conversations.ListMembers(ctx, conv.ID())
		if err != nil {
			return nil, wrapInfrastructureError("查询会话成员失败", err)
		}
		for _, m := range members {
			if m.MemberType() == conversation.MemberUser && m.Status() == conversation.MemberActive && m.MemberID() != userID {
				peers[m.MemberID()] = struct{}{}
			}
		}
	}
	ids := make([]int64, 0, len(peers))
	for id := range peers {
		ids = append(ids, id)
	}
	return ids, nil
}

// ListMembers 返回会话成员列表；操作者需为该会话活跃成员。
func (s *ConversationService) ListMembers(ctx context.Context, q messagingquery.ListMembersQuery) ([]*messagingresult.ConversationMemberResult, error) {
	if _, err := s.requireActiveMembership(ctx, q.ConversationID, q.OperatorUserID); err != nil {
		return nil, err
	}
	members, err := s.conversations.ListMembers(ctx, q.ConversationID)
	if err != nil {
		return nil, wrapInfrastructureError("查询会话成员失败", err)
	}
	items := make([]*messagingresult.ConversationMemberResult, 0, len(members))
	for _, m := range members {
		items = append(items, toConversationMemberResult(m))
	}
	return items, nil
}
