package conversation

import "context"

// Repository 定义会话聚合的仓储接口。
type Repository interface {
	Create(ctx context.Context, c *Conversation) error
	FindByID(ctx context.Context, id int64) (*Conversation, error)
	// FindByDMKey 按单聊去重键查找，未命中返回 ErrNotFound。
	FindByDMKey(ctx context.Context, dmKey string) (*Conversation, error)
	Save(ctx context.Context, c *Conversation) error

	AddMember(ctx context.Context, m *Member) error
	FindMember(ctx context.Context, conversationID int64, memberType MemberType, memberID int64) (*Member, error)
	SaveMember(ctx context.Context, m *Member) error
	ListMembers(ctx context.Context, conversationID int64) ([]*Member, error)
	// ListByMember 返回某主体加入的全部会话（含其成员视图），按 last_msg_at 倒序。
	ListByMember(ctx context.Context, memberType MemberType, memberID int64) ([]*Conversation, error)
}
