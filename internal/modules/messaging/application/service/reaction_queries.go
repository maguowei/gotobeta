package service

import (
	"context"

	messagingquery "github.com/maguowei/gotobeta/internal/modules/messaging/application/query"
	messagingresult "github.com/maguowei/gotobeta/internal/modules/messaging/application/result"
)

// ListReactions 列举消息的全部表情回应（只读，成员可见）。
func (s *MessageService) ListReactions(ctx context.Context, q messagingquery.ListReactionsQuery) ([]*messagingresult.ReactionResult, error) {
	if err := assertWorkspaceScope(ctx, q.WorkspaceID); err != nil {
		return nil, err
	}
	if err := s.assertMessageInConversation(ctx, q.MessageID, q.ConversationID); err != nil {
		return nil, err
	}
	if _, err := s.requireActiveMember(ctx, q.ConversationID, q.OperatorUserID); err != nil {
		return nil, err
	}

	list, err := s.reactions.ListByMessageID(ctx, q.MessageID)
	if err != nil {
		return nil, wrapInfrastructureError("查询表情回应失败", err)
	}
	out := make([]*messagingresult.ReactionResult, 0, len(list))
	for _, rc := range list {
		out = append(out, &messagingresult.ReactionResult{
			MessageID: rc.MessageID(),
			UserID:    rc.UserID(),
			Emoji:     rc.Emoji(),
		})
	}
	return out, nil
}
