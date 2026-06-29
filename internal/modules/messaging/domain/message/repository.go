package message

import "context"

// Repository 定义消息聚合的仓储接口。
type Repository interface {
	Create(ctx context.Context, m *Message) error
	FindByID(ctx context.Context, id int64) (*Message, error)
	// FindByClientMsgID 按会话内幂等键查找，未命中返回 ErrNotFound。
	FindByClientMsgID(ctx context.Context, conversationID int64, clientMsgID string) (*Message, error)
	Save(ctx context.Context, m *Message) error
	// ListAfterSeq 返回会话内 (afterSeq, +∞) 区间、按 seq 升序的消息，最多 limit 条。
	ListAfterSeq(ctx context.Context, conversationID, afterSeq int64, limit int) ([]*Message, error)
}
