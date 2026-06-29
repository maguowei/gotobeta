package service

import (
	"context"

	messagingquery "github.com/maguowei/gotobeta/internal/modules/messaging/application/query"
	messagingresult "github.com/maguowei/gotobeta/internal/modules/messaging/application/result"
)

// PullMessages 增量拉取会话内 (afterSeq, +∞) 区间消息，需为该会话活跃成员。
func (s *MessageService) PullMessages(ctx context.Context, q messagingquery.PullMessagesQuery) ([]*messagingresult.MessageResult, error) {
	if _, err := s.requireActiveMember(ctx, q.ConversationID, q.OperatorUserID); err != nil {
		return nil, err
	}
	limit := q.Limit
	if limit <= 0 {
		limit = s.pageSize
	}
	if limit > s.maxPageSize {
		limit = s.maxPageSize
	}
	msgs, err := s.messages.ListAfterSeq(ctx, q.ConversationID, q.AfterSeq, limit)
	if err != nil {
		return nil, wrapInfrastructureError("拉取消息失败", err)
	}
	items := make([]*messagingresult.MessageResult, 0, len(msgs))
	for _, m := range msgs {
		items = append(items, toMessageResult(m))
	}
	return items, nil
}
